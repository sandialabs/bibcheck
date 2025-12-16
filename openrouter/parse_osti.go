// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
)

func NewParseOSTIRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "osti",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"osti": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"osti"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) ParseOSTI(text string) (string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Check if the bibliography entry contains an OSTI ID.
If so, provide the OSTI ID.
Otherwise, provide an empty string.
Produce JSON.
`),
			userString(text),
		},
		ResponseFormat: NewParseOSTIRF(),
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
			OSTI string `json:"osti"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.OSTI, nil
	}

	return "", fmt.Errorf("content was not string")
}
