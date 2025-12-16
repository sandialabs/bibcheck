// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cwpearson/bibliography-checker/entries"
	"github.com/cwpearson/bibliography-checker/openai"
)

func NewParseSoftwareRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
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

func (w *Workflow) ParseSoftware(text string) (*entries.Software, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the name, developers, and homepage URL of the software package referenced in this bibliography entry.
If specific information is not provided, leave the field empty.
Produce JSON.`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseSoftwareRF(),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	s := entries.Software{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	s.HomepageUrl = strings.TrimSpace(s.HomepageUrl)

	return &s, nil
}
