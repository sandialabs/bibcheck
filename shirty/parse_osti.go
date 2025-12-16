// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func NewParseOSTIRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "osti",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"osti": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"osti"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseOSTI(text string) (string, error) {

	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Check if the bibliography entry contains an OSTI ID.
If so, provide the OSTI ID.
Otherwise, provide an empty string.
Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseOSTIRF(),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return "", fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected one choice in response")
	}

	s := struct {
		OSTI string `json:"osti"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.OSTI, nil
}
