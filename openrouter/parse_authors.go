// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseAuthorsRF() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ParseAuthorsJSONSchema(),
	}
}

func (c *Client) ParseAuthors(text string) (*entries.Authors, error) {
	model := "google/gemini-2.5-flash"
	temperature := new(int)
	*temperature = 0

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract authors from the provided bibliography entry.
- Return every author exactly as written in the entry.
- If there are no authors, return an empty array.
- Set has_et_al to true when the entry uses "et al." or otherwise indicates the list is incomplete.
- Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParseAuthorsRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
		Temperature: temperature,
	}

	authors := entries.Authors{}
	if err := c.chatStructured(req, &authors); err != nil {
		return nil, fmt.Errorf("ParseAuthors error: %w", err)
	}

	return &authors, nil
}
