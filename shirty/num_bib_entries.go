// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func (w *Workflow) NumBibEntries(text string) (int, error) {

	temp := new(float64)
	*temp = 0.1

	model := "meta-llama/Llama-3.3-70B-Instruct"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine the number of entries / related works in the document's bibliography.
Produce JSON.`),
			openai.MakeUserMessage(fmt.Sprintf("DOCUMENT TEXT:\n%s", text)),
		},
		Temperature: temp,
		ResponseFormat: openai.NewResponseFormat(
			map[string]any{
				"name":   "num_bib_entries",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"num_entries": map[string]string{
							"type": "integer",
						},
					},
					"required":             []string{"num_entries"},
					"additionalProperties": false,
				},
			},
		),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return -1, fmt.Errorf("openai client chat error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return -1, fmt.Errorf("expected 1 choice in openai response")
	}

	s := struct {
		NumEntries int `json:"num_entries"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return -1, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.NumEntries, nil
}
