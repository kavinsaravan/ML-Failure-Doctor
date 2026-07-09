package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"crashlens/backend/classifier"
	"crashlens/backend/db"
)

type JobResult struct {
	ExitCode       int
	Logs           string
	GPUMetrics     string
	RuntimeSeconds float64
	Error          error
}

// RunPythonJob executes a Python script and captures output
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

	// Execute Python script
	cmd := exec.Command("python3", scriptPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	runtime := time.Since(startTime).Seconds()

	// Combine stdout and stderr
	logs := stdout.String() + "\n" + stderr.String()

	result := &JobResult{
		Logs:           logs,
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

	// Collect GPU metrics if available
	result.GPUMetrics = collectGPUMetrics()

	// Update workload with results
	finishTime := time.Now()
	workload.FinishedAt = &finishTime
	workload.RuntimeSeconds = &runtime
	workload.ExitCode = &result.ExitCode
	workload.JobLogs = &logs

	if result.GPUMetrics != "" {
		workload.GPUMetrics = &result.GPUMetrics
	}

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

	return result, nil
}

func collectGPUMetrics() string {
	// Try rocm-smi first (AMD GPUs)
	cmd := exec.Command("rocm-smi", "--showmeminfo", "vram")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output)
	}

	// Fallback to nvidia-smi (NVIDIA GPUs)
	cmd = exec.Command("nvidia-smi", "--query-gpu=memory.used,memory.total", "--format=csv,noheader")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output)
	}

	return ""
}
