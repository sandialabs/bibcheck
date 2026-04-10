// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"

	"github.com/sandialabs/bibcheck/schema"
)

func NewParsePubRF() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ParsePubJSONSchema(),
	}
}

// ParsePub returns the title of a journal or a book from a bibliography entry
func (c *Client) ParsePub(text string) (string, error) {
	model := "google/gemini-2.5-flash"
	temperature := new(int)
	*temperature = 0

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the title of the journal, book, proceedings, report, or other publication venue from the provided bibliography entry.
- Do not return the title of the article or work itself.
- Return the venue exactly as it appears in the bibliography entry.
- If there is no such venue, return an empty string.
- Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParsePubRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
		Temperature: temperature,
	}

	result := struct {
		Title string `json:"title"`
	}{}
	if err := c.chatStructured(req, &result); err != nil {
		return "", fmt.Errorf("ParsePub error: %w", err)
	}

	return result.Title, nil
}
