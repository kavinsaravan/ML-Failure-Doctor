package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"crashlens/classifier"
	"crashlens/db"
)

type JobResult struct {
	ExitCode       int
	Logs           string
	GPUMetrics     string
	RuntimeSeconds float64
	Error          error
}

type MetricsSnapshot struct {
	Timestamp        time.Time `json:"timestamp"`
	GPUMemoryUsed    float64   `json:"gpu_memory_used_mb"`
	GPUMemoryTotal   float64   `json:"gpu_memory_total_mb"`
	GPUMemoryPercent float64   `json:"gpu_memory_percent"`
	GPUUtilization   float64   `json:"gpu_utilization_percent"`
	Temperature      float64   `json:"temperature_celsius"`
}

// RunPythonJob executes a Python script with streaming logs and metrics collection
func RunPythonJob(scriptPath string, workloadID int, database *db.DB) (*JobResult, error) {
	startTime := time.Now()

	// Update workload status to running
	workload, err := database.GetWorkload(fmt.Sprintf("%d", workloadID))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	workload.StartedAt = &now
	workload.Status = "running"
	database.UpdateWorkload(fmt.Sprintf("%d", workloadID), workload)

	// Prepare command
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "python3", scriptPath)

	// Create pipes for streaming stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Buffer to collect all logs
	var logBuffer bytes.Buffer

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Start metrics collection in background
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	metricsChan := make(chan MetricsSnapshot, 100)
	go collectMetricsStream(metricsCtx, scriptPath, metricsChan)

	// Stream stdout
	go streamOutput(stdout, &logBuffer, "STDOUT")

	// Stream stderr
	go streamOutput(stderr, &logBuffer, "STDERR")

	// Wait for command to finish
	err = cmd.Wait()
	runtime := time.Since(startTime).Seconds()

	// Stop metrics collection
	cancelMetrics()
	close(metricsChan)

	// Collect all metrics snapshots
	var allMetrics []MetricsSnapshot
	for metric := range metricsChan {
		allMetrics = append(allMetrics, metric)
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
	if err != nil {
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
		workload.FailureType = &failureType

		// Calculate wasted GPU seconds (assuming 1 GPU for now)
		wastedGPU := classifier.CalculateWastedGPUSeconds(runtime, 1)
		workload.WastedGPUSeconds = &wastedGPU
	} else {
		workload.Status = "succeeded"
	}

	database.UpdateWorkload(fmt.Sprintf("%d", workloadID), workload)

	log.Printf("Job %d completed: status=%s, exit_code=%d, runtime=%.2fs",
		workloadID, workload.Status, result.ExitCode, runtime)

	return result, nil
}

// streamOutput reads from a pipe and writes to buffer with optional logging
func streamOutput(reader io.Reader, buffer *bytes.Buffer, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line + "\n")
		// Optionally log to console for debugging
		// log.Printf("[%s] %s", prefix, line)
	}
}

// collectMetricsStream periodically collects GPU metrics during job execution
func collectMetricsStream(ctx context.Context, scriptPath string, metricsChan chan<- MetricsSnapshot) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Determine job type from script path for simulation
	jobType := getJobType(scriptPath)
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := collectGPUMetricsSnapshot(jobType, iteration)
			metricsChan <- snapshot
			iteration++
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

// collectGPUMetricsSnapshot collects a single snapshot of GPU metrics
func collectGPUMetricsSnapshot(jobType string, iteration int) MetricsSnapshot {
	// Try to collect real AMD GPU metrics first
	if realMetrics := collectRealROCmMetrics(); realMetrics != nil {
		return *realMetrics
	}

	// Fallback to simulated metrics
	return simulateGPUMetrics(jobType, iteration)
}

// collectRealROCmMetrics attempts to collect real AMD GPU metrics using rocm-smi
func collectRealROCmMetrics() *MetricsSnapshot {
	// Try rocm-smi --showmemuse --showuse
	cmd := exec.Command("rocm-smi", "--showmemuse", "--showuse", "--showtemp")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse rocm-smi output
	lines := strings.Split(string(output), "\n")
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
	}

	// Simple parsing - looking for patterns like:
	// GPU[0] : Memory Usage: 12345 / 24576 MB
	// GPU[0] : GPU use: 85%
	for _, line := range lines {
		if strings.Contains(line, "Memory Usage") {
			// Extract memory values
			fmt.Sscanf(line, "GPU[0] : Memory Usage: %f / %f MB",
				&snapshot.GPUMemoryUsed, &snapshot.GPUMemoryTotal)
			if snapshot.GPUMemoryTotal > 0 {
				snapshot.GPUMemoryPercent = (snapshot.GPUMemoryUsed / snapshot.GPUMemoryTotal) * 100
			}
		} else if strings.Contains(line, "GPU use") {
			fmt.Sscanf(line, "GPU[0] : GPU use: %f%%", &snapshot.GPUUtilization)
		} else if strings.Contains(line, "Temperature") {
			fmt.Sscanf(line, "GPU[0] : Temperature: %f C", &snapshot.Temperature)
		}
	}

	// If we got valid data, return it
	if snapshot.GPUMemoryTotal > 0 {
		return snapshot
	}

	return nil
}

// simulateGPUMetrics generates realistic GPU metrics for demo purposes
func simulateGPUMetrics(jobType string, iteration int) MetricsSnapshot {
	snapshot := MetricsSnapshot{
		Timestamp:      time.Now(),
		GPUMemoryTotal: 24576, // 24GB AMD GPU
		Temperature:    65.0 + float64(iteration)*2.0,
	}

	switch jobType {
	case "gpu_oom":
		// Simulate gradual memory increase leading to OOM
		progress := []float64{20, 45, 70, 85, 91, 95, 98, 99}
		utilization := []float64{35, 62, 89, 95, 98, 99, 99, 99}

		idx := iteration
		if idx >= len(progress) {
			idx = len(progress) - 1
		}

		snapshot.GPUMemoryPercent = progress[idx]
		snapshot.GPUUtilization = utilization[idx]
		snapshot.GPUMemoryUsed = (snapshot.GPUMemoryPercent / 100.0) * snapshot.GPUMemoryTotal

	case "timeout":
		// Simulate stalled metrics
		snapshot.GPUMemoryPercent = 45.0
		snapshot.GPUUtilization = 5.0 + rand.Float64()*3.0 // Low, fluctuating
		snapshot.GPUMemoryUsed = (snapshot.GPUMemoryPercent / 100.0) * snapshot.GPUMemoryTotal

	case "successful":
		// Simulate normal training with moderate resource usage
		baseMemory := 40.0 + float64(iteration%3)*5.0
		baseUtil := 70.0 + rand.Float64()*20.0

		snapshot.GPUMemoryPercent = baseMemory
		snapshot.GPUUtilization = baseUtil
		snapshot.GPUMemoryUsed = (snapshot.GPUMemoryPercent / 100.0) * snapshot.GPUMemoryTotal

	default:
		// Default metrics for other job types
		snapshot.GPUMemoryPercent = 30.0 + rand.Float64()*40.0
		snapshot.GPUUtilization = 50.0 + rand.Float64()*30.0
		snapshot.GPUMemoryUsed = (snapshot.GPUMemoryPercent / 100.0) * snapshot.GPUMemoryTotal
	}

	return snapshot
}
