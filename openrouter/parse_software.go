// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseSoftwareRF() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.SoftwareJSONSchema(),
	}
}

func (c *Client) ParseSoftware(text string) (*entries.Software, error) {
	baseURL := c.baseUrl
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the name, developers, and homepage URL of the software package referenced in this bibliography entry.
If specific information is not provided, leave the field empty.
Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewParseSoftwareRF(),
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
		s := entries.Software{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &s, nil
	}

	return nil, fmt.Errorf("content was not string")
}
