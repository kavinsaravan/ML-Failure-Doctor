package agent_diagnosis

import (
	"crashlens/db"
	"encoding/json"
	"strings"
)

// DetectAgentLoops detects if an agent is stuck in a loop
// Rule: If the same tool is called 3+ times with the same or very similar input
func DetectAgentLoops(steps []db.AgentStep) (bool, string) {
	toolCalls := make(map[string]int)
	var loopDetails []string

	for _, step := range steps {
		if step.StepType == "TOOL_CALL" && step.Input != nil {
			key := step.Name + ":" + *step.Input
			toolCalls[key]++

			if toolCalls[key] >= 3 {
				loopDetails = append(loopDetails, step.Name+" called "+string(rune(toolCalls[key]))+" times with same input")
			}
		}
	}

	if len(loopDetails) > 0 {
		return true, "Agent detected in infinite loop: " + strings.Join(loopDetails, "; ")
	}

	return false, ""
}

// DetectToolCallFailure detects if any tool call failed
// Rule: If any TOOL_CALL step has status = "failed" or "error"
func DetectToolCallFailure(steps []db.AgentStep) (bool, string) {
	var failedTools []string

	for _, step := range steps {
		if step.StepType == "TOOL_CALL" && (step.Status == "failed" || step.Status == "error") {
			toolInfo := step.Name
			if step.Output != nil {
				toolInfo += ": " + *step.Output
			}
			failedTools = append(failedTools, toolInfo)
		}
	}

	if len(failedTools) > 0 {
		return true, "Tool call(s) failed: " + strings.Join(failedTools, "; ")
	}

	return false, ""
}

// DetectModelTimeout detects if any model call took too long
// Rule: If any MODEL_CALL latency > 10000ms (10 seconds)
func DetectModelTimeout(steps []db.AgentStep, thresholdMs int) (bool, string) {
	if thresholdMs == 0 {
		thresholdMs = 10000 // Default 10 seconds
	}

	var timeoutCalls []string

	for _, step := range steps {
		if step.StepType == "MODEL_CALL" && step.LatencyMS != nil && *step.LatencyMS > thresholdMs {
			timeoutCalls = append(timeoutCalls, step.Name+" took "+string(rune(*step.LatencyMS))+"ms")
		}
	}

	if len(timeoutCalls) > 0 {
		return true, "Model timeout detected: " + strings.Join(timeoutCalls, "; ")
	}

	return false, ""
}

// ClassifyAgentFailure runs all detection rules and returns the failure type
func ClassifyAgentFailure(steps []db.AgentStep) (string, string) {
	// Check for explicit ERROR steps first
	for _, step := range steps {
		if step.StepType == "ERROR" {
			// Agent already reported a specific failure type
			return step.Name, *step.Output
		}
	}

	// Check for AGENT_LOOP
	if detected, message := DetectAgentLoops(steps); detected {
		return "AGENT_LOOP", message
	}

	// Check for TOOL_CALL_FAILURE
	if detected, message := DetectToolCallFailure(steps); detected {
		return "TOOL_CALL_FAILURE", message
	}

	// Check for MODEL_TIMEOUT
	if detected, message := DetectModelTimeout(steps, 10000); detected {
		return "MODEL_TIMEOUT", message
	}

	return "UNKNOWN", "Agent failed but no specific failure pattern detected"
}

// DiagnosisInput contains all data needed for agent diagnosis
type DiagnosisInput struct {
	AgentName   string          `json:"agent_name"`
	Task        string          `json:"task"`
	FailureType string          `json:"failure_type"`
	Steps       []db.AgentStep  `json:"steps"`
	TotalToolCalls   int        `json:"total_tool_calls"`
	TotalModelCalls  int        `json:"total_model_calls"`
	TotalTokens      int        `json:"total_tokens"`
	TotalLatencyMS   int        `json:"total_latency_ms"`
}

// FormatStepsForDiagnosis formats agent steps into a readable trace
func FormatStepsForDiagnosis(steps []db.AgentStep) string {
	var formatted strings.Builder

	for i, step := range steps {
		formatted.WriteString("Step ")
		formatted.WriteString(string(rune(i + 1)))
		formatted.WriteString(": [")
		formatted.WriteString(step.StepType)
		formatted.WriteString("] ")
		formatted.WriteString(step.Name)
		formatted.WriteString("\n")

		if step.Input != nil && *step.Input != "" {
			formatted.WriteString("  Input: ")
			formatted.WriteString(*step.Input)
			formatted.WriteString("\n")
		}

		if step.Output != nil && *step.Output != "" {
			formatted.WriteString("  Output: ")
			formatted.WriteString(*step.Output)
			formatted.WriteString("\n")
		}

		formatted.WriteString("  Status: ")
		formatted.WriteString(step.Status)

		if step.LatencyMS != nil {
			formatted.WriteString(" (")
			formatted.WriteString(string(rune(*step.LatencyMS)))
			formatted.WriteString("ms)")
		}
		formatted.WriteString("\n\n")
	}

	return formatted.String()
}

// BuildAgentDiagnosisPrompt creates the prompt for Gemma to diagnose agent failures
func BuildAgentDiagnosisPrompt(input DiagnosisInput) string {
	var prompt strings.Builder

	prompt.WriteString("You are an AI agent observability assistant.\n\n")
	prompt.WriteString("Given this agent execution trace, generate:\n")
	prompt.WriteString("1. Root cause\n")
	prompt.WriteString("2. Evidence from tool calls and model calls\n")
	prompt.WriteString("3. Recommended fixes\n")
	prompt.WriteString("4. Prevention advice\n\n")
	prompt.WriteString("Use only the provided trace data. Do not invent facts.\n\n")

	prompt.WriteString("Agent Name: ")
	prompt.WriteString(input.AgentName)
	prompt.WriteString("\n\n")

	prompt.WriteString("Task: ")
	prompt.WriteString(input.Task)
	prompt.WriteString("\n\n")

	prompt.WriteString("Failure Type: ")
	prompt.WriteString(input.FailureType)
	prompt.WriteString("\n\n")

	prompt.WriteString("Execution Stats:\n")
	prompt.WriteString("- Total tool calls: ")
	prompt.WriteString(string(rune(input.TotalToolCalls)))
	prompt.WriteString("\n- Total model calls: ")
	prompt.WriteString(string(rune(input.TotalModelCalls)))
	prompt.WriteString("\n- Total tokens: ")
	prompt.WriteString(string(rune(input.TotalTokens)))
	prompt.WriteString("\n- Total latency: ")
	prompt.WriteString(string(rune(input.TotalLatencyMS)))
	prompt.WriteString("ms\n\n")

	prompt.WriteString("Trace:\n")
	prompt.WriteString(FormatStepsForDiagnosis(input.Steps))

	return prompt.String()
}

// AgentDiagnosisReport is the structured output from Gemma
type AgentDiagnosisReport struct {
	RootCause        string   `json:"root_cause"`
	Evidence         []string `json:"evidence"`
	RecommendedFixes []string `json:"recommended_fixes"`
	Prevention       string   `json:"prevention"`
}

// ParseDiagnosisResponse parses Gemma's response into a structured report
func ParseDiagnosisResponse(response string) (*AgentDiagnosisReport, error) {
	var report AgentDiagnosisReport
	err := json.Unmarshal([]byte(response), &report)
	if err != nil {
		return nil, err
	}
	return &report, nil
}
