// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseOnlineRF() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.WebsiteJSONSchema(),
	}
}

func (c *Client) ParseOnline(text string) (*entries.Online, error) {
	baseURL := c.baseUrl
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the title, authors, and URL of the online resource from this bibliography entry.
If not provided, leave empty.
Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParseOnlineRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		w := entries.Online{}
		if err := json.Unmarshal([]byte(cstring), &w); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &w, nil
	}

	return nil, fmt.Errorf("content was not string")
}
