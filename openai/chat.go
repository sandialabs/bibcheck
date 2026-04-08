// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
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

const maxRetries = 3

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

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create the HTTP request
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

		// Make the request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			if isRetryableTimeout(err) && attempt < maxRetries {
				waitFor := retryDelayForAttempt(attempt)
				log.Printf(
					"openai: timeout retry_after=%s attempt=%d/%d error=%v",
					waitFor,
					attempt+1,
					maxRetries+1,
					err,
				)
				time.Sleep(waitFor)
				continue
			}
			return nil, fmt.Errorf("http.Client.Do error: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
		}

		correlationIDs := extractCorrelationIDs(resp.Header)
		correlationLog := formatCorrelationIDs(correlationIDs)

		// Check status code
		if resp.StatusCode == http.StatusOK {
			var chatResp ChatResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
			return &chatResp, nil
		}

		retryAfter, hasRetryAfter := retryAfterDuration(resp.Header)
		if (resp.StatusCode == http.StatusTooManyRequests || hasRetryAfter) && attempt < maxRetries {
			waitFor := retryAfter
			if !hasRetryAfter {
				waitFor = retryDelayForAttempt(attempt)
			}
			log.Printf(
				"openai: rate limited by upstream status=%d retry_after=%s attempt=%d/%d correlation_ids=%s",
				resp.StatusCode,
				waitFor,
				attempt+1,
				maxRetries+1,
				correlationLog,
			)
			time.Sleep(waitFor)
			continue
		}

		log.Printf(
			"openai: API request failed status=%d correlation_ids=%s",
			resp.StatusCode,
			correlationLog,
		)
		return nil, fmt.Errorf(
			"API request failed with status %d (correlation_ids=%s): %s",
			resp.StatusCode,
			correlationLog,
			string(body),
		)
	}

	return nil, fmt.Errorf("API request exhausted retries")
}

func Temperature(t float64) *float64 {
	return &t
}

func retryDelayForAttempt(attempt int) time.Duration {
	return time.Duration(attempt+1) * time.Second
}

func retryAfterDuration(headers http.Header) (time.Duration, bool) {
	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	if retryAfter == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}

	if when, err := http.ParseTime(retryAfter); err == nil {
		d := time.Until(when)
		if d < 0 {
			d = 0
		}
		return d, true
	}

	return 0, false
}

func extractCorrelationIDs(headers http.Header) []string {
	keys := []string{
		"x-request-id",
		"request-id",
		"x-correlation-id",
		"correlation-id",
		"apim-request-id",
		"x-ms-client-request-id",
		"x-litellm-request-id",
		"x-litellm-call-id",
		"traceparent",
	}

	ids := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(headers.Get(key))
		if value == "" {
			continue
		}
		entry := key + "=" + value
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		ids = append(ids, entry)
	}
	return ids
}

func formatCorrelationIDs(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

func isRetryableTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "Client.Timeout exceeded while awaiting headers") ||
		strings.Contains(msg, "context deadline exceeded")
}
