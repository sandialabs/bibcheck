// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func (w *Workflow) NumBibEntries(b *documents.Bibliography) (int, error) {
	text, err := b.Content()
	if err != nil {
		return -1, err
	}

	temp := new(float64)
	*temp = 0.1

	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine the number of entries / related works in the document's bibliography.
Produce JSON.`),
			openai.MakeUserMessage(fmt.Sprintf("DOCUMENT TEXT:\n%s", text)),
		},
		Temperature:    temp,
		ResponseFormat: openai.NewResponseFormat(schema.NumEntriesJSONSchema("num_bib_entries", "integer")),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return -1, fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := struct {
		NumEntries int `json:"num_entries"`
	}{}
	if err := json.Unmarshal(content, &s); err != nil {
		return -1, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.NumEntries, nil
}
