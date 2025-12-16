// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
	"log"
)

func NewNumEntriesResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "num_entries",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"num_entries": map[string]string{
						"type":        "number",
						"description": "the number of bibliography entries in the document",
					},
				},
				"required":             []string{"num_entries"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) NumEntries(b64 string) (int, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString("Extract the number of bibliography entries from the provided document. Produce JSON"),
			userBase64File(b64),
		},
		ResponseFormat: NewNumEntriesResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return -1, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return -1, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		s := struct {
			NumEntries int `json:"num_entries"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			log.Println(cstring)
			return -1, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.NumEntries, nil
	}

	return -1, fmt.Errorf("content was not string")
}
