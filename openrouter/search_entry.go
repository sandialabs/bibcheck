// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"
	"strings"
)

func (c *Client) SearchEntry(text string) (bool, string, error) {
	baseURL := "https://openrouter.ai/api/v1"
	// model := "perplexity/sonar"
	model := "perplexity/sonar-pro"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`The user is trying to determine whether this bibliography entry is real.
It is not sufficient that the entry APPEARS convincing - it must match a scientific publication in the search results.
Authors, title, and venue must match exactly (allowing for et.al, transcription-style errors, and variations in abbreviations).
Respond YES [very brief comments] if the entry appears in the search results.
Otherwise, respond NO [very brief comments].
The user DOES NOT WANT a summary of search results, NOR of the cited work if it exists.
`),
			userString(text),
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
