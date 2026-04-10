// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"

	"github.com/sandialabs/bibcheck/schema"
)

func NewNumEntriesResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.NumEntriesJSONSchema("num_entries", "number"),
	}
}

func (c *Client) NumEntries(b64 string) (int, error) {
	req := ChatRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []Message{
			systemString(`Determine the number of entries in the bibliography or references section of the provided document.
- Count bibliography entries only.
- Do not count citations in the main body.
- Produce JSON.`),
			userBase64File(b64),
		},
		ResponseFormat: NewNumEntriesResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	result := struct {
		NumEntries int `json:"num_entries"`
	}{}
	if err := c.chatStructured(req, &result); err != nil {
		return -1, fmt.Errorf("NumEntries error: %w", err)
	}

	return result.NumEntries, nil
}
