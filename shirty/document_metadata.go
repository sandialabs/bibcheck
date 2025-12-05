// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/openai"
)

func NewExtractDocumentMetadataRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
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
					"publication_date": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"title", "authors", "publication_date"},
				"additionalProperties": false,
			},
		},
	}
}

func newFromTextRequest(text string) *openai.ChatRequest {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	return &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the following from the provided document:
- Title (string)
- Authors (array of string)
- Publication Date (string, prefering YYYY-MM-DD, but YYYY-MM or YYYY okay)

Use the following guidelines:
- Prefer user-visible data to embedded metadata
- The user wants data about the document itself: don't provide information from a bibliography or external references.
- Provide empty values when information is not present.
- Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
	}
}

func newFromHtmlRequest(html string) *openai.ChatRequest {
	model := "meta-llama/Llama-3.3-70B-Instruct"

	// model has a limit of 128k tokens
	// in one example, a string of 1158716 symbols -> 353591 tokens
	if len(html) > 300_000 {
		log.Printf("Provided more than 300k HTML characters (%d) but context limit is 128k tokens. Truncating to first 225k and last 75k html characters", len(html))
		html = html[:225_000] + html[len(html)-75_000:]
	}

	return &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine the following from the provided website HTML:
- Title (string)
- Authors (array of string)
- Publication/Update Date (string, prefering YYYY-MM-DD, but YYYY-MM or YYYY okay)

Use the following guidelines:
- Wherever user-visible values contradict hidden values (e.g. html meta tags), provide the user-visible values.
- The user wants info about the document itself: don't provide information about external links or references.
- Provide empty values when the requested information cannot be determined.
- Produce JSON.
`),
			openai.MakeUserMessage(html),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
	}
}

func (c *Workflow) HTMLMetadata(html []byte) (*documents.Metadata, error) {
	return c.extractDocumentMetadataImpl(newFromHtmlRequest(string(html)))
}

func (c *Workflow) TextMetadata(text string) (*documents.Metadata, error) {
	return c.extractDocumentMetadataImpl(newFromTextRequest(text))
}

func (w *Workflow) PDFMetadata(content []byte) (*documents.Metadata, error) {
	tResp, err := w.TextractContent(content)
	if err != nil {
		return nil, fmt.Errorf("textract error: %w", err)
	}
	return w.TextMetadata(tResp.Text)
}

func (c *Workflow) extractDocumentMetadataImpl(req *openai.ChatRequest) (*documents.Metadata, error) {
	resp, err := c.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("chat completion error: %w", err)
	}

	// log.Println(resp.Choices)

	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected one choice in response")
	}

	d := documents.Metadata{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &d); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &d, nil

}
