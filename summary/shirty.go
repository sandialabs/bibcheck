// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package summary

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/shirty"
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

type ShirtySummarizer struct {
	W *shirty.Workflow
}

func NewShirtySummarizer(w *shirty.Workflow) *ShirtySummarizer {
	return &ShirtySummarizer{
		W: w,
	}
}

// Analyze returns (mismatch, comment, error)
func (s *ShirtySummarizer) Summarize(lr *lookup.Result) (bool, string, error) {
	temp := new(float64)
	*temp = 0.0

	others := []string{}

	model := analyze_model_openai_gpt_oss_120B
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(analyze_prompt_openai_gpt_oss_120B),
			openai.MakeUserMessage(
				fmt.Sprintf("ORIGINAL ENTRY TEXT:\n%s", lr.Text) +
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

	content, err := s.W.ChatGetChoiceZero(req)
	if err != nil {
		return false, "", err
	}
	result := struct {
		PossibleMismatch bool
		Comment          string
	}{}
	if err := json.Unmarshal(content, &s); err != nil {
		return false, "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return result.PossibleMismatch, result.Comment, nil
}
