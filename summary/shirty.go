// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package summary

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/shirty"
)

var (
	// gpt-oss-120b seems unable to consistently obey the response format
	analyze_model_llama_33_70B_instruct  = "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"
	analyze_prompt_llama_33_70B_instruct = `The user will provide you with a bibliography entry, and some results for searching external databases for that entry. Determine whether the bibliography entry matches the search results.
- Search results that conflict with the entry are almost certainly a mismatch
    - The author list must provide the same authors in the same order (allowing for "et al." at the end)
    - The title, venue, and date must be the same.
    - Allow for formatting differences (author name abbreviations, title capitalization, date formatting, etc)
- Provide a one phrase explanation
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
	if lr.Arxiv.Entry != nil {
		others = append(others, lr.Arxiv.Entry.ToString())
	}
	if lr.Crossref.Work != nil {
		others = append(others, lr.Crossref.Work.ToString())
	}
	if lr.DOIOrg.Found {
		others = append(others, "<DOI from bibliography entry exists, no metadata provided.>")
	}
	if lr.Elsevier.Result != nil {
		others = append(others, lr.Elsevier.Result.ToString())
	}
	if lr.Online.Metadata != nil {
		others = append(others, lr.Online.Metadata.ToString())
	}
	if lr.OSTI.Record != nil {
		others = append(others, lr.OSTI.Record.ToString())
	}

	// if there are no results, the provided entry can't be grounded in real results
	if len(others) == 0 {
		log.Printf("No search results to summarize")
		return true, "insufficient search result data", nil
	}

	model := analyze_model_llama_33_70B_instruct
	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(analyze_prompt_llama_33_70B_instruct),
			openai.MakeUserMessage(
				fmt.Sprintf("BIBLIOGRAPHY ENTRY:\n%s", lr.Text) +
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
						"explanation": map[string]string{
							"type": "string",
						},
						"possible_mismatch": map[string]string{
							"type": "boolean",
						},
					},
					"required":             []string{"possible_mismatch", "explanation"},
					"additionalProperties": false,
				},
			},
		),
	}
	content, err := s.W.OpenAIClient().ChatGetChoiceZero(req)
	if err != nil {
		return false, "", err
	}

	result := struct {
		Explanation      string `json:"explanation"`
		PossibleMismatch bool   `json:"possible_mismatch"`
	}{}
	if err := json.Unmarshal(content, &result); err != nil {
		return false, "", fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return result.PossibleMismatch, result.Explanation, nil
}
