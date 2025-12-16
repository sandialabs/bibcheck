// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func NewParseTitleRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "title",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"title"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseTitle(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	temp := new(float64)
	*temp = 0

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the title from the provided bibliography entry.
- Produce JSON
- Extract the title exactly as it appears in the bibliography entry
- If there is no title, produce an empty string.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseTitleRF(),
		Temperature:    temp,
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return "", fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected one choice in response")
	}

	s := struct {
		Title string `json:"title"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Title, nil
}
