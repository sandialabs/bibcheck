// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"
	"log"
	"strings"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/schema"
)

const (
	summaryModelGemini25FlashLite  = "google/gemini-2.5-flash-lite"
	summaryPromptGemini25FlashLite = `The user will provide you with a bibliography entry, and some results for searching external databases for that entry. Determine whether the bibliography entry matches the search results.
- Search results that conflict with the entry are almost certainly a mismatch
    - The author list must provide the same authors in the same order (allowing for "et al." at the end)
    - The title, venue, and date must be the same.
    - Allow for formatting differences (author name abbreviations, title capitalization, date formatting, etc)
- Provide a one phrase explanation
- Produce JSON
`
)

// Summarize returns (mismatch, comment, error).
func (c *Client) Summarize(lr *lookup.Result) (bool, string, error) {
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

	temperature := new(int)
	*temperature = 0

	result := struct {
		Explanation      string `json:"explanation"`
		PossibleMismatch bool   `json:"possible_mismatch"`
	}{}

	req := ChatRequest{
		Model: summaryModelGemini25FlashLite,
		Messages: []Message{
			systemString(summaryPromptGemini25FlashLite),
			userString(
				fmt.Sprintf("BIBLIOGRAPHY ENTRY:\n%s", lr.Text) +
					"\n\nSEARCH RESULT:\n" +
					strings.Join(searchResults, "\n\nSEARCH RESULT:\n"),
			),
		},
		ResponseFormat: &ResponseFormat{
			Type:       "json_schema",
			JSONSchema: schema.SummaryJSONSchema(),
		},
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
		Temperature: temperature,
	}

	if err := c.chatStructured(req, &result); err != nil {
		return false, "", fmt.Errorf("chat completion error: %w", err)
	}

	return result.PossibleMismatch, result.Explanation, nil
}
