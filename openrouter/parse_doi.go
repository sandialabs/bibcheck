// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
)

func NewExtractDOIRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "doi",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"doi": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"doi"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) ParseDOI(text string) (string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Check if the bibliography entry contains a DOI.
If so, provide the DOI.
Otherwise, provide an empty string.
Produce JSON.
`),
			userString(text),
		},
		ResponseFormat: NewExtractDOIRF(),
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
			DOI string `json:"doi"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.DOI, nil
	}

	return "", fmt.Errorf("content was not string")
}
