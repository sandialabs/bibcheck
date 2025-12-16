// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/cwpearson/bibliography-checker/entries"
)

func NewParseSoftwareRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "software",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]string{
						"type": "string",
					},
					"developers": map[string]any{
						"type": "array",
						"items": map[string]string{
							"type": "string",
						},
					},
					"homepage_url": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"name", "developers", "homepage_url"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) ParseSoftware(text string) (*entries.Software, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the name, developers, and homepage URL of the software package referenced in this bibliography entry.
If specific information is not provided, leave the field empty.
Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParseSoftwareRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		s := entries.Software{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &s, nil
	}

	return nil, fmt.Errorf("content was not string")
}
