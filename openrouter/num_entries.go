// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"fmt"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/schema"
)

func NewNumEntriesResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.NumEntriesJSONSchema("num_entries", "number"),
	}
}

func (c *Client) NumEntries(b64 string) (int, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return -1, fmt.Errorf("decode base64 pdf error: %w", err)
	}

	bibliography, err := c.PrepareBibliographyContent(raw)
	if err != nil {
		return -1, fmt.Errorf("prepare bibliography error: %w", err)
	}

	return c.NumBibliographyEntries(bibliography)
}

func (c *Client) NumBibliographyEntries(b *documents.Bibliography) (int, error) {
	req := ChatRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []Message{
			systemString(`Determine the number of entries in the bibliography or references section of the provided document.
- Count bibliography entries only.
- Do not count citations in the main body.
- Produce JSON.`),
			userBase64File(base64.StdEncoding.EncodeToString(b.PDF)),
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
		return -1, fmt.Errorf("NumBibliographyEntries error: %w", err)
	}

	return result.NumEntries, nil
}
