// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an OpenRouter API client
type Client struct {
	apiKey     string
	baseUrl    string
	httpClient *http.Client
}

// NewClient creates a new OpenRouter client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseUrl: "https://openrouter.ai/api/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type File struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type FileContent struct {
	Type string `json:"type"`
	File File   `json:"file"`
}

func MakeTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

func MakeFileContent(b64 string) FileContent {
	return FileContent{
		Type: "file",
		File: File{
			Filename: "document.pdf",
			FileData: fmt.Sprintf("data:application/pdf;base64,%s", b64),
		},
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func userString(s string) Message {
	return Message{
		Role: "user",
		Content: []any{
			MakeTextContent(s),
		},
	}
}

func userBase64File(b64 string) Message {
	return Message{
		Role: "user",
		Content: []any{
			MakeFileContent(b64),
		},
	}
}

func userStringAndBase64File(s, b64 string) Message {
	return Message{
		Role: "user",
		Content: []any{
			MakeTextContent(s),
			MakeFileContent(b64),
		},
	}
}

func systemString(s string) Message {
	return Message{
		Role: "system",
		Content: []any{
			MakeTextContent(s),
		},
	}
}

// ResponseFormat represents the expected response format
type ResponseFormat struct {
	Type       string                 `json:"type"`
	JSONSchema map[string]interface{} `json:"json_schema"`
}

// Provider represents provider configuration
type Provider struct {
	RequireParameters bool   `json:"require_parameters"`
	Sort              string `json:"sort,omitempty"`
}

// ChatRequest represents the request payload
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Provider       Provider        `json:"provider,omitempty"`
	Temperature    *int            `json:"temperature,omitempty"`
}

// ChatResponse represents the API response
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Message Message `json:"message"`
}

// ChatCompletion sends a chat completion request
func (c *Client) ChatCompletion(req ChatRequest, baseURL string) (*ChatResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// log.Println(string(jsonData))

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// log.Println(chatResp)

	return &chatResp, nil

}
