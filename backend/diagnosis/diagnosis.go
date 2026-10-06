package diagnosis

import (
	"fmt"
	"strings"
	"time"

	"crashlens/classifier"
	"crashlens/db"
	"crashlens/fireworks"
)

type Report struct {
	FailureType     string    `json:"failure_type"`
	Confidence      float64   `json:"confidence"`
	RootCause       string    `json:"root_cause"`
	Evidence        []string  `json:"evidence"`
	RecommendedFix  string    `json:"recommended_fix"`
	SafeToRetry     bool      `json:"safe_to_retry"`
	DiagnosedAt     time.Time `json:"diagnosed_at"`
	ConfidenceBasis string    `json:"confidence_basis"`
	Prevention      string    `json:"prevention,omitempty"`
	Source          string    `json:"source"` // "ai" or "rules"
}

func RunDiagnosis(workload *db.Workload, fwClient *fireworks.Client) Report {
	// Classify the failure type with confidence
	var classResult classifier.ClassificationResult
	var failureType string

	// Confidence describes a heuristic log match, not measured AI accuracy.
	logs, gpuMetrics := "", ""
	if workload.JobLogs != nil {
		logs = *workload.JobLogs
	}
	if workload.GPUMetrics != nil {
		gpuMetrics = *workload.GPUMetrics
	}
	classResult = classifier.ClassifyWithConfidence(logs, gpuMetrics)
	failureType = classResult.FailureType
	if workload.FailureType != nil && *workload.FailureType == classifier.Timeout && strings.Contains(logs, "Job execution timeout:") {
		failureType = classifier.Timeout
		classResult.Confidence = .92
	}
	if failureType == classifier.UnknownError && workload.FailureType != nil {
		failureType = *workload.FailureType // Preserve externally supplied labels without inflating confidence.
	}

	// Extract evidence from logs
	evidence := []string{}
	if workload.JobLogs != nil {
		evidence = classifier.ExtractEvidenceFromLogs(*workload.JobLogs, 5)
	}

	// Try AI diagnosis if Fireworks client is available
	if fwClient != nil && workload.JobLogs != nil {
		aiReport := callAI(fwClient, workload, failureType, evidence, classResult.Confidence)
		if aiReport != nil {
			return *aiReport
		}
	}

	// Fallback to rule-based diagnosis
	return ruleBasedDiagnosis(failureType, evidence, classResult.Confidence)
}

func callAI(fwClient *fireworks.Client, workload *db.Workload, failureType string, evidence []string, confidence float64) *Report {
	// Build comprehensive workload data for AI diagnosis
	runtimeStr := "unknown"
	if workload.RuntimeSeconds != nil {
		runtimeStr = fmt.Sprintf("%.2f", *workload.RuntimeSeconds)
	}

	workloadData := fmt.Sprintf(`Workload Information:
- Name: %s
- Type: %s
- Status: %s
- Failure Type (detected): %s
- Runtime: %s seconds
`, workload.Name, workload.Type, workload.Status, failureType, runtimeStr)

	// Add logs evidence (limit to last 10KB to prevent token overflow)
	if workload.JobLogs != nil {
		logs := *workload.JobLogs
		maxLogSize := 10 * 1024 // 10KB
		if len(logs) > maxLogSize {
			logs = "...(truncated)...\n" + logs[len(logs)-maxLogSize:]
		}
		workloadData += fmt.Sprintf("\nJob Logs:\n%s\n", logs)
	}

	// Add GPU metrics (limit to last 5KB)
	if workload.GPUMetrics != nil {
		metrics := *workload.GPUMetrics
		maxMetricsSize := 5 * 1024 // 5KB
		if len(metrics) > maxMetricsSize {
			metrics = "...(truncated)...\n" + metrics[len(metrics)-maxMetricsSize:]
		}
		workloadData += fmt.Sprintf("\nGPU Metrics:\n%s\n", metrics)
	}

	// Call Fireworks AI with function-calling for structured output
	result, err := fwClient.DiagnoseFailure(workloadData)
	if err != nil || result == nil {
		return nil
	}

	// Convert DiagnosisResult to Report format
	report := Report{
		FailureType:     failureType,
		Confidence:      confidence,
		RootCause:       result.RootCause,
		Evidence:        result.Evidence,
		RecommendedFix:  formatRecommendedFixes(result.RecommendedFixes),
		SafeToRetry:     result.SafeToRetry,
		DiagnosedAt:     time.Now(),
		Source:          "ai",
		ConfidenceBasis: "heuristic_log_match",
		Prevention:      result.Prevention,
	}

	return &report
}

func formatRecommendedFixes(fixes []string) string {
	result := ""
	for i, fix := range fixes {
		result += fmt.Sprintf("%d. %s\n", i+1, fix)
	}
	return result
}

