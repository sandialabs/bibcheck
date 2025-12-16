// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

const (
	BibIdFormatUnknown      string = ""
	BibIdFormatNumeric      string = "numeric"
	BibIdFormatAlphanumeric string = "alphanumeric"
)

func (w *Workflow) BibIdFormat(text string) (string, error) {

	temp := new(float64)
	*temp = 0.1

	model := "meta-llama/Llama-3.3-70B-Instruct"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine the bibliography cross-reference format in the provided document:
- numeric (e.g. [1])
- alphanumeric (e.g. [Smith1997])
Produce JSON.`),
			openai.MakeUserMessage(fmt.Sprintf("DOCUMENT TEXT:\n%s", text)),
		},
		Temperature: temp,
		ResponseFormat: openai.NewResponseFormat(
			map[string]any{
				"name":   "bib_id_format",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id_format": map[string]any{
							"type": "string",
							"enum": []string{BibIdFormatNumeric, BibIdFormatAlphanumeric},
						},
					},
					"required":             []string{"id_format"},
					"additionalProperties": false,
				},
			},
		),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return BibIdFormatUnknown, fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return BibIdFormatUnknown, fmt.Errorf("expected 1 choice in openai response")
	}

	s := struct {
		Format string `json:"id_format"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return BibIdFormatUnknown, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Format, nil
}
