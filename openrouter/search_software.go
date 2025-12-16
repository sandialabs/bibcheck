// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"
	"strings"

	"github.com/cwpearson/bibliography-checker/entries"
)

func (c *Client) SearchSoftware(software *entries.Software) (bool, string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	// model := "perplexity/sonar"
	model := "perplexity/sonar-pro"

	query := fmt.Sprintf("Homepage: %s\nName:%s\nDevelopers: %s", software.HomepageUrl, software.Name, strings.Join(software.Developers, ", "))

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`User will provide a homepage URL, name, and authors for a software package.
Respond with YES [very brief explanation] if software with the provided information appears in the results (allowing for et.al, transcription-style errors, and variations in abbreviations)
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
