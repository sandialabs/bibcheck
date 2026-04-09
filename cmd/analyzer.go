// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/sandialabs/bibcheck/summary"
)

type analyzer struct {
	openrouterClient *openrouter.Client
	shirtyProvider   *shirty.Workflow
	elsevierClient   *elsevier.Client
	summarizer       *summary.ShirtySummarizer
	class            entries.Classifier
	entryParser      entries.Parser
	docRawExtract    documents.EntryFromRawExtractor
	docTextExtract   documents.EntryFromTextExtractor
	docMeta          documents.MetaExtractor
}

type preparedDocument struct {
	pdfPath    string
	pdfEncoded string
	pdfText    string
	entryCount int
}

func newAnalyzer(settings config.Settings) (*analyzer, error) {
	a := &analyzer{}

	if settings.OpenRouterAPIKey != "" && settings.OpenRouterBaseURL != "" {
		a.openrouterClient = openrouter.NewClient(
			settings.OpenRouterAPIKey,
			openrouter.WithBaseURL(settings.OpenRouterBaseURL),
		)
	}
	if settings.ShirtyAPIKey != "" && settings.ShirtyBaseURL != "" {
		a.shirtyProvider = shirty.NewWorkflow(
			settings.ShirtyAPIKey,
			shirty.WithBaseUrl(settings.ShirtyBaseURL),
		)
		a.summarizer = summary.NewShirtySummarizer(
			shirty.NewWorkflow(
				settings.ShirtyAPIKey,
				shirty.WithBaseUrl(settings.ShirtyBaseURL),
			),
		)
	}
	if settings.ElsevierAPIKey != "" {
		a.elsevierClient = elsevier.NewClient(settings.ElsevierAPIKey)
	}

	if a.openrouterClient != nil {
		a.class = a.openrouterClient
		a.entryParser = a.openrouterClient
		a.docRawExtract = a.openrouterClient
		a.docMeta = a.openrouterClient
	}
	if a.shirtyProvider != nil {
		a.class = a.shirtyProvider
		a.entryParser = a.shirtyProvider
		a.docTextExtract = a.shirtyProvider
		a.docMeta = a.shirtyProvider
	}

	if a.class == nil || a.entryParser == nil || a.docMeta == nil {
		return nil, errors.New("need shirty or openrouter config")
	}

	return a, nil
}

func (a *analyzer) prepareDocument(pdfPath string, needCount bool) (*preparedDocument, error) {
	doc := &preparedDocument{
		pdfPath: pdfPath,
	}

	if a.openrouterClient != nil {
		pdfEncoded, err := lookup.Encode(pdfPath)
		if err != nil {
			return nil, fmt.Errorf("pdf encode error: %w", err)
		}
		doc.pdfEncoded = pdfEncoded
		if needCount {
			entryCount, err := a.openrouterClient.NumEntries(pdfEncoded)
			if err != nil {
				return nil, fmt.Errorf("bibliography size error: %w", err)
			}
			doc.entryCount = entryCount
		}
		return doc, nil
	}

	if a.shirtyProvider != nil {
		if err := a.ensureText(doc); err != nil {
			return nil, err
		}
		if needCount {
			entryCount, err := a.shirtyProvider.NumBibEntries(doc.pdfText)
			if err != nil {
				return nil, fmt.Errorf("bibliography size error: %w", err)
			}
			doc.entryCount = entryCount
		}
		return doc, nil
	}

	return nil, errors.New("need shirty or openrouter config")
}

func (a *analyzer) analyzeEntry(doc *preparedDocument, entryNumber int) (entryView, error) {
	cfg := &lookup.EntryConfig{
		ElsevierClient: a.elsevierClient,
	}

	var (
		lr  *lookup.Result
		err error
	)
	if a.docRawExtract != nil {
		lr, err = lookup.EntryFromBase64(doc.pdfEncoded, entryNumber, pipeline, a.class, a.docRawExtract, a.docMeta, a.entryParser, cfg)
	} else if a.docTextExtract != nil {
		if err := a.ensureText(doc); err != nil {
			return entryView{}, err
		}
		lr, err = lookup.EntryFromText(doc.pdfText, entryNumber, pipeline, a.class, a.docTextExtract, a.docMeta, a.entryParser, cfg)
	} else {
		return entryView{}, errors.New("requires something that can extract a bib entry from a pdf")
	}
	if err != nil {
		return entryView{}, err
	}

	outcome := summaryOutcome{}
	if a.summarizer != nil {
		mismatch, comment, summaryErr := a.summarizer.Summarize(lr)
		outcome.mismatch = mismatch
		outcome.comment = comment
		outcome.err = summaryErr
		if summaryErr != nil {
			log.Printf("summarizer error: %v", summaryErr)
		}
	}

	return buildEntryView(entryNumber, lr, outcome), nil
}

func (a *analyzer) ensureText(doc *preparedDocument) error {
	if doc.pdfText != "" {
		return nil
	}
	if a.shirtyProvider == nil {
		return errors.New("text extraction requires shirty config")
	}

	textractResp, err := a.shirtyProvider.Textract(doc.pdfPath)
	if err != nil {
		return fmt.Errorf("textract error: %w", err)
	}
	doc.pdfText = textractResp.Text
	return nil
}
