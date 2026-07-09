package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type FireworksRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type FireworksResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func runDiagnosis(workload Workload) DiagnosisResponse {
	// First, classify the failure type if not already done
	failureType := ""
	if workload.FailureType != nil {
		failureType = *workload.FailureType
	} else if workload.JobLogs != nil {
		gpuMetrics := ""
		if workload.GPUMetrics != nil {
			gpuMetrics = *workload.GPUMetrics
		}
		failureType = ClassifyFailure(*workload.JobLogs, gpuMetrics)
	}

	// Extract evidence from logs
	evidence := []string{}
	if workload.JobLogs != nil {
		evidence = ExtractEvidenceFromLogs(*workload.JobLogs, 5)
	}

	// If we have Fireworks API key, use Gemma for advanced diagnosis
	apiKey := os.Getenv("FIREWORKS_API_KEY")
	if apiKey != "" && workload.JobLogs != nil {
		aiDiagnosis := callFireworksAI(apiKey, workload, failureType, evidence)
		if aiDiagnosis != nil {
			return *aiDiagnosis
		}
	}

	// Fallback to rule-based diagnosis
	return ruleBasedDiagnosis(failureType, evidence, workload)
}

func callFireworksAI(apiKey string, workload Workload, failureType string, evidence []string) *DiagnosisResponse {
	// Build context for AI
	context := fmt.Sprintf(`You are an AI expert in ML infrastructure and AMD ROCm GPU debugging.

Workload Information:
- Name: %s
- Type: %s
- Status: %s
- Failure Type (detected): %s

Logs Evidence:
%s

Task: Provide a structured diagnosis of this failure.

Respond in JSON format with these fields:
{
  "root_cause": "Brief description of the root cause",
  "evidence": ["list", "of", "key", "evidence", "lines"],
  "recommended_fix": "Specific actionable steps to fix this issue",
  "safe_to_retry": true/false
}
`, workload.Name, workload.Type, workload.Status, failureType, formatEvidence(evidence))

	if workload.GPUMetrics != nil {
		context += fmt.Sprintf("\nGPU Metrics:\n%s\n", *workload.GPUMetrics)
	}

	// Call Fireworks AI API
	reqBody := FireworksRequest{
		Model: "accounts/fireworks/models/gemma2-9b-it",
		Messages: []Message{
			{Role: "user", Content: context},
		},
		MaxTokens: 1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil
	}

	req, err := http.NewRequest("POST", "https://api.fireworks.ai/inference/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var fireworksResp FireworksResponse
	if err := json.Unmarshal(body, &fireworksResp); err != nil {
		return nil
	}

	if len(fireworksResp.Choices) == 0 {
		return nil
	}

	// Parse the AI response
	aiResponse := fireworksResp.Choices[0].Message.Content

	// Try to extract JSON from the response
	var diagnosis DiagnosisResponse

	// The response might be wrapped in markdown code blocks, so try to extract JSON
	jsonStart := bytes.Index([]byte(aiResponse), []byte("{"))
	jsonEnd := bytes.LastIndex([]byte(aiResponse), []byte("}"))

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := aiResponse[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &diagnosis); err == nil {
			diagnosis.DiagnosedAt = time.Now()
			return &diagnosis
		}
	}

	return nil
}

func ruleBasedDiagnosis(failureType string, evidence []string, workload Workload) DiagnosisResponse {
	diagnosis := DiagnosisResponse{
		Evidence:    evidence,
		DiagnosedAt: time.Now(),
	}

	switch failureType {
	case "gpu_oom":
		diagnosis.RootCause = "GPU Out of Memory (OOM) - The model or batch size exceeded available GPU memory"
		diagnosis.RecommendedFix = `1. Reduce batch size in training configuration
2. Enable gradient checkpointing to save memory
3. Use mixed precision training (FP16)
4. Consider using a smaller model variant
5. Increase GPU memory by using larger AMD GPU instances`
		diagnosis.SafeToRetry = false

	case "missing_checkpoint":
		diagnosis.RootCause = "Missing checkpoint file - The training job expected to resume from a checkpoint that doesn't exist"
		diagnosis.RecommendedFix = `1. Verify checkpoint path in configuration
2. Check if checkpoint was properly saved in previous run
3. Ensure checkpoint directory has correct permissions
4. If starting fresh, remove checkpoint resume flag from config`
		diagnosis.SafeToRetry = true

	case "dependency_error":
		diagnosis.RootCause = "Python dependency or import error - Required packages are missing or incompatible"
		diagnosis.RecommendedFix = `1. Run 'pip install -r requirements.txt' to install dependencies
2. Check Python version compatibility
3. Verify ROCm-compatible PyTorch is installed for AMD GPUs
4. Check for conflicting package versions
5. Use 'pip list' to verify installed packages`
		diagnosis.SafeToRetry = true

	case "data_path_error":
		diagnosis.RootCause = "Data file or path not found - Training data is inaccessible"
		diagnosis.RecommendedFix = `1. Verify data path in configuration file
2. Check if data was downloaded/preprocessed correctly
3. Ensure data directory has correct permissions
4. Verify S3 bucket or remote storage credentials if applicable
5. Check for typos in file paths`
		diagnosis.SafeToRetry = true

	case "timeout":
		diagnosis.RootCause = "Job timeout - The workload exceeded the maximum allowed runtime"
		diagnosis.RecommendedFix = `1. Increase timeout limit in job configuration
2. Optimize training loop for faster iterations
3. Reduce number of training epochs
4. Profile code to identify bottlenecks
5. Consider using faster AMD GPU instances`
		diagnosis.SafeToRetry = true

	case "rocm_error":
		diagnosis.RootCause = "AMD ROCm runtime error - HIP/ROCm encountered a GPU-related error"
		diagnosis.RecommendedFix = `1. Verify ROCm installation: 'rocm-smi' command
2. Check ROCm version compatibility with PyTorch
3. Update ROCm drivers to latest version
4. Verify GPU is properly detected: 'rocm-smi --showid'
5. Check for kernel/driver conflicts
6. Review AMD GPU compatibility matrix`
		diagnosis.SafeToRetry = true

	case "gpu_driver_error":
		diagnosis.RootCause = "GPU driver version mismatch or driver not available"
		diagnosis.RecommendedFix = `1. Update AMD GPU drivers
2. Verify ROCm is properly installed
3. Check driver compatibility with ROCm version
4. Restart system after driver updates
5. Verify GPU is accessible: 'rocm-smi' or 'rocminfo'`
		diagnosis.SafeToRetry = true

	default:
		diagnosis.RootCause = "Unknown error - Unable to automatically classify the failure"
		diagnosis.RecommendedFix = `1. Review full error logs for specific error messages
2. Check system resources (CPU, Memory, Disk)
3. Verify all dependencies are installed
4. Check for ROCm/AMD GPU compatibility issues
5. Review job configuration for errors
6. Contact support with full logs if issue persists`
		diagnosis.SafeToRetry = false
	}

	return diagnosis
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