func ruleBasedDiagnosis(failureType string, evidence []string, confidence float64) Report {
	report := Report{
		FailureType:     failureType,
		Confidence:      confidence,
		Evidence:        evidence,
		DiagnosedAt:     time.Now(),
		Source:          "rules",
		ConfidenceBasis: "heuristic_log_match",
	}

	switch failureType {
	case classifier.GPUOutOfMemory:
		report.RootCause = "GPU Out of Memory (OOM) - The model or batch size exceeded available GPU memory"
		report.RecommendedFix = `1. Reduce batch size in training configuration
2. Enable gradient checkpointing to save memory
3. Use mixed precision training (FP16/BF16)
4. Consider using a smaller model variant
5. Increase GPU memory by using larger GPU instances (A100, H100, MI250X, etc.)
6. Use gradient accumulation instead of larger batches`
		report.SafeToRetry = false

	case classifier.MissingCheckpoint:
		report.RootCause = "Missing checkpoint file - The training job expected to resume from a checkpoint that doesn't exist"
		report.RecommendedFix = `1. Verify checkpoint path in configuration
2. Check if checkpoint was properly saved in previous run
3. Ensure checkpoint directory has correct permissions
4. If starting fresh, remove checkpoint resume flag from config`
		report.SafeToRetry = false

	case classifier.DependencyError:
		report.RootCause = "Python dependency or import error - Required packages are missing or incompatible"
		report.RecommendedFix = `1. Run 'pip install -r requirements.txt' to install dependencies
2. Check Python version compatibility
3. For NVIDIA: Install CUDA-compatible PyTorch (pytorch.org)
4. For AMD: Install ROCm-compatible PyTorch with ROCm index URL
5. Check for conflicting package versions
6. Use 'pip list' to verify installed packages`
		report.SafeToRetry = false

	case classifier.DataPathError:
		report.RootCause = "Data file or path not found - Training data is inaccessible"
		report.RecommendedFix = `1. Verify data path in configuration file
2. Check if data was downloaded/preprocessed correctly
3. Ensure data directory has correct permissions
4. Verify S3 bucket or remote storage credentials if applicable
5. Check for typos in file paths`
		report.SafeToRetry = false

	case classifier.Timeout:
		report.RootCause = "Job timeout - The workload exceeded the maximum allowed runtime"
		report.RecommendedFix = `1. Increase timeout limit in job configuration
2. Optimize training loop for faster iterations
3. Reduce number of training epochs
4. Profile code to identify bottlenecks
5. Consider using faster GPU instances (A100, H100, MI250X)
6. Check for data loading bottlenecks`
		report.SafeToRetry = false

	case classifier.ROCmError:
		report.RootCause = "AMD ROCm runtime error - HIP/ROCm encountered a GPU-related error"
		report.RecommendedFix = `1. Verify ROCm installation: 'rocm-smi' command
2. Check ROCm version compatibility with PyTorch
3. Update ROCm drivers to latest version
4. Verify GPU is properly detected: 'rocm-smi --showid'
5. Check for kernel/driver conflicts
6. Review AMD GPU compatibility matrix`
		report.SafeToRetry = false

	case classifier.CUDAError:
		report.RootCause = "NVIDIA CUDA runtime error - CUDA/cuDNN encountered a GPU-related error"
		report.RecommendedFix = `1. Verify CUDA installation: 'nvidia-smi' command
2. Check CUDA version compatibility with PyTorch
3. Update NVIDIA drivers to latest version
4. Verify GPU is properly detected: 'nvidia-smi -L'
5. Check cuDNN library compatibility
6. Review NVIDIA GPU compute capability requirements`
		report.SafeToRetry = false

	case classifier.GPUDriverError:
		report.RootCause = "GPU driver version mismatch or driver not available"
		report.RecommendedFix = `1. Update GPU drivers (nvidia-smi for NVIDIA, rocm-smi for AMD)
2. Verify correct GPU runtime is installed (CUDA or ROCm)
3. Check driver compatibility with framework version
4. Restart system after driver updates
5. Verify GPU is accessible: 'nvidia-smi' or 'rocm-smi'`
		report.SafeToRetry = false

	default:
		report.RootCause = "Unknown error - Unable to automatically classify the failure"
		report.RecommendedFix = `1. Review full error logs for specific error messages
2. Check system resources (CPU, Memory, Disk)
3. Verify all dependencies are installed
4. Check GPU compatibility (nvidia-smi or rocm-smi)
5. Review job configuration for errors
6. Verify framework installation (PyTorch, TensorFlow, etc.)
7. Contact support with full logs if issue persists`
		report.SafeToRetry = false
	}

	return report
}

func formatEvidence(evidence []string) string {
	result := ""
	for i, line := range evidence {
		result += fmt.Sprintf("%d. %s\n", i+1, line)
	}
	if result == "" {
		result = "(No specific error lines extracted)"
	}
	return result
}
