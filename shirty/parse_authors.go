// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openai"
)

func NewParseAuthorsRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "authors",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"authors": map[string]any{
						"type": "array",
						"items": map[string]string{
							"type": "string",
						},
					},
					"has_et_al": map[string]string{
						"type": "boolean",
					},
				},
				"required":             []string{"authors", "has_et_al"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseAuthors(text string) (*entries.Authors, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	temp := new(float64)
	*temp = 0

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract authors from the provided bibliography entry.
- Produce JSON
- Extract all authors from the bibliography, exactly as written.
- Produce an array of authors
- If there are no authors, produce an empty array.
- Indicate whether the bibliography entry contains "et al." or some other indication that it's not a comprehensive list of authors.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseAuthorsRF(),
		Temperature:    temp,
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	s := entries.Authors{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return &s, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &s, nil
}
