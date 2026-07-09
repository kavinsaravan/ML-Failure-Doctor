package fireworks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type Client struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewClient() *Client {
	apiKey := os.Getenv("FIREWORKS_API_KEY")
	if apiKey == "" {
		return nil
	}

	return &Client{
		APIKey:  apiKey,
		BaseURL: "https://api.fireworks.ai/inference/v1/chat/completions",
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) ChatCompletion(messages []Message) (string, error) {
	if c == nil {
		return "", nil
	}

	reqBody := Request{
		Model:     "accounts/fireworks/models/gemma2-9b-it",
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
