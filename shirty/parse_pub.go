// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParsePubRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ParsePubJSONSchema(),
	}
}

// ParsePub returns the title of a journal or a book from a bibliography entry
func (w *Workflow) ParsePub(text string) (string, error) {
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

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

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return "", fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := struct {
		Title string `json:"title"`
	}{}
	if err := json.Unmarshal(content, &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Title, nil
}
