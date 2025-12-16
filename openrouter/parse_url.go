// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
)

func NewParseURLRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "url",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"url"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) ParseURL(text string) (string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Check if the bibliography entry contains a URL that the content is available at.
If so, provide the URL.
Otherwise, provide an empty string.
Produce JSON.
`),
			userString(text),
		},
		ResponseFormat: NewParseURLRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return "", fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		s := struct {
			Url string `json:"url"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.Url, nil
	}

	return "", fmt.Errorf("content was not string")
}
