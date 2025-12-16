// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/cwpearson/bibliography-checker/openai"
)

func NewExtractDOIRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "doi",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"doi": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"doi"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseDOI(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Check if the bibliography entry contains a DOI.
If so, provide the DOI.
Otherwise, provide an empty string.
Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewExtractDOIRF(),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return "", fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected one choice in response")
	}

	s := struct {
		DOI string `json:"doi"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.DOI, nil
}
