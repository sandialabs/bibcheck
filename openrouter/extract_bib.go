// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"fmt"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/schema"
)

type Entry struct {
	EntryId   string `json:"entry_id"`
	EntryText string `json:"entry_text"`
}

func NewExtractBibResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ExtractBibJSONSchema(),
	}
}

func (c *Client) ExtractBib(b *documents.Bibliography) ([]Entry, error) {
	req := ChatRequest{
		Model: "google/gemini-2.5-pro",
		Messages: []Message{
			systemString(`Extract the bibliography from the provided document.
- Only extract entries from the document bibliography or references section.
- Do not create a bibliography reference for the document itself.
- Return each bibliography entry as a single line.
- Preserve any errors present in the extracted entries.
- Use the document's bibliography identifier for entry_id, but omit that identifier from entry_text.
- Produce JSON.`),
			userBase64File(base64.StdEncoding.EncodeToString(b.PDF)),
		},
		ResponseFormat: NewExtractBibResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	entries := []Entry{}
	if err := c.chatStructured(req, &entries); err != nil {
		return nil, fmt.Errorf("ExtractBib error: %w", err)
	}

	return entries, nil
}
