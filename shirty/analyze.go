// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/openai"
)

var (
	analyze_model_openai_gpt_oss_120B  = "openai/gpt-oss-120b"
	analyze_prompt_openai_gpt_oss_120B = `Compare the provided bibliography entry with metadata resulting from searching for the cited work. Flag if any search result data is inconsistent with the provided bibliography entry.
- It's okay if some formatting is different (author name abbreviations, title capitalization, etc)
- Key metadata needs to be accurate and complete: author list, title, venue, date, etc
- It's okay if the search results are missing some metadata fields
- Provide a brief accompanying explanation of the matching determination
- Produce JSON
`
)

// Analyze returns (mismatch, comment, error)
func (w *Workflow) Analyze(orig string, others []string) (bool, string, error) {
	temp := new(float64)
	*temp = 0.0

	model := analyze_model_openai_gpt_oss_120B
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(analyze_prompt_openai_gpt_oss_120B),
			openai.MakeUserMessage(
				fmt.Sprintf("ORIGINAL ENTRY TEXT:\n%s", orig) +
					"\n\nSEARCH RESULT:\n" +
					strings.Join(others, "\n\nSEARCH RESULT:\n"),
			),
		},
		Temperature: temp,
		ResponseFormat: openai.NewResponseFormat(
			map[string]any{
				"name":   "compare",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"possible_mismatch": map[string]string{
							"type": "boolean",
						},
						"comment": map[string]string{
							"type": "string",
						},
					},
					"required":             []string{"possible_mismatch", "comment"},
					"additionalProperties": false,
				},
			},
		),
	}

	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return false, "", fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return false, "", fmt.Errorf("expected 1 choice in openai response")
	}

	s := struct {
		PossibleMismatch bool
		Comment          string
	}{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &s); err != nil {
		return false, "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return s.PossibleMismatch, s.Comment, nil
}
