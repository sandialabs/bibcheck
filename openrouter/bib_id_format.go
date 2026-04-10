// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"fmt"

	"github.com/sandialabs/bibcheck/bibliography"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/schema"
)

const (
	BibIdFormatUnknown      string = bibliography.BibIDFormatUnknown
	BibIdFormatNumeric      string = bibliography.BibIDFormatNumeric
	BibIdFormatAlphanumeric string = bibliography.BibIDFormatAlphanumeric
)

func NewBibIDFormatResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.BibIDFormatJSONSchema(BibIdFormatNumeric, BibIdFormatAlphanumeric),
	}
}

func (c *Client) BibIdFormat(b *documents.Bibliography) (string, error) {
	req := ChatRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []Message{
			systemString(`Determine the bibliography cross-reference format in the provided document.
- numeric (for example [1])
- alphanumeric (for example [Smith1997])
- Base the answer on the bibliography or references section.
- Produce JSON.`),
			userBase64File(base64.StdEncoding.EncodeToString(b.PDF)),
		},
		ResponseFormat: NewBibIDFormatResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}

	result := struct {
		Format string `json:"id_format"`
	}{}
	if err := c.chatStructured(req, &result); err != nil {
		return BibIdFormatUnknown, fmt.Errorf("BibIdFormat error: %w", err)
	}

	return result.Format, nil
}
