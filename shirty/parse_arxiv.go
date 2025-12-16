// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func NewExtractArxivRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "arxiv",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"arxiv": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"arxiv"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseArxiv(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Check if the bibliography entry contains an arxiv.org URL.
If so, provide the arxiv URL.
Otherwise, provide an empty string.
Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewExtractArxivRF(),
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
		Arxiv string `json:"arxiv"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Arxiv, nil
}
