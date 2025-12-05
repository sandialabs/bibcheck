// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openai"
)

func NewParseOnlineRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "website",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]string{
						"type": "string",
					},
					"authors": map[string]any{
						"type": "array",
						"items": map[string]string{
							"type": "string",
						},
					},
					"url": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"title", "authors", "url"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseOnline(text string) (*entries.Online, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the title, authors, and URL of the online resource from this bibliography entry.
- Produce JSON
- If the bibliography entry does not appear to be an online resource (e.g., no URL), produce an empty string for all values
- If the title or authors are missing, produce an empty string for the corresponding value
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseOnlineRF(),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	online := entries.Online{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &online); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &online, nil
}
