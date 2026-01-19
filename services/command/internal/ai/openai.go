package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	openAIURL     = "https://api.openai.com/v1/chat/completions"
	defaultModel  = "gpt-5-nano" // Fast and cheap for simple parsing tasks
	requestTimeout = 30 * time.Second
)

// Client handles communication with OpenAI API
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  defaultModel,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// SetModel allows overriding the default model
func (c *Client) SetModel(model string) {
	c.model = model
}

// chatRequest is the OpenAI API request format
type chatRequest struct {
	Model               string        `json:"model"`
	Messages            []chatMessage `json:"messages"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI API response format
type chatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ParseCommand sends a user message to the LLM and parses the response into a Command
func (c *Client) ParseCommand(ctx context.Context, userMessage string) (*Command, error) {
	// Build the request
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userMessage},
		},
		MaxCompletionTokens: 2000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", openAIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if chatResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse the LLM's JSON response into a Command
	content := chatResp.Choices[0].Message.Content

	var cmd Command
	if err := json.Unmarshal([]byte(content), &cmd); err != nil {
		// LLM didn't return valid JSON - treat as unclear
		return &Command{
			Action:      "unclear",
			Parameters:  map[string]string{"reason": "Failed to parse LLM response", "raw": content},
			Confidence:  0.0,
			Explanation: "LLM response was not valid JSON",
		}, nil
	}

	return &cmd, nil
}
