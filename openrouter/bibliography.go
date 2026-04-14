// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/schema"
)

const (
	defaultBibliographyPageModel  = "google/gemini-2.5-flash"
	defaultBibliographyPagePrompt = `Determine whether the provided PDF page contains any part of the paper's bibliography or references section.
- Return true if the page contains a bibliography heading, one or more bibliography entries, or a continuation of bibliography entries from another page.
- Return false otherwise, i.e., for any page that DOES NOT contain any part of a bibliography: body pages, appendices, acknowledgments, author bios, unrelated back matter, etc.
- Produce JSON.`
)

type BibliographyPageDetectorConfig struct {
	Model            string
	Prompt           string
	PDFEngine        *PDFEngine
	ReasoningEnabled *bool
}

type BibliographyPageDetection struct {
	Page                 int
	ContainsBibliography bool
	Latency              time.Duration
	Usage                *ChatUsage
}

func DefaultBibliographyPageDetectorConfig() BibliographyPageDetectorConfig {
	return BibliographyPageDetectorConfig{
		Model:  defaultBibliographyPageModel,
		Prompt: defaultBibliographyPagePrompt,
	}
}

func (cfg BibliographyPageDetectorConfig) withDefaults() BibliographyPageDetectorConfig {
	defaults := DefaultBibliographyPageDetectorConfig()
	if cfg.Model == "" {
		cfg.Model = defaults.Model
	}
	if cfg.Prompt == "" {
		cfg.Prompt = defaults.Prompt
	}
	return cfg
}

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
	matches, err := c.DetectBibliographyPages(pdf, DefaultBibliographyPageDetectorConfig())
	if err != nil {
		return nil, err
	}

	flags := make([]bool, len(matches))
	for i, match := range matches {
		flags[i] = match.ContainsBibliography
	}

	return bibliographyFromMatches(pdf, flags)
}

func (c *Client) DetectBibliographyPages(pdf []byte, cfg BibliographyPageDetectorConfig) ([]BibliographyPageDetection, error) {
	pageCount, err := documents.PDFPageCount(pdf)
	if err != nil {
		return nil, fmt.Errorf("pdf page count error: %w", err)
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("expected pdf to have at least one page")
	}

	cfg = cfg.withDefaults()
	matches := make([]BibliographyPageDetection, pageCount)
	seenBibliography := false
	for page := pageCount; page >= 1; page-- {
		pagePDF, err := documents.PDFSlicePages(pdf, page, page)
		if err != nil {
			return nil, fmt.Errorf("slice page %d error: %w", page, err)
		}
		match, err := c.pageContainsBibliography(pagePDF, cfg)
		if err != nil {
			return nil, fmt.Errorf("page %d bibliography classification error: %w", page, err)
		}
		match.Page = page
		matches[page-1] = match
		log.Printf("bibliography page %d/%d match=%t", page, pageCount, match.ContainsBibliography)

		if match.ContainsBibliography {
			seenBibliography = true
			continue
		}
		if seenBibliography {
			log.Printf("stopping bibliography detection at page %d/%d after contiguous bibliography block", page, pageCount)
			break
		}
	}

	return matches, nil
}

func bibliographyFromMatches(pdf []byte, matches []bool) (*documents.Bibliography, error) {
	startPage, endPage, ok := bibliographyPageRange(matches)
	bibPDF := pdf
	var err error
	if !ok {
		startPage = 1
		endPage = len(matches)
		log.Printf("bibliography pages not detected; falling back to full pdf (%d pages)", len(matches))
	} else {
		bibPDF, err = documents.PDFSlicePages(pdf, startPage, endPage)
		if err != nil {
			return nil, fmt.Errorf("slice bibliography pages %d-%d error: %w", startPage, endPage, err)
		}
		log.Printf("bibliography pages detected: %d-%d of %d", startPage, endPage, len(matches))
	}

	return &documents.Bibliography{
		PDF:       bibPDF,
		StartPage: startPage,
		EndPage:   endPage,
	}, nil
}

func (c *Client) pageContainsBibliography(pagePDF []byte, cfg BibliographyPageDetectorConfig) (BibliographyPageDetection, error) {
	req := ChatRequest{
		Model: cfg.Model,
		Messages: []Message{
			systemString(cfg.Prompt),
			userBase64File(base64.StdEncoding.EncodeToString(pagePDF)),
		},
		ResponseFormat: NewBibliographyPageResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
	}
	if cfg.PDFEngine != nil {
		req.Plugins = PDFParserPlugins(*cfg.PDFEngine)
	}
	if cfg.ReasoningEnabled != nil {
		if req.Reasoning == nil {
			req.Reasoning = &Reasoning{}
		}
		req.Reasoning.Enabled = new(bool)
		*req.Reasoning.Enabled = true
	}

	result := struct {
		ContainsBibliography bool `json:"contains_bibliography"`
	}{}
	start := time.Now()
	resp, err := c.ChatCompletion(req, c.baseUrl)
	latency := time.Since(start)
	if err != nil {
		return BibliographyPageDetection{}, fmt.Errorf("pageContainsBibliography error: %w", err)
	}

	content, err := choiceZeroString(resp)
	if err != nil {
		return BibliographyPageDetection{}, err
	}
	if err := unmarshalStructuredContent(content, &result); err != nil {
		return BibliographyPageDetection{}, err
	}

	return BibliographyPageDetection{
		ContainsBibliography: result.ContainsBibliography,
		Latency:              latency,
		Usage:                resp.Usage,
	}, nil
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
