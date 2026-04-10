// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"

	"github.com/sandialabs/bibcheck/schema"
)

func NewParseTitleRF() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ParseTitleJSONSchema(),
	}
}

func (c *Client) ParseTitle(text string) (string, error) {
	model := "google/gemini-2.5-flash"
	temperature := new(int)
	*temperature = 0

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the title from the provided bibliography entry.
- Extract the title exactly as it appears in the bibliography entry.
- If there is no title, return an empty string.
- Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParseTitleRF(),
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
		return "", fmt.Errorf("ParseTitle error: %w", err)
	}

	return result.Title, nil
}
