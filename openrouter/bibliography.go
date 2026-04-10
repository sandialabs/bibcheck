// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/schema"
)

func NewBibliographyPageResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.BibliographyPageJSONSchema(),
	}
}

func (c *Client) PrepareBibliography(filePath string) (*documents.Bibliography, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read pdf error: %w", err)
	}
	return c.PrepareBibliographyContent(data)
}

func (c *Client) PrepareBibliographyContent(pdf []byte) (*documents.Bibliography, error) {
	pageCount, err := documents.PDFPageCount(pdf)
	if err != nil {
		return nil, fmt.Errorf("pdf page count error: %w", err)
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("expected pdf to have at least one page")
	}

	matches := make([]bool, pageCount)
	for page := 1; page <= pageCount; page++ {
		pagePDF, err := documents.PDFSlicePages(pdf, page, page)
		if err != nil {
			return nil, fmt.Errorf("slice page %d error: %w", page, err)
		}
		match, err := c.pageContainsBibliography(pagePDF)
		if err != nil {
			return nil, fmt.Errorf("page %d bibliography classification error: %w", page, err)
		}
		matches[page-1] = match
		log.Printf("bibliography page %d/%d match=%t", page, pageCount, match)
	}

	startPage, endPage, ok := bibliographyPageRange(matches)
	bibPDF := pdf
	if !ok {
		startPage = 1
		endPage = pageCount
		log.Printf("bibliography pages not detected; falling back to full pdf (%d pages)", pageCount)
	} else {
		bibPDF, err = documents.PDFSlicePages(pdf, startPage, endPage)
		if err != nil {
			return nil, fmt.Errorf("slice bibliography pages %d-%d error: %w", startPage, endPage, err)
		}
		log.Printf("bibliography pages detected: %d-%d of %d", startPage, endPage, pageCount)
	}

	return &documents.Bibliography{
		PDF:       bibPDF,
		StartPage: startPage,
		EndPage:   endPage,
	}, nil
}

func (c *Client) pageContainsBibliography(pagePDF []byte) (bool, error) {
	temperature := new(int)
	*temperature = 0

	req := ChatRequest{
		Model: "google/gemini-2.5-flash",
		Messages: []Message{
			systemString(`Determine whether the provided PDF page contains any part of the paper's bibliography or references section.
- Return true if the page contains a bibliography heading, one or more bibliography entries, or a continuation of bibliography entries from another page.
- Return false otherwise, i.e., for any page that DOES NOT contain any part of a bibliography: body pages, appendices, acknowledgments, author bios, unrelated back matter, etc.
- Produce JSON.`),
			userBase64File(base64.StdEncoding.EncodeToString(pagePDF)),
		},
		ResponseFormat: NewBibliographyPageResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
		Temperature: temperature,
	}

	result := struct {
		ContainsBibliography bool `json:"contains_bibliography"`
	}{}
	if err := c.chatStructured(req, &result); err != nil {
		return false, fmt.Errorf("pageContainsBibliography error: %w", err)
	}

	return result.ContainsBibliography, nil
}

func bibliographyPageRange(matches []bool) (startPage, endPage int, ok bool) {
	for i, match := range matches {
		if !match {
			continue
		}
		if !ok {
			startPage = i + 1
			ok = true
		}
		endPage = i + 1
	}
	return startPage, endPage, ok
}
