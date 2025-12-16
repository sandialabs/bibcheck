// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cwpearson/bibliography-checker/documents"
)

func NewExtractDocumentMetadataRF() *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: map[string]any{
			"name":   "metadata",
			"strict": true,
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]string{
						"type": "string",
					},
					"authors": map[string]any{
						"type": "array",
						"items": map[string]string{
							"type": "string",
						},
					},
					"contributing_org": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"title", "authors", "contributing_org"},
				"additionalProperties": false,
			},
		},
	}
}

func encodedPdfRequest(model, encoded string) *ChatRequest {
	return &ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the following from the provided document:
- Title
- Authors
- Contributing Organization

Use the following guidelines:
- Prefer user-visible data to embedded metadata
- The user wants data about the document itself: don't provide information from a bibliography or external references.
- Provide empty values when information is not present.
- Produce JSON.
`),
			userBase64File(encoded),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}
}

func htmlRequest(model, html string) *ChatRequest {
	return &ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the following from the provided website HTML:
- Title
- Authors
- Contributing Organization

Use the following guidelines:
- Prefer user-visible data to embedded metadata
- The user wants data about the document itself: don't provide information external links or references.
- Provide empty values when information is not present.
- Produce JSON.
`),
			userString(html),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}
}

func (c *Client) HTMLMetadata(html []byte) (*documents.Metadata, error) {
	model := "google/gemini-2.5-flash"
	// model := "mistralai/mistral-medium-3"
	return c.extractDocumentMetadataImpl(htmlRequest(model, string(html)))
}

func (c *Client) PDFMetadata(raw []byte) (*documents.Metadata, error) {
	model := "google/gemini-2.5-flash"
	// model := "mistralai/mistral-medium-3"
	encoded := base64.StdEncoding.EncodeToString(raw)
	return c.extractDocumentMetadataImpl(encodedPdfRequest(model, encoded))
}

func (c *Client) extractDocumentMetadataImpl(req *ChatRequest) (*documents.Metadata, error) {
	baseURL := "https://openrouter.ai/api/v1"

	resp, err := c.ChatCompletion(*req, baseURL)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		d := documents.Metadata{}
		if err := json.Unmarshal([]byte(cstring), &d); err != nil {
			log.Println(cstring)
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &d, nil
	}

	return nil, fmt.Errorf("content was not string")
}

func (c *Client) ExtractDocumentMetadata(encoded string) (*documents.Metadata, error) {
	baseURL := "https://openrouter.ai/api/v1"
	model := "google/gemini-2.5-flash"

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Extract the title and authors from the provided document.
The user wants data about the document itself: ignore any bibliography or external references in the document.
Produce JSON.
Provide an empty value for any information not present.
`),
			userBase64File(encoded),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
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
		d := documents.Metadata{}
		if err := json.Unmarshal([]byte(cstring), &d); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return &d, nil
	}

	return nil, fmt.Errorf("content was not string")
}
