// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/bibliography"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

const (
	BibIdFormatUnknown      string = bibliography.BibIDFormatUnknown
	BibIdFormatNumeric      string = bibliography.BibIDFormatNumeric
	BibIdFormatAlphanumeric string = bibliography.BibIDFormatAlphanumeric
)

func (w *Workflow) BibIdFormat(b *documents.Bibliography) (string, error) {
	text, err := b.Content()
	if err != nil {
		return BibIdFormatUnknown, err
	}

	temp := new(float64)
	*temp = 0.1

	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine the bibliography cross-reference format in the provided document:
- numeric (e.g. [1])
- alphanumeric (e.g. [Smith1997])
Produce JSON.`),
			openai.MakeUserMessage(fmt.Sprintf("DOCUMENT TEXT:\n%s", text)),
		},
		Temperature:    temp,
		ResponseFormat: openai.NewResponseFormat(schema.BibIDFormatJSONSchema(BibIdFormatNumeric, BibIdFormatAlphanumeric)),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return BibIdFormatUnknown, fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := struct {
		Format string `json:"id_format"`
	}{}
	if err := json.Unmarshal(content, &s); err != nil {
		return BibIdFormatUnknown, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Format, nil
}
