package classifier

import (
	"testing"
)

func TestClassifyWithConfidence(t *testing.T) {
	tests := []struct {
		name              string
		logs              string
		gpuMetrics        string
		expectedType      string
		minConfidence     float64
	}{
		{
			name:          "CUDA OOM",
			logs:          "RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB",
			expectedType:  GPUOutOfMemory,
			minConfidence: 0.95,
		},
		{
			name:          "HIP OOM",
			logs:          "RuntimeError: HIP out of memory. Allocation failed",
			expectedType:  GPUOutOfMemory,
			minConfidence: 0.95,
		},
		{
			name:          "ROCm OOM",
			logs:          "Error: rocm out of memory during tensor allocation",
			expectedType:  GPUOutOfMemory,
			minConfidence: 0.90,
		},
		{
			name:          "Generic OOM with word boundary",
			logs:          "Fatal error: oom during training",
			expectedType:  GPUOutOfMemory,
			minConfidence: 0.85,
		},
		{
			name:          "False positive - room should not match",
			logs:          "Error in conference room setup",
			expectedType:  UnknownError,
			minConfidence: 0.50,
		},
		{
			name:          "Missing checkpoint",
			logs:          "FileNotFoundError: checkpoint not found at /path/to/model.pt",
			expectedType:  MissingCheckpoint,
			minConfidence: 0.95,
		},
		{
			name:          "Dependency error - ModuleNotFoundError",
			logs:          "ModuleNotFoundError: No module named 'torch'",
			expectedType:  DependencyError,
			minConfidence: 0.95,
		},
		{
			name:          "Dependency error - ImportError",
			logs:          "ImportError: cannot import name 'cuda' from 'torch'",
			expectedType:  DependencyError,
			minConfidence: 0.90,
		},
		{
			name:          "Data path error",
			logs:          "FileNotFoundError: dataset not found at /data/train.csv",
			expectedType:  DataPathError,
			minConfidence: 0.90,
		},
		{
			name:          "Timeout",
			logs:          "Error: execution timeout after 3600 seconds",
			expectedType:  Timeout,
			minConfidence: 0.92,
		},
		{
			name:          "ROCm error",
			logs:          "HIP error: hipErrorLaunchFailure during kernel execution",
			expectedType:  ROCmError,
			minConfidence: 0.88,
		},
		{
			name:          "CUDA error",
			logs:          "CUDA runtime error: invalid configuration argument",
			expectedType:  CUDAError,
			minConfidence: 0.88,
		},
		{
			name:          "GPU driver error",
			logs:          "Error: CUDA driver version is insufficient for runtime",
			expectedType:  GPUDriverError,
			minConfidence: 0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyWithConfidence(tt.logs, tt.gpuMetrics)

			if result.FailureType != tt.expectedType {
				t.Errorf("Expected failure type %s, got %s", tt.expectedType, result.FailureType)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
			}
		})
	}
}

func TestExtractEvidenceFromLogs(t *testing.T) {
	logs := `Starting training...
Epoch 1/10
Error: CUDA out of memory
Traceback (most recent call last):
  File "train.py", line 42
RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB`

	evidence := ExtractEvidenceFromLogs(logs, 3)

	if len(evidence) == 0 {
		t.Fatal("Expected evidence to be extracted")
	}

	// Check that error lines are included
	found := false
	for _, line := range evidence {
		if len(line) > 0 && len(line) < 500 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected evidence lines to be non-empty and under 500 chars")
	}
}

func TestCalculateWastedGPUSeconds(t *testing.T) {
	tests := []struct {
		name           string
		runtimeSeconds float64
		numGPUs        int
		expected       float64
	}{
		{"Single GPU", 100.0, 1, 100.0},
		{"Two GPUs", 100.0, 2, 200.0},
		{"Zero GPUs defaults to 1", 100.0, 0, 100.0},
		{"Negative GPUs defaults to 1", 100.0, -1, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateWastedGPUSeconds(tt.runtimeSeconds, tt.numGPUs)
			if result != tt.expected {
				t.Errorf("Expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}
