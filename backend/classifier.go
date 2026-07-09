package main

import (
	"regexp"
	"strings"
)

// ClassifyFailure analyzes logs and metrics to determine failure type
func ClassifyFailure(logs, gpuMetrics string) string {
	logsLower := strings.ToLower(logs)

	// GPU OOM detection
	if strings.Contains(logsLower, "out of memory") ||
		strings.Contains(logsLower, "oom") ||
		strings.Contains(logsLower, "cudamalloc failed") ||
		strings.Contains(logsLower, "hip error: out of memory") ||
		strings.Contains(logsLower, "rocm out of memory") {
		return "gpu_oom"
	}

	// Missing checkpoint detection
	if strings.Contains(logsLower, "checkpoint") &&
		(strings.Contains(logsLower, "not found") ||
		 strings.Contains(logsLower, "missing") ||
		 strings.Contains(logsLower, "does not exist")) {
		return "missing_checkpoint"
	}

	// Dependency error detection
	if strings.Contains(logsLower, "importerror") ||
		strings.Contains(logsLower, "modulenotfounderror") ||
		strings.Contains(logsLower, "no module named") ||
		strings.Contains(logsLower, "cannot import") {
		return "dependency_error"
	}

	// Data path error detection
	if (strings.Contains(logsLower, "file") || strings.Contains(logsLower, "path")) &&
		(strings.Contains(logsLower, "not found") ||
		 strings.Contains(logsLower, "no such file") ||
		 strings.Contains(logsLower, "does not exist")) {
		return "data_path_error"
	}

	// Timeout detection
	if strings.Contains(logsLower, "timeout") ||
		strings.Contains(logsLower, "timed out") ||
		strings.Contains(logsLower, "deadline exceeded") {
		return "timeout"
	}

	// ROCm/HIP specific errors
	if strings.Contains(logsLower, "hip error") ||
		strings.Contains(logsLower, "rocm error") ||
		strings.Contains(logsLower, "hsa error") {
		return "rocm_error"
	}

	// CUDA compatibility issues (for mixed environments)
	if strings.Contains(logsLower, "cuda") &&
		(strings.Contains(logsLower, "not available") ||
		 strings.Contains(logsLower, "driver version")) {
		return "gpu_driver_error"
	}

	// Generic GPU error
	if strings.Contains(logsLower, "gpu") && strings.Contains(logsLower, "error") {
		return "gpu_error"
	}

	return "unknown_error"
}

// ExtractEvidenceFromLogs extracts relevant error lines from logs
func ExtractEvidenceFromLogs(logs string, maxLines int) []string {
	evidence := []string{}
	lines := strings.Split(logs, "\n")

	// Error patterns to look for
	errorPatterns := []string{
		"error",
		"exception",
		"failed",
		"traceback",
		"fatal",
		"critical",
		"out of memory",
		"not found",
		"hip error",
		"rocm",
	}

	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, pattern := range errorPatterns {
			if strings.Contains(lineLower, pattern) {
				// Clean up the line
				cleaned := strings.TrimSpace(line)
				if cleaned != "" && len(cleaned) < 500 {
					evidence = append(evidence, cleaned)
					if len(evidence) >= maxLines {
						return evidence
					}
					break
				}
			}
		}
	}

	return evidence
}

// ExtractGPUMemoryUsage extracts memory usage from GPU metrics
func ExtractGPUMemoryUsage(gpuMetrics string) *GPUMemoryInfo {
	if gpuMetrics == "" {
		return nil
	}

	// Parse GPU memory usage from rocm-smi or nvidia-smi output
	info := &GPUMemoryInfo{}

	// Look for memory patterns like "Memory Used: 15360 MiB / 16384 MiB"
	memPattern := regexp.MustCompile(`(\d+)\s*(?:MiB|MB|GiB|GB).*?/\s*(\d+)\s*(?:MiB|MB|GiB|GB)`)
	matches := memPattern.FindStringSubmatch(gpuMetrics)

	if len(matches) >= 3 {
		info.Used = matches[1]
		info.Total = matches[2]
	}

	return info
}

type GPUMemoryInfo struct {
	Used  string
	Total string
}

// CalculateWastedGPUSeconds calculates GPU time wasted on failed job
func CalculateWastedGPUSeconds(runtimeSeconds float64, numGPUs int) float64 {
	if numGPUs <= 0 {
		numGPUs = 1 // Default to 1 GPU if not specified
	}
	return runtimeSeconds * float64(numGPUs)
}
