// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/cwpearson/bibliography-checker/entries"
	"github.com/cwpearson/bibliography-checker/openai"
)

func NewCompareEntriesRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "compare_entries",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"explanation": map[string]any{
						"type": "string",
					},
					"is_equivalent": map[string]string{
						"type": "boolean",
					},
				},
				"required":             []string{"explanation", "is_equivalent"},
				"additionalProperties": false,
			},
		},
	}
}

func (w *Workflow) Compare(e1, e2 string) (*entries.CompareResult, error) {
	temp := new(float64)
	*temp = 0.2

	model := "meta-llama/Llama-3.3-70B-Instruct"

	query := fmt.Sprintf("ENTRY 1:\n\n%s\n\nENTRY 2:\n\n%s\n", e1, e2)

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`The user will provide two bibliography entries.
Report whether both entries reference the same thing. Briefly comment on any conflicts.
It is okay if one entry has more data than the other: only consider information that is present in both entries.
Make a judement even if available data is limited.
All common data fields must match EXACTLY, with ONLY the following exceptions:
- author lists must be in the same order
- allow trailing authors to be hidden by et al. (but first author must match!)
- allow transcription-style (capitalization, spacing, hypenation, etc.) errors
- allow variations in abbreviations
- allow venue differences if one entry is an OSTI record without other venue information.
Produce JSON.
`),
			openai.MakeUserMessage(query),
		},
		ResponseFormat: NewCompareEntriesRF(),
		Temperature:    temp,
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	cr := entries.CompareResult{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &cr); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &cr, nil
}
