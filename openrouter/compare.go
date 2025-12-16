// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
)

func NewCompareEntriesRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "compare_entries",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"explanation": map[string]any{
						"type": "string",
					},
					"is_equivalent": map[string]string{
						"type": "boolean",
					},
				},
				"required":             []string{"explanation", "is_equivalent"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) Compare(e1, e2 string) (*entries.CompareResult, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-pro"

	query := fmt.Sprintf("ENTRY 1:\n\n%s\n\nENTRY 2:\n\n%s\n", e1, e2)

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`The user will provide two bibliography entries.
Report whether the entries equivalent, and provide a very brief explanation.
When comparing, only consider information that is present in both entries.
Data present in both entries must match exactly, with the following considerations:
- author lists must be in the same order
- allow et al. in authors list
- allow transcription-style (capitalization, spacing, hypenation, etc.) errors
- allow variations in abbreviations
- allow venue differences if one entry is an OSTI record without other venue information.
Produce JSON.
`),
			userString(query),
		},
		ResponseFormat: NewCompareEntriesRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		r := entries.CompareResult{}
		if err := json.Unmarshal([]byte(cstring), &r); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &r, nil
	}

	return nil, fmt.Errorf("content was not string")
}
