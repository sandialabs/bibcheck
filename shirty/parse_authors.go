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
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

	temp := new(float64)
	*temp = 0

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract authors from the provided bibliography entry.
- Extract all authors from the bibliography, exactly as written.
- Produce an array of authors
- If there are no authors, produce an empty array.
- Indicate whether the bibliography entry contains "et al." or some other indication that it's not a comprehensive list of authors.
- Produce JSON
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseAuthorsRF(),
		Temperature:    temp,
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return nil, fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := entries.Authors{}
	if err := json.Unmarshal(content, &s); err != nil {
		return &s, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &s, nil
}
