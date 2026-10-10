package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"crashlens/classifier"
	"crashlens/db"
	"crashlens/metrics"
)

type JobResult struct {
	ExitCode       int
	Logs           string
	GPUMetrics     string
	RuntimeSeconds float64
	Error          error
}

type MetricsSnapshot struct {
	Source           string    `json:"source"`
	Timestamp        time.Time `json:"timestamp"`
	GPUMemoryUsed    float64   `json:"gpu_memory_used_mb"`
	GPUMemoryTotal   float64   `json:"gpu_memory_total_mb"`
	GPUMemoryPercent float64   `json:"gpu_memory_percent"`
	GPUUtilization   float64   `json:"gpu_utilization_percent"`
	Temperature      float64   `json:"temperature_celsius"`
}

// RunPythonJob executes a Python script with streaming logs and metrics collection
func RunPythonJob(scriptPath string, workloadID int, database *db.DB) (*JobResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return RunPythonJobContext(ctx, scriptPath, workloadID, database)
}

func RunPythonJobContext(ctx context.Context, scriptPath string, workloadID int, database *db.DB) (*JobResult, error) {
	startTime := time.Now()

	// Update workload status to running
	workload, err := database.GetWorkload(fmt.Sprintf("%d", workloadID))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	workload.StartedAt = &now
	workload.Status = "running"
	if err := database.UpdateWorkload(fmt.Sprintf("%d", workloadID), workload); err != nil {
		return nil, err
	}

	failStart := func(cause error) (*JobResult, error) {
		finished := time.Now()
		runtime := finished.Sub(startTime).Seconds()
		code := 1
		logs := "Job launch failed: " + cause.Error()
		failure := classifier.UnknownError
		workload.Status = "failed"
		workload.FinishedAt = &finished
		workload.RuntimeSeconds = &runtime
		workload.ExitCode = &code
		workload.JobLogs = &logs
		workload.FailureType = &failure
		if err := database.UpdateWorkload(fmt.Sprintf("%d", workloadID), workload); err != nil {
			return nil, fmt.Errorf("%v; save failure: %w", cause, err)
		}
		return nil, cause
	}
	// Unbuffered Python output makes progress available before process exit.
	cmd := exec.CommandContext(ctx, "python3", "-u", scriptPath)
	configureCancellation(cmd)
	var logBuffer bytes.Buffer
	var logMutex sync.Mutex
	output := &lockedLogWriter{buffer: &logBuffer, mutex: &logMutex}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		return failStart(err)
	}

	// Start metrics collection in background
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	allMetrics := []MetricsSnapshot{}
	var metricsMutex sync.Mutex
	metricsDone := make(chan struct{})
	go func() {
		defer close(metricsDone)
		collectMetricsToSlice(metricsCtx, scriptPath, &allMetrics, &metricsMutex)
	}()

	// Persist current logs and samples while the process is running.
	liveCtx, stopLive := context.WithCancel(context.Background())
	liveDone := make(chan struct{})
	go func() {
		defer close(liveDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-liveCtx.Done():
				return
			case <-ticker.C:
				logMutex.Lock()
				logs := logBuffer.String()
				logMutex.Unlock()
				metricsMutex.Lock()
				samples := append([]MetricsSnapshot{}, allMetrics...)
				metricsMutex.Unlock()
				encoded, _ := json.Marshal(samples)
				if err := database.UpdateTelemetry(workloadID, logs, string(encoded), time.Since(startTime).Seconds()); err != nil {
					log.Printf("Live telemetry for job %d: %v", workloadID, err)
				}
			}
		}
	}()
	err = cmd.Wait()
	runtime := time.Since(startTime).Seconds()
	cancelMetrics()
	stopLive()
	<-metricsDone
	<-liveDone
	if ctx.Err() != nil {
		text := "Job canceled: " + ctx.Err().Error()
		if ctx.Err() == context.DeadlineExceeded {
			text = "Job execution timeout: " + ctx.Err().Error()
		}
		output.Write([]byte("\n" + text + "\n"))
	}

	// Format metrics as JSON
	metricsJSON, _ := json.MarshalIndent(allMetrics, "", "  ")

	// Get final logs
	logs := logBuffer.String()

	result := &JobResult{
		Logs:           logs,
		GPUMetrics:     string(metricsJSON),
		RuntimeSeconds: runtime,
	}

	// Get exit code
	if err != nil || ctx.Err() != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Error = err
		}
	} else {
		result.ExitCode = 0
	}

	// Update workload with results
	finishTime := time.Now()
	workload.FinishedAt = &finishTime
	workload.RuntimeSeconds = &runtime
	workload.ExitCode = &result.ExitCode
	workload.JobLogs = &logs

	if result.GPUMetrics != "" {
		workload.GPUMetrics = &result.GPUMetrics
	}

	// Determine status based on exit code
	if result.ExitCode != 0 {
		workload.Status = "failed"
		failureType := classifier.ClassifyFailure(logs, result.GPUMetrics)
		if ctx.Err() == context.DeadlineExceeded {
			failureType = classifier.Timeout
		}
		workload.FailureType = &failureType

		// Calculate wasted GPU seconds (assuming 1 GPU for now)
		wastedGPU := classifier.CalculateWastedGPUSeconds(runtime, 1)
		workload.WastedGPUSeconds = &wastedGPU
	} else {
		workload.Status = "succeeded"
	}

	if err := database.UpdateWorkload(fmt.Sprintf("%d", workloadID), workload); err != nil {
		return nil, err
	}

	log.Printf("Job %d completed: status=%s, exit_code=%d, runtime=%.2fs",
		workloadID, workload.Status, result.ExitCode, runtime)

	return result, nil
}

