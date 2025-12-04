// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func NewParsePubRF() *openai.ResponseFormat {
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

// ParsePub returns the title of a journal or a book from a bibliography entry
func (w *Workflow) ParsePub(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	temp := new(float64)
	*temp = 0

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the title of the journal or book from the provided bibliography entry.
- Produce JSON
- Extract a journal or book title; DO NOT EXTRACT the title of the article (or whatever) itself.
- Extract the journal or book title exactly as it appears in the bibliography entry.
- If there is no such journal or book title, produce an empty string.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParsePubRF(),
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
