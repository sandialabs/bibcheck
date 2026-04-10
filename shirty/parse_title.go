// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseTitleRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ParseTitleJSONSchema(),
	}
}

func (w *Workflow) ParseTitle(text string) (string, error) {
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

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
