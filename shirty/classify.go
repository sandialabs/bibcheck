// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewClassifyEntryRF() *openai.ResponseFormat {
	return openai.NewResponseFormat(schema.ClassifyEntryJSONSchema())
}

func (w *Workflow) Classify(text string) (string, error) {
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

	temp := new(float64)
	*temp = 0

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine what kind of bibliography entry the user provides:
- ` + entries.KindScientificPublication + `
- ` + entries.KindSoftwarePackage + `
- ` + entries.KindWebsite + `
- ` + entries.KindUnknown + `

Hew to the following guidelines
- "` + entries.KindWebsite + `" should be used for anything with a URL that does not fit another category.
- "` + entries.KindScientificPublication + `" usually requires authors, a title, and a venue.
- Produce JSON.`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewClassifyEntryRF(),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return entries.KindUnknown, fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := struct {
		Kind string `json:"kind"`
	}{}
	if err := json.Unmarshal(content, &s); err != nil {
		return entries.KindUnknown, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Kind, nil
}
