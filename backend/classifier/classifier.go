package classifier

import (
	"regexp"
	"strings"
)

// FailureType constants
const (
	GPUOutOfMemory   = "GPU_OUT_OF_MEMORY"
	MissingCheckpoint = "MISSING_CHECKPOINT"
	DependencyError  = "DEPENDENCY_ERROR"
	DataPathError    = "DATA_PATH_ERROR"
	Timeout          = "TIMEOUT"
	ROCmError        = "ROCM_ERROR"
	GPUDriverError   = "GPU_DRIVER_ERROR"
	UnknownError     = "UNKNOWN_ERROR"
)

// ClassificationResult contains the classification outcome with confidence
type ClassificationResult struct {
	FailureType string
	Confidence  float64
}

// ClassifyFailure analyzes logs and metrics to determine failure type
func ClassifyFailure(logs, gpuMetrics string) string {
	result := ClassifyWithConfidence(logs, gpuMetrics)
	return result.FailureType
}

// ClassifyWithConfidence analyzes logs and returns failure type with confidence score
func ClassifyWithConfidence(logs, gpuMetrics string) ClassificationResult {
	logsLower := strings.ToLower(logs)

	// GPU OOM detection - high confidence patterns
	if strings.Contains(logsLower, "hip out of memory") ||
		strings.Contains(logsLower, "hip error: out of memory") {
		return ClassificationResult{GPUOutOfMemory, 0.95}
	}
	if strings.Contains(logsLower, "out of memory") || strings.Contains(logsLower, "oom") {
		return ClassificationResult{GPUOutOfMemory, 0.85}
	}
	if strings.Contains(logsLower, "cudamalloc failed") ||
		strings.Contains(logsLower, "rocm out of memory") {
		return ClassificationResult{GPUOutOfMemory, 0.90}
	}

	// Missing checkpoint detection
	if strings.Contains(logsLower, "checkpoint not found") {
		return ClassificationResult{MissingCheckpoint, 0.95}
	}
	if strings.Contains(logsLower, "checkpoint") &&
		(strings.Contains(logsLower, "not found") ||
		 strings.Contains(logsLower, "missing") ||
		 strings.Contains(logsLower, "does not exist")) {
		return ClassificationResult{MissingCheckpoint, 0.85}
	}

	// Dependency error detection - high confidence
	if strings.Contains(logsLower, "modulenotfounderror") ||
		strings.Contains(logsLower, "importerror") {
		return ClassificationResult{DependencyError, 0.95}
	}
	if strings.Contains(logsLower, "no module named") ||
		strings.Contains(logsLower, "cannot import") {
		return ClassificationResult{DependencyError, 0.90}
	}
	if strings.Contains(logsLower, "rocm version mismatch") ||
		strings.Contains(logsLower, "rocm-compatible") && strings.Contains(logsLower, "not found") {
		return ClassificationResult{DependencyError, 0.92}
	}

	// Data path error detection
	if strings.Contains(logsLower, "dataset") &&
		(strings.Contains(logsLower, "not found") || strings.Contains(logsLower, "does not exist")) {
		return ClassificationResult{DataPathError, 0.90}
	}
	if strings.Contains(logsLower, "no such file or directory") {
		return ClassificationResult{DataPathError, 0.85}
	}
	if (strings.Contains(logsLower, "file") || strings.Contains(logsLower, "path")) &&
		(strings.Contains(logsLower, "not found") || strings.Contains(logsLower, "does not exist")) {
		return ClassificationResult{DataPathError, 0.75}
	}

	// Timeout detection
	if strings.Contains(logsLower, "no progress detected") {
		return ClassificationResult{Timeout, 0.90}
	}
	if strings.Contains(logsLower, "runtime exceeded") ||
		strings.Contains(logsLower, "execution timeout") {
		return ClassificationResult{Timeout, 0.92}
	}
	if strings.Contains(logsLower, "timeout") ||
		strings.Contains(logsLower, "timed out") ||
		strings.Contains(logsLower, "deadline exceeded") {
		return ClassificationResult{Timeout, 0.80}
	}

	// ROCm/HIP specific errors
	if strings.Contains(logsLower, "hip error") ||
		strings.Contains(logsLower, "rocm error") ||
		strings.Contains(logsLower, "hsa error") {
		return ClassificationResult{ROCmError, 0.88}
	}

	// CUDA compatibility issues (for mixed environments)
	if strings.Contains(logsLower, "cuda") &&
		(strings.Contains(logsLower, "not available") ||
		 strings.Contains(logsLower, "driver version")) {
		return ClassificationResult{GPUDriverError, 0.85}
	}

	return ClassificationResult{UnknownError, 0.50}
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
