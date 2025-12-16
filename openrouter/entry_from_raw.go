// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
)

func NewBibEntryTextResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "bib_entry",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"entry_exists": map[string]string{
						"type": "boolean",
					},
					"bibliography_entry": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"entry_exists"},
				"additionalProperties": false,
			},
		},
	}
}

func (c *Client) EntryFromRaw(b64 string, i int) (string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	// model := "google/gemini-2.5-flash"
	model := "google/gemini-2.5-pro"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Report whether the requested bibliography entry exists in the provided document.
If so, extract ONLY THAT ENTRY from the bibliography of the provided document.
- The user is not asking for a bibliography entry of the provided document itself.
- Provide it as a single line with the exact bibliographic entry contents.
- Preserve any errors in the entry.
- Omit the inline reference ID that the document uses, e.g. [1] or [Smith1997].
Produce JSON.`),
			userStringAndBase64File(fmt.Sprintf("Extract bibliography entry %d", i), b64),
		},
		ResponseFormat: NewBibEntryTextResponseFormat(),
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
			EntryExists       bool   `json:"entry_exists"`
			BibliographyEntry string `json:"bibliography_entry"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		if !s.EntryExists {
			return "", fmt.Errorf("entry does not exist")
		}

		return s.BibliographyEntry, nil
	}

	return "", fmt.Errorf("content was not string")
}
