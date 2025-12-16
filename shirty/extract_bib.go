// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/cwpearson/bibliography-checker/openai"
)

type Entry struct {
	EntryId   string `json:"entry_id"`
	EntryText string `json:"entry_text"`
}

func (c *Workflow) ExtractBib(text string) ([]Entry, error) {

	temp := new(float64)
	*temp = 0.1

	// notes on models
	// - meta-llama/Llama-3.2-90B-Vision-Instruct: works okay. Likes to keep the inline reference
	// - meta-llama/Llama-4-Scout-17B-16E-Instruct: doesn't seem to be able to follow the prompt
	// - meta-llama/Llama-3.3-70B-Instruct: works okay. Likes to keep the inline reference
	// - microsoft/Phi-3.5-vision-instruct: 500 error
	model := "meta-llama/Llama-3.3-70B-Instruct"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the bibliography from the provided document.
- Do not create a bibliography reference for the text itself - only extract the bibliography from the document.
- Provide each entry as a single line with the exact bibliographic entry contents.
- Preserve any errors in the entries.
Produce JSON.`),
			openai.MakeUserMessage(text),
		},
		Temperature: temp,
		ResponseFormat: openai.NewResponseFormat(
			map[string]any{
				"name":       "bibliography",
				"properties": map[string]any{},
				"strict":     true,
				"schema": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"entry_id": map[string]string{
								"type": "string",
							},
							"entry_text": map[string]string{
								"type": "string",
							},
						},
						"required":             []string{"entry_id", "entry_text"},
						"additionalProperties": false,
					},
				},
			},
		),
	}

	resp, err := c.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected 1 choice in openai response")
	}

	es := []Entry{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &es); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return es, nil
}