// Keep a bounded tail of logs while allowing concurrent stdout/stderr writes.
type lockedLogWriter struct {
	buffer *bytes.Buffer
	mutex  *sync.Mutex
}

func (w *lockedLogWriter) Write(chunk []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.buffer.Write(chunk)
	if w.buffer.Len() > 256*1024 {
		tail := append([]byte{}, w.buffer.Bytes()[w.buffer.Len()-256*1024:]...)
		w.buffer.Reset()
		w.buffer.Write(tail)
	}
	return len(chunk), nil
}

// collectMetricsToSlice periodically collects GPU metrics during job execution
func collectMetricsToSlice(ctx context.Context, scriptPath string, samples *[]MetricsSnapshot, mutex *sync.Mutex) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Determine job type from script path for simulation
	jobType := getJobType(scriptPath)
	collector := metrics.GetCollector(jobType)
	fallback := metrics.NewSimulatedCollector(jobType)
	if ctx.Err() == nil {
		snapshot := snapshotFromCollector(collector, fallback)
		mutex.Lock()
		*samples = append(*samples, snapshot)
		mutex.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := snapshotFromCollector(collector, fallback)
			mutex.Lock()
			*samples = append(*samples, snapshot)
			if len(*samples) > 300 {
				*samples = append([]MetricsSnapshot{}, (*samples)[len(*samples)-300:]...)
			}
			mutex.Unlock()
		}
	}
}

// getJobType extracts job type from script path for metric simulation
func getJobType(scriptPath string) string {
	if strings.Contains(scriptPath, "gpu_oom") {
		return "gpu_oom"
	} else if strings.Contains(scriptPath, "timeout") {
		return "timeout"
	} else if strings.Contains(scriptPath, "successful") {
		return "successful"
	}
	return "default"
}

// Adapt metrics from the shared collector implementation to the runner payload.
func snapshotFromCollector(collector metrics.MetricCollector, fallback *metrics.SimulatedCollector) MetricsSnapshot {
	sample, err := collector.Collect()
	if err != nil {
		sample, _ = fallback.Collect()
	}
	return MetricsSnapshot{Timestamp: sample.Timestamp, GPUMemoryUsed: sample.GPUMemoryUsedMB,
		GPUMemoryTotal: sample.GPUMemoryTotalMB, GPUMemoryPercent: sample.GPUMemoryPercent,
		GPUUtilization: sample.GPUUtilizationPercent, Temperature: float64(sample.TemperatureCelsius), Source: sample.Source}
}
