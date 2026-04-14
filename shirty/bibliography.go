// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewBibliographyPageRF() *openai.ResponseFormat {
	return &openai.ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.BibliographyPageJSONSchema(),
	}
}

func (w *Workflow) PrepareBibliography(filePath string) (*documents.Bibliography, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read pdf error: %w", err)
	}
	return w.PrepareBibliographyContent(data)
}

func (w *Workflow) PrepareBibliographyContent(pdf []byte) (*documents.Bibliography, error) {
	pageCount, err := documents.PDFPageCount(pdf)
	if err != nil {
		return nil, fmt.Errorf("pdf page count error: %w", err)
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("expected pdf to have at least one page")
	}

	matches := make([]bool, pageCount)
	seenBibliography := false
	for page := pageCount; page >= 1; page-- {
		pagePDF, err := documents.PDFSlicePages(pdf, page, page)
		if err != nil {
			return nil, fmt.Errorf("slice page %d error: %w", page, err)
		}
		resp, err := w.TextractContent(pagePDF)
		if err != nil {
			return nil, fmt.Errorf("textract page %d error: %w", page, err)
		}
		match, err := w.pageContainsBibliography(resp.Text)
		if err != nil {
			return nil, fmt.Errorf("page %d bibliography classification error: %w", page, err)
		}
		matches[page-1] = match
		log.Printf("bibliography page %d/%d match=%t", page, pageCount, match)

		if match {
			seenBibliography = true
			continue
		}
		if seenBibliography {
			log.Printf("stopping bibliography detection at page %d/%d after contiguous bibliography block", page, pageCount)
			break
		}
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

	textractResp, err := w.TextractContent(bibPDF)
	if err != nil {
		return nil, fmt.Errorf("textract bibliography pdf error: %w", err)
	}

	return &documents.Bibliography{
		PDF:       bibPDF,
		Text:      textractResp.Text,
		StartPage: startPage,
		EndPage:   endPage,
	}, nil
}

func (w *Workflow) pageContainsBibliography(text string) (bool, error) {
	req := &openai.ChatRequest{
		Model: "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8",
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Determine whether the provided page text contains any part of the paper's bibliography or references section.
- Return true if the page contains a bibliography heading, one or more bibliography entries, or a continuation of bibliography entries from another page.
- Return false otherwise, i.e., for any page that DOES NOT contain any part of a bibliography: body pages, appendices, acknowledgments, author bios, unrelated back matter, etc.
- Produce JSON.`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewBibliographyPageRF(),
		Temperature:    openai.Temperature(0),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return false, fmt.Errorf("chat choice 0 error: %w", err)
	}

	resp := struct {
		ContainsBibliography bool `json:"contains_bibliography"`
	}{}
	if err := json.Unmarshal(content, &resp); err != nil {
		return false, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return resp.ContainsBibliography, nil
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
