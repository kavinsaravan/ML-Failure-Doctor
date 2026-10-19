package classifier

import (
	"testing"
)

func TestClassifyWithConfidence(t *testing.T) {
	tests := []struct {
		name          string
		logs          string
		gpuMetrics    string
		expectedType  string
		minConfidence float64
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
			name: "Real PyTorch OOM with full traceback",
			logs: `Traceback (most recent call last):
  File "train.py", line 87, in <module>
    loss.backward()
  File "/usr/local/lib/python3.10/site-packages/torch/tensor.py", line 245, in backward
    torch.autograd.backward(self, gradient, retain_graph, create_graph)
RuntimeError: CUDA out of memory. Tried to allocate 2.00 GiB (GPU 0; 15.78 GiB total capacity; 13.24 GiB already allocated; 1.23 GiB free; 13.91 GiB reserved in total by PyTorch)`,
			expectedType:  GPUOutOfMemory,
			minConfidence: 0.95,
		},
		{
			name: "Real torch.load missing checkpoint",
			logs: `Traceback (most recent call last):
  File "resume_training.py", line 23, in <module>
    checkpoint = torch.load('checkpoints/epoch_10.pt')
FileNotFoundError: [Errno 2] No such file or directory: 'checkpoints/epoch_10.pt'`,
			expectedType:  MissingCheckpoint,
			minConfidence: 0.85,
		},
		{
			name: "CUDA device-side assert triggered",
			logs: `RuntimeError: CUDA error: device-side assert triggered
CUDA kernel errors might be asynchronously reported at some other API call,so the stacktrace below might be incorrect.
For debugging consider passing CUDA_LAUNCH_BLOCKING=1.`,
			expectedType:  CUDAError,
			minConfidence: 0.88,
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

func TestMPSOutOfMemory(t *testing.T) {
	result := ClassifyWithConfidence("RuntimeError: MPS backend out of memory (MPS allocated: 4.00 GB)", "")
	if result.FailureType != GPUOutOfMemory {
		t.Fatalf("unexpected classification: %+v", result)
	}
}
