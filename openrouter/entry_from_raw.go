// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"fmt"

	"github.com/sandialabs/bibcheck/schema"
)

func NewBibEntryTextResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.BibliographyEntryLookupJSONSchema(),
	}
}

func (c *Client) EntryFromRaw(b64 string, i int) (string, error) {
	req := ChatRequest{
		Model: "google/gemini-2.5-pro",
		Messages: []Message{
			systemString(`Report whether the requested bibliography entry exists in the provided document.
If so, extract ONLY THAT ENTRY from the bibliography of the provided document.
- Focus on the bibliography or references section.
- The user is not asking for a bibliography entry of the provided document itself.
- Provide it as a single line with the exact bibliographic entry contents.
- Preserve any errors in the entry.
- Omit the inline reference ID that the document uses, e.g. [1] or [Smith1997].
Produce JSON.`),
			userStringAndBase64File(fmt.Sprintf("Extract bibliography entry %d", i), b64),
		},
		ResponseFormat: NewBibEntryTextResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	result := struct {
		EntryExists       bool   `json:"entry_exists"`
		BibliographyEntry string `json:"bibliography_entry"`
	}{}
	if err := c.chatStructured(req, &result); err != nil {
		return "", fmt.Errorf("EntryFromRaw error: %w", err)
	}

	if !result.EntryExists {
		return "", fmt.Errorf("entry does not exist")
	}

	return result.BibliographyEntry, nil
}
