// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/cwpearson/bibliography-checker/entries"
	"github.com/cwpearson/bibliography-checker/openai"
)

func NewClassifyEntryRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "entry_exists",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind": map[string]any{
						"type": "string",
						"enum": []string{
							entries.KindScientificPublication,
							entries.KindSoftwarePackage,
							entries.KindWebsite,
							entries.KindUnknown},
					},
				},
				"required":             []string{"kind"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) Classify(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

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

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return entries.KindUnknown, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return entries.KindUnknown, fmt.Errorf("expected one choice in response")
	}

	s := struct {
		Kind string `json:"kind"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return entries.KindUnknown, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Kind, nil
}
