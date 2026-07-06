// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/documentmetadata"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewExtractDocumentMetadataRF() *openai.ResponseFormat {
	return openai.NewResponseFormat(schema.DocumentMetadataJSONSchema())
}

func newFromTextRequest(model, text string) *openai.ChatRequest {
	return &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the following from the provided document:
- Title (string)
- Authors (array of string)
- Publication Date (string, prefering YYYY-MM-DD, but YYYY-MM or YYYY okay)

Use the following guidelines:
- Prefer user-visible data to embedded metadata.
- The user wants data about the document itself: don't provide information from a bibliography or external references.
- Provide empty values when information is not present.
- Produce JSON.
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
	}
}

func newFromHtmlRequest(model, html string) *openai.ChatRequest {
	return &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(documentmetadata.HTMLPrompt),
			openai.MakeUserMessage(documentmetadata.PrepareHTML([]byte(html), documentmetadata.DefaultConfig())),
		},
		ResponseFormat: NewExtractDocumentMetadataRF(),
	}
}

func (c *Workflow) HTMLMetadata(html []byte) (*documents.Metadata, error) {
	return c.extractDocumentMetadataImpl(newFromHtmlRequest(c.model, string(html)))
}

func (c *Workflow) TextMetadata(text string) (*documents.Metadata, error) {
	return c.extractDocumentMetadataImpl(newFromTextRequest(c.model, text))
}

func (w *Workflow) PDFMetadata(content []byte) (*documents.Metadata, error) {
	tResp, err := w.TextractContent(content)
	if err != nil {
		return nil, fmt.Errorf("textract error: %w", err)
	}
	return w.TextMetadata(tResp.Text)
}

func (c *Workflow) extractDocumentMetadataImpl(req *openai.ChatRequest) (*documents.Metadata, error) {
	content, err := c.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return nil, fmt.Errorf("chat choice 0 error: %w", err)
	}

	d := documents.Metadata{}
	if err := json.Unmarshal(content, &d); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &d, nil

}
