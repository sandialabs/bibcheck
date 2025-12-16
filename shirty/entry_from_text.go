// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/openai"
)

func (w *Workflow) EntryFromText(text string, id int) (string, error) {
	// notes on models
	// - meta-llama/Llama-3.2-90B-Vision-Instruct: works okay. Likes to keep the inline reference
	// - meta-llama/Llama-4-Scout-17B-16E-Instruct: doesn't seem to be able to follow the prompt
	// - meta-llama/Llama-3.3-70B-Instruct: works okay. Likes to keep the inline reference
	// - microsoft/Phi-3.5-vision-instruct: 500 error
	temp := new(float64)
	*temp = 0.0
	model := "meta-llama/Llama-3.3-70B-Instruct"
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the requested bibliography entry from the document's bibliography. Produce JSON.
- Only extract the requested entry.
- Do not create a bibliography reference for the provided document, extract the entry from the bibliography.
- Extract the entire, complete requested entry.
- Provide the extracted entry as a single line.
- The provided text may be mangled due to automated extraction from a source document, try to accomodate.
- Preserve any other errors or incompleteness in the entry
`),
			openai.MakeUserMessage(fmt.Sprintf("Extract bibliography entry %d from the provided document below:\n\nDOCUMENT TEXT:\n\n%s", id, text)),
		},
		Temperature: temp,
		ResponseFormat: openai.NewResponseFormat(
			map[string]any{
				"name":   "bib_entry",
				"strict": true,
				"schema": map[string]any{
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
		),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return "", fmt.Errorf("expected 1 choice in openai response")
	}

	s := struct {
		EntryId   string `json:"entry_id"`
		EntryText string `json:"entry_text"`
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.EntryText, nil
}
