// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

var (
	// gpt-oss-120b seems unable to consistently obey the response format
	summaryModelLlama33_70BInstruct  = "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"
	summaryPromptLlama33_70BInstruct = `The user will provide you with a bibliography entry, and some results for searching external databases for that entry. Determine whether the bibliography entry matches the search results.
- Search results that conflict with the entry are almost certainly a mismatch
    - The author list must provide the same authors in the same order (allowing for "et al." at the end)
    - The title, venue, and date must be the same.
    - Allow for formatting differences (author name abbreviations, title capitalization, date formatting, etc)
- Provide a one phrase explanation
- Produce JSON
`
)

// Summarize returns (mismatch, comment, error).
func (w *Workflow) Summarize(lr *lookup.Result) (bool, string, error) {
	temp := new(float64)
	*temp = 0.0

	searchResults := []string{}
	if lr.Arxiv.Entry != nil {
		searchResults = append(searchResults, lr.Arxiv.Entry.ToString())
	}
	if lr.Crossref.Work != nil {
		searchResults = append(searchResults, lr.Crossref.Work.ToString())
	}
	if lr.DOIOrg.Found {
		searchResults = append(searchResults, "<DOI from bibliography entry exists, no metadata provided.>")
	}
	if lr.Elsevier.Result != nil {
		searchResults = append(searchResults, lr.Elsevier.Result.ToString())
	}
	if lr.Online.Metadata != nil {
		searchResults = append(searchResults, lr.Online.Metadata.ToString())
	}
	if lr.OSTI.Record != nil {
		searchResults = append(searchResults, lr.OSTI.Record.ToString())
	}

	if len(searchResults) == 0 {
		log.Printf("No search results to summarize")
		return true, "insufficient search result data", nil
	}

	req := &openai.ChatRequest{
		Model: summaryModelLlama33_70BInstruct,
		Messages: []openai.Message{
			openai.MakeSystemMessage(summaryPromptLlama33_70BInstruct),
			openai.MakeUserMessage(
				fmt.Sprintf("BIBLIOGRAPHY ENTRY:\n%s", lr.Text) +
					"\n\nSEARCH RESULT:\n" +
					strings.Join(searchResults, "\n\nSEARCH RESULT:\n"),
			),
		},
		Temperature:    temp,
		ResponseFormat: openai.NewResponseFormat(schema.SummaryJSONSchema()),
	}
	content, err := w.oaiClient.ChatGetChoiceZero(req)
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
