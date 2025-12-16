// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
)

func NewExtractArxivRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "arxiv",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"arxiv": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"arxiv"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) ParseArxiv(text string) (string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Check if the bibliography entry contains an arxiv.org URL.
If so, provide the arxiv URL.
Otherwise, provide an empty string.
Produce JSON.
`),
			userString(text),
		},
		ResponseFormat: NewExtractArxivRF(),
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
			Arxiv string `json:"arxiv"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.Arxiv, nil
	}

	return "", fmt.Errorf("content was not string")
}
