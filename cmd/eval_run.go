// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/eval"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/sandialabs/bibcheck/summary"
	"github.com/sandialabs/bibcheck/version"
)

func runEvalCommand() error {
	workspaceRoot, err := filepath.Abs(evalWorkspace)
	if err != nil {
		return fmt.Errorf("resolve eval workspace %q: %w", evalWorkspace, err)
	}

	workspace := eval.NewWorkspace(workspaceRoot)
	corpus, err := workspace.LoadCorpus()
	if err != nil {
		return err
	}
	run := &eval.Run{
		FormatVersion: eval.FormatVersion,
		RunID:         newEvalRunID(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		GitSHA:        version.GitSha(),
		Pipeline:      pipeline,
		CorpusRoot:    corpus.CorpusRoot,
		Papers:        make([]eval.RunPaper, 0, len(corpus.Papers)),
	}

	for _, paper := range corpus.Papers {
		run.Papers = append(run.Papers, eval.RunPaper{
			PaperID:      paper.ID,
			VenueID:      paper.VenueID,
			RelativePath: paper.RelativePath,
			Status:       eval.RunStatusPending,
			ResultPath:   filepath.ToSlash(filepath.Join(eval.ResultsDirName, paper.ID+".json")),
		})
	}
	recomputeRunSummary(run)

	if len(corpus.Papers) == 0 {
		if err := workspace.SaveRun(run); err != nil {
			return err
		}
		fmt.Printf("Created run %s\n", run.RunID)
		fmt.Printf("Processed 0 papers: done=0 error=0\n")
		fmt.Printf("Wrote %s\n", workspace.RunPath(run.RunID))
		return nil
	}

	runner, err := newEvalRunner(config.Runtime())
	if err != nil {
		return err
	}
	run.Providers = eval.RunProviders{
		ShirtyConfigured:     runner.shirtyProvider != nil,
		OpenRouterConfigured: runner.openrouterClient != nil,
		ElsevierConfigured:   runner.elsevierClient != nil,
	}

	if err := workspace.SaveRun(run); err != nil {
		return err
	}

	for idx := range run.Papers {
		paper := &run.Papers[idx]
		now := time.Now().UTC()
		paper.Status = eval.RunStatusRunning
		paper.StartedAt = &now
		paper.Error = ""
		run.UpdatedAt = now
		recomputeRunSummary(run)
		if err := workspace.SaveRun(run); err != nil {
			return err
		}

		result, err := runner.analyzePaper(run.RunID, corpus.CorpusRoot, paper.PaperID, paper.VenueID, paper.RelativePath)
		finishedAt := time.Now().UTC()
		paper.FinishedAt = &finishedAt
		run.UpdatedAt = finishedAt

		if err != nil {
			paper.Status = eval.RunStatusError
			paper.Error = err.Error()
			recomputeRunSummary(run)
			if saveErr := workspace.SaveRun(run); saveErr != nil {
				return saveErr
			}
			log.Printf("eval paper error [%s]: %v", paper.PaperID, err)
			continue
		}

		if err := workspace.SavePaperResult(result); err != nil {
			return err
		}
		paper.Status = eval.RunStatusDone
		paper.Error = ""
		recomputeRunSummary(run)
		if err := workspace.SaveRun(run); err != nil {
			return err
		}
	}

	fmt.Printf("Created run %s\n", run.RunID)
	fmt.Printf("Processed %d papers: done=%d error=%d\n", len(run.Papers), run.StatusSummary.Done, run.StatusSummary.Error)
	fmt.Printf("Wrote %s\n", workspace.RunPath(run.RunID))
	return nil
}

type evalRunner struct {
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

func newEvalRunner(settings config.Settings) (*evalRunner, error) {
	runner := &evalRunner{}

	if settings.OpenRouterAPIKey != "" && settings.OpenRouterBaseURL != "" {
		runner.openrouterClient = openrouter.NewClient(
			settings.OpenRouterAPIKey,
			openrouter.WithBaseURL(settings.OpenRouterBaseURL),
		)
	}
	if settings.ShirtyAPIKey != "" && settings.ShirtyBaseURL != "" {
		runner.shirtyProvider = shirty.NewWorkflow(
			settings.ShirtyAPIKey,
			shirty.WithBaseUrl(settings.ShirtyBaseURL),
		)
		runner.summarizer = summary.NewShirtySummarizer(
			shirty.NewWorkflow(
				settings.ShirtyAPIKey,
				shirty.WithBaseUrl(settings.ShirtyBaseURL),
			),
		)
	}
	if settings.ElsevierAPIKey != "" {
		runner.elsevierClient = elsevier.NewClient(settings.ElsevierAPIKey)
	}

	if runner.openrouterClient != nil {
		runner.class = runner.openrouterClient
		runner.entryParser = runner.openrouterClient
		runner.docRawExtract = runner.openrouterClient
		runner.docMeta = runner.openrouterClient
	}
	if runner.shirtyProvider != nil {
		runner.class = runner.shirtyProvider
		runner.entryParser = runner.shirtyProvider
		runner.docTextExtract = runner.shirtyProvider
		runner.docMeta = runner.shirtyProvider
	}

	if runner.class == nil || runner.entryParser == nil || runner.docMeta == nil {
		return nil, errors.New("need shirty or openrouter config")
	}

	return runner, nil
}

func (r *evalRunner) analyzePaper(runID, corpusRoot, paperID, venueID, relativePath string) (*eval.PaperResult, error) {
	pdfPath := filepath.Join(corpusRoot, filepath.FromSlash(relativePath))
	entryCount, pdfEncoded, pdfText, err := r.entryCount(pdfPath)
	if err != nil {
		return nil, err
	}

	result := &eval.PaperResult{
		FormatVersion: eval.FormatVersion,
		RunID:         runID,
		PaperID:       paperID,
		VenueID:       venueID,
		PDFPath:       pdfPath,
		CompletedAt:   time.Now().UTC(),
		TotalEntries:  entryCount,
		Entries:       make([]eval.EntryResult, 0, entryCount),
	}

	cfg := &lookup.EntryConfig{
		ElsevierClient: r.elsevierClient,
	}

	for i := 1; i <= entryCount; i++ {
		var lr *lookup.Result
		if r.docRawExtract != nil {
			lr, err = lookup.EntryFromBase64(pdfEncoded, i, pipeline, r.class, r.docRawExtract, r.docMeta, r.entryParser, cfg)
		} else if r.docTextExtract != nil {
			lr, err = lookup.EntryFromText(pdfText, i, pipeline, r.class, r.docTextExtract, r.docMeta, r.entryParser, cfg)
		} else {
			return nil, errors.New("requires something that can extract a bib entry from a pdf")
		}

		if err != nil {
			result.Entries = append(result.Entries, eval.EntryResult{
				PaperID:            paperID,
				VenueID:            venueID,
				EntryNumber:        i,
				ExtractedEntryText: "",
				FinalDecision:      "error",
				SummaryState:       string(summaryStateError),
				SummaryComment:     err.Error(),
			})
			continue
		}

		outcome := summaryOutcome{}
		if r.summarizer != nil {
			mismatch, comment, summaryErr := r.summarizer.Summarize(lr)
			outcome.mismatch = mismatch
			outcome.comment = comment
			outcome.err = summaryErr
			if summaryErr != nil {
				log.Printf("summarizer error: %v", summaryErr)
			}
		}

		view := buildEntryView(i, lr, outcome)
		result.Entries = append(result.Entries, toEvalEntryResult(paperID, venueID, view))
	}

	return result, nil
}

func (r *evalRunner) entryCount(pdfPath string) (int, string, string, error) {
	if r.openrouterClient != nil {
		pdfEncoded, err := lookup.Encode(pdfPath)
		if err != nil {
			return 0, "", "", fmt.Errorf("pdf encode error: %w", err)
		}
		entryCount, err := r.openrouterClient.NumEntries(pdfEncoded)
		if err != nil {
			return 0, "", "", fmt.Errorf("bibliography size error: %w", err)
		}
		return entryCount, pdfEncoded, "", nil
	}

	if r.shirtyProvider != nil {
		textractResp, err := r.shirtyProvider.Textract(pdfPath)
		if err != nil {
			return 0, "", "", fmt.Errorf("textract error: %w", err)
		}
		entryCount, err := r.shirtyProvider.NumBibEntries(textractResp.Text)
		if err != nil {
			return 0, "", "", fmt.Errorf("bibliography size error: %w", err)
		}
		return entryCount, "", textractResp.Text, nil
	}

	return 0, "", "", errors.New("need shirty or openrouter config")
}

func toEvalEntryResult(paperID, venueID string, view entryView) eval.EntryResult {
	sources := make([]eval.SourceResult, 0, len(view.sources))
	primarySource := ""
	finalDecision := "no_match"

	for _, source := range view.sources {
		sources = append(sources, eval.SourceResult{
			Name:   source.name,
			Status: source.status,
			Detail: source.detail,
		})
		if primarySource == "" && source.status == "matched" {
			primarySource = source.name
			finalDecision = "match_found"
		}
		if source.status == "error" && finalDecision != "match_found" {
			finalDecision = "error"
		}
	}

	if view.summaryState == summaryStateError && finalDecision != "match_found" {
		finalDecision = "error"
	}

	return eval.EntryResult{
		PaperID:            paperID,
		VenueID:            venueID,
		EntryNumber:        view.number,
		ExtractedEntryText: view.originalText,
		FinalDecision:      finalDecision,
		PrimarySource:      primarySource,
		Sources:            sources,
		SummaryState:       string(view.summaryState),
		SummaryComment:     view.summaryComment,
	}
}

func recomputeRunSummary(run *eval.Run) {
	run.StatusSummary = eval.RunStatusSummary{}
	for _, paper := range run.Papers {
		switch paper.Status {
		case eval.RunStatusPending:
			run.StatusSummary.Pending++
		case eval.RunStatusRunning:
			run.StatusSummary.Running++
		case eval.RunStatusDone:
			run.StatusSummary.Done++
		case eval.RunStatusError:
			run.StatusSummary.Error++
		}
	}
}

func newEvalRunID() string {
	return strings.ToLower(time.Now().UTC().Format("20060102t150405z"))
}
