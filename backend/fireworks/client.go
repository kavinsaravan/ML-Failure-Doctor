package fireworks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

type FunctionParameter struct {
	Type        string                       `json:"type"`
	Properties  map[string]PropertySchema    `json:"properties,omitempty"`
	Items       *PropertySchema              `json:"items,omitempty"`
	Description string                       `json:"description,omitempty"`
	Required    []string                     `json:"required,omitempty"`
}

type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

type Function struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  FunctionParameter  `json:"parameters"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type ToolChoice struct {
	Type     string               `json:"type"`
	Function FunctionCallChoice   `json:"function"`
}

type FunctionCallChoice struct {
	Name string `json:"name"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  *ToolChoice `json:"tool_choice,omitempty"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function FunctionCallResult   `json:"function"`
}

type FunctionCallResult struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Response struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type DiagnosisResult struct {
	RootCause        string   `json:"root_cause"`
	Evidence         []string `json:"evidence"`
	RecommendedFixes []string `json:"recommended_fixes"`
	SafeToRetry      bool     `json:"safe_to_retry"`
	Prevention       string   `json:"prevention"`
}

type Client struct {
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewClient() *Client {
	apiKey := os.Getenv("FIREWORKS_API_KEY")
	if apiKey == "" {
		return nil
	}

	// Use environment variable for model, fallback to deployment
	model := os.Getenv("FIREWORKS_MODEL")
	if model == "" {
		model = "accounts/kavinsaravan-hhm94d1/deployments/gas0qxbe"
	}

	return &Client{
		APIKey:  apiKey,
		BaseURL: "https://api.fireworks.ai/inference/v1/chat/completions",
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) DiagnoseFailure(workloadData string) (*DiagnosisResult, error) {
	if c == nil {
		return nil, nil
	}

	// Define the function schema for structured diagnosis output
	diagnosisFunction := Tool{
		Type: "function",
		Function: Function{
			Name:        "diagnose_ml_failure",
			Description: "Diagnose ML workload failure and provide structured analysis",
			Parameters: FunctionParameter{
				Type: "object",
				Properties: map[string]PropertySchema{
					"root_cause": {
						Type:        "string",
						Description: "Concise explanation of the root cause of the failure",
					},
					"evidence": {
						Type:        "array",
						Description: "List of evidence from logs and metrics supporting the diagnosis",
						Items: &PropertySchema{
							Type: "string",
						},
					},
					"recommended_fixes": {
						Type:        "array",
						Description: "List of specific fixes to resolve the issue",
						Items: &PropertySchema{
							Type: "string",
						},
					},
					"safe_to_retry": {
						Type:        "boolean",
						Description: "Whether the job is safe to retry without changes",
					},
					"prevention": {
						Type:        "string",
						Description: "Advice on how to prevent this failure in the future",
					},
				},
				Required: []string{"root_cause", "evidence", "recommended_fixes", "safe_to_retry", "prevention"},
			},
		},
	}

	// Create the diagnosis prompt
	systemPrompt := `You are an ML infrastructure debugging assistant specializing in AMD GPU and ROCm workloads.
Analyze the workload failure data and provide a comprehensive diagnosis.
Focus on AMD-specific issues like HIP/ROCm errors, GPU memory management, and driver compatibility.`

	userPrompt := `Given the following workload failure data, diagnose the issue:

` + workloadData + `

Provide:
1. Root cause - concise explanation of what went wrong
2. Evidence - specific log lines, metrics, or patterns that support your diagnosis
3. Recommended fixes - actionable steps to resolve the issue
4. Safe to retry - whether retrying without changes will succeed
5. Prevention - advice to prevent this failure in the future`

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	reqBody := Request{
		Model:      c.Model,
		Messages:   messages,
		MaxTokens:  2000,
		Tools:      []Tool{diagnosisFunction},
		ToolChoice: &ToolChoice{
			Type: "function",
			Function: FunctionCallChoice{
				Name: "diagnose_ml_failure",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fireworksResp Response
	if err := json.Unmarshal(body, &fireworksResp); err != nil {
		return nil, err
	}

	if len(fireworksResp.Choices) == 0 {
		return nil, nil
	}

	// Extract function call arguments
	message := fireworksResp.Choices[0].Message
	if len(message.ToolCalls) == 0 {
		return nil, nil
	}

	// Parse the function arguments JSON
	var result DiagnosisResult
	if err := json.Unmarshal([]byte(message.ToolCalls[0].Function.Arguments), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DiagnoseAgentFailure uses function-calling to diagnose agent execution failures
func (c *Client) DiagnoseAgentFailure(prompt string) (string, error) {
	if c == nil {
		return "", nil
	}

	// Define the function schema for agent diagnosis
	agentDiagnosisFunction := Tool{
		Type: "function",
		Function: Function{
			Name:        "diagnose_agent_failure",
			Description: "Diagnose autonomous agent failure and provide structured analysis",
			Parameters: FunctionParameter{
				Type: "object",
				Properties: map[string]PropertySchema{
					"root_cause": {
						Type:        "string",
						Description: "Concise explanation of why the agent failed",
					},
					"evidence": {
						Type:        "array",
						Description: "List of evidence from tool calls, model calls, and execution trace",
						Items: &PropertySchema{
							Type: "string",
						},
					},
					"recommended_fixes": {
						Type:        "array",
						Description: "List of specific fixes for the agent implementation",
						Items: &PropertySchema{
							Type: "string",
						},
					},
					"prevention": {
						Type:        "string",
						Description: "Advice on how to prevent this agent failure pattern",
					},
				},
				Required: []string{"root_cause", "evidence", "recommended_fixes", "prevention"},
			},
		},
	}

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	reqBody := Request{
		Model:      c.Model,
		Messages:   messages,
		MaxTokens:  2000,
		Tools:      []Tool{agentDiagnosisFunction},
		ToolChoice: &ToolChoice{
			Type: "function",
			Function: FunctionCallChoice{
				Name: "diagnose_agent_failure",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var fireworksResp Response
	if err := json.Unmarshal(body, &fireworksResp); err != nil {
		return "", err
	}

	if len(fireworksResp.Choices) == 0 || len(fireworksResp.Choices[0].Message.ToolCalls) == 0 {
		return "", nil
	}

	// Return the function call arguments as JSON string
	return fireworksResp.Choices[0].Message.ToolCalls[0].Function.Arguments, nil
}

// Legacy method for backward compatibility
func (c *Client) ChatCompletion(messages []Message) (string, error) {
	if c == nil {
		return "", nil
	}

	reqBody := Request{
		Model:     c.Model,
		Messages:  messages,
		MaxTokens: 1000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var fireworksResp Response
	if err := json.Unmarshal(body, &fireworksResp); err != nil {
		return "", err
	}

	if len(fireworksResp.Choices) == 0 {
		return "", nil
	}

	return fireworksResp.Choices[0].Message.Content, nil
}
