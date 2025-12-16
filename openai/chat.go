// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cwpearson/bibliography-checker/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type       string `json:"type"`
	JSONSchema any    `json:"json_schema"`
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
}

type Choice struct {
	Message Message `json:"Message"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

func NewResponseFormat(schema any) *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema,
	}
}

func MakeUserMessage(c string) Message {
	return Message{
		Role:    "user",
		Content: c,
	}
}

func MakeSystemMessage(c string) Message {
	return Message{
		Role:    "system",
		Content: c,
	}
}

func (c *Client) Chat(req *ChatRequest) (*ChatResponse, error) {
	url := c.baseUrl + "/chat/completions"

	// Marshal the request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("User-Agent", config.UserAgent())

	client := &http.Client{
		Timeout: c.timeout,
	}

	// Make the request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

func Temperature(t float64) *float64 {
	return &t
}
