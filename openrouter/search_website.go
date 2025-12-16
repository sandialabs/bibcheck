// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/entries"
)

func (c *Client) SearchWebsite(website *entries.Website) (bool, string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	// model := "perplexity/sonar"
	model := "perplexity/sonar-pro"

	query := fmt.Sprintf("URL: %s\nTitle:%s\nAuthors: %s", website.URL, website.Title, strings.Join(website.Authors, ", "))

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`User will provide a website URL, title, and authors.
Respond with YES [very brief explanation] if a website with the provided information appears in the search (allowing for et.al, transcription-style errors, and variations in abbreviations)
Otherwise, respond NO [very brief explanation].
DO NOT SUMMARIZE THE SEARCH RESULTS.`),
			userString(query),
		},
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return false, "", fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return false, "", fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {

		cleaned := strings.ToLower(strings.TrimSpace(cstring))

		if strings.Contains(cleaned, "yes") {
			after, _ := strings.CutPrefix(cleaned, "yes")
			after = strings.TrimSpace(after)
			after = strings.ReplaceAll(after, "\n", "")
			return true, after, nil
		} else if strings.Contains(cleaned, "no") {
			after, _ := strings.CutPrefix(cleaned, "no")
			after = strings.TrimSpace(after)
			after = strings.ReplaceAll(after, "\n", "")
			return false, after, nil
		} else {
			return false, "", fmt.Errorf("model did not obey the formatting instructions")
		}
	}

	return false, "", fmt.Errorf("content was not string")
}
