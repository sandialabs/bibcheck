// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func NewParseURLRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "url",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"url"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) ParseURL(text string) (string, error) {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Check if the bibliography entry contains a URL that the content is available at.
If so, provide the URL.
Otherwise, provide an empty string.
Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseURLRF(),
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
		Url string `json:"url"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.Url, nil

}
