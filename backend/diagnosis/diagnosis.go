package diagnosis

import (
	"fmt"
	"time"

	"crashlens/classifier"
	"crashlens/db"
	"crashlens/fireworks"
)

type Report struct {
	FailureType    string    `json:"failure_type"`
	Confidence     float64   `json:"confidence"`
	RootCause      string    `json:"root_cause"`
	Evidence       []string  `json:"evidence"`
	RecommendedFix string    `json:"recommended_fix"`
	SafeToRetry    bool      `json:"safe_to_retry"`
	DiagnosedAt    time.Time `json:"diagnosed_at"`
}

func RunDiagnosis(workload *db.Workload, fwClient *fireworks.Client) Report {
	// Classify the failure type with confidence
	var classResult classifier.ClassificationResult
	var failureType string

	if workload.FailureType != nil {
		failureType = *workload.FailureType
		classResult = classifier.ClassificationResult{
			FailureType: failureType,
			Confidence:  0.95, // High confidence if already classified
		}
	} else if workload.JobLogs != nil {
		gpuMetrics := ""
		if workload.GPUMetrics != nil {
			gpuMetrics = *workload.GPUMetrics
		}
		classResult = classifier.ClassifyWithConfidence(*workload.JobLogs, gpuMetrics)
		failureType = classResult.FailureType
	} else {
		classResult = classifier.ClassificationResult{
			FailureType: classifier.UnknownError,
			Confidence:  0.50,
		}
		failureType = classifier.UnknownError
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
	workloadData := fmt.Sprintf(`Workload Information:
- Name: %s
- Type: %s
- Status: %s
- Failure Type (detected): %s
- Runtime: %v seconds
`, workload.Name, workload.Type, workload.Status, failureType, workload.RuntimeSeconds)

	// Add logs evidence
	if workload.JobLogs != nil {
		workloadData += fmt.Sprintf("\nJob Logs:\n%s\n", *workload.JobLogs)
	}

	// Add GPU metrics
	if workload.GPUMetrics != nil {
		workloadData += fmt.Sprintf("\nGPU Metrics:\n%s\n", *workload.GPUMetrics)
	}

	// Call Fireworks AI with function-calling for structured output
	result, err := fwClient.DiagnoseFailure(workloadData)
	if err != nil || result == nil {
		return nil
	}

	// Convert DiagnosisResult to Report format
	report := Report{
		FailureType:    failureType,
		Confidence:     confidence,
		RootCause:      result.RootCause,
		Evidence:       result.Evidence,
		RecommendedFix: formatRecommendedFixes(result.RecommendedFixes),
		SafeToRetry:    result.SafeToRetry,
		DiagnosedAt:    time.Now(),
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
		FailureType: failureType,
		Confidence:  confidence,
		Evidence:    evidence,
		DiagnosedAt: time.Now(),
	}

	switch failureType {
	case classifier.GPUOutOfMemory:
		report.RootCause = "GPU Out of Memory (OOM) - The model or batch size exceeded available GPU memory"
		report.RecommendedFix = `1. Reduce batch size in training configuration
2. Enable gradient checkpointing to save memory
3. Use mixed precision training (FP16)
4. Consider using a smaller model variant
5. Increase GPU memory by using larger AMD GPU instances`
		report.SafeToRetry = false

	case classifier.MissingCheckpoint:
		report.RootCause = "Missing checkpoint file - The training job expected to resume from a checkpoint that doesn't exist"
		report.RecommendedFix = `1. Verify checkpoint path in configuration
2. Check if checkpoint was properly saved in previous run
3. Ensure checkpoint directory has correct permissions
4. If starting fresh, remove checkpoint resume flag from config`
		report.SafeToRetry = true

	case classifier.DependencyError:
		report.RootCause = "Python dependency or import error - Required packages are missing or incompatible"
		report.RecommendedFix = `1. Run 'pip install -r requirements.txt' to install dependencies
2. Check Python version compatibility
3. Verify ROCm-compatible PyTorch is installed for AMD GPUs
4. Check for conflicting package versions
5. Use 'pip list' to verify installed packages`
		report.SafeToRetry = true

	case classifier.DataPathError:
		report.RootCause = "Data file or path not found - Training data is inaccessible"
		report.RecommendedFix = `1. Verify data path in configuration file
2. Check if data was downloaded/preprocessed correctly
3. Ensure data directory has correct permissions
4. Verify S3 bucket or remote storage credentials if applicable
5. Check for typos in file paths`
		report.SafeToRetry = true

	case classifier.Timeout:
		report.RootCause = "Job timeout - The workload exceeded the maximum allowed runtime"
		report.RecommendedFix = `1. Increase timeout limit in job configuration
2. Optimize training loop for faster iterations
3. Reduce number of training epochs
4. Profile code to identify bottlenecks
5. Consider using faster AMD GPU instances`
		report.SafeToRetry = true

	case classifier.ROCmError:
		report.RootCause = "AMD ROCm runtime error - HIP/ROCm encountered a GPU-related error"
		report.RecommendedFix = `1. Verify ROCm installation: 'rocm-smi' command
2. Check ROCm version compatibility with PyTorch
3. Update ROCm drivers to latest version
4. Verify GPU is properly detected: 'rocm-smi --showid'
5. Check for kernel/driver conflicts
6. Review AMD GPU compatibility matrix`
		report.SafeToRetry = true

	case classifier.GPUDriverError:
		report.RootCause = "GPU driver version mismatch or driver not available"
		report.RecommendedFix = `1. Update AMD GPU drivers
2. Verify ROCm is properly installed
3. Check driver compatibility with ROCm version
4. Restart system after driver updates
5. Verify GPU is accessible: 'rocm-smi' or 'rocminfo'`
		report.SafeToRetry = true

	default:
		report.RootCause = "Unknown error - Unable to automatically classify the failure"
		report.RecommendedFix = `1. Review full error logs for specific error messages
2. Check system resources (CPU, Memory, Disk)
3. Verify all dependencies are installed
4. Check for ROCm/AMD GPU compatibility issues
5. Review job configuration for errors
6. Contact support with full logs if issue persists`
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
