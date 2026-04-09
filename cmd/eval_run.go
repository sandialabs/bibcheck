// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/eval"
	"github.com/sandialabs/bibcheck/version"
)

func runEvalCommand() error {
	workspaceRoot, err := filepath.Abs(evalWorkspace)
	if err != nil {
		return fmt.Errorf("resolve eval workspace %q: %w", evalWorkspace, err)
	}

	workspace := eval.NewWorkspace(workspaceRoot)
	run, resumed, err := prepareEvalRun(workspace)
	if err != nil {
		return err
	}

	normalizeResumablePapers(run)
	recomputeRunSummary(run)
	if err := workspace.SaveRun(run); err != nil {
		return err
	}

	selected := selectRunnablePapers(run, evalRetryErrors)
	if len(selected) == 0 {
		if resumed {
			fmt.Printf("Resumed run %s\n", run.RunID)
		} else {
			fmt.Printf("Created run %s\n", run.RunID)
		}
		fmt.Printf("Processed 0 papers: done=%d error=%d\n", run.StatusSummary.Done, run.StatusSummary.Error)
		fmt.Printf("Wrote %s\n", workspace.RunPath(run.RunID))
		return nil
	}

	analyzer, err := newAnalyzer(config.Runtime())
	if err != nil {
		return err
	}
	run.Providers = eval.RunProviders{
		ShirtyConfigured:     analyzer.shirtyProvider != nil,
		OpenRouterConfigured: analyzer.openrouterClient != nil,
		ElsevierConfigured:   analyzer.elsevierClient != nil,
	}

	if err := workspace.SaveRun(run); err != nil {
		return err
	}

	processedThisInvocation := 0
	for _, idx := range selected {
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

		result, err := analyzeEvalPaper(analyzer, run.RunID, run.CorpusRoot, paper.PaperID, paper.VenueID, paper.RelativePath)
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
		processedThisInvocation++
	}

	if resumed {
		fmt.Printf("Resumed run %s\n", run.RunID)
	} else {
		fmt.Printf("Created run %s\n", run.RunID)
	}
	fmt.Printf("Processed %d papers: done=%d error=%d\n", processedThisInvocation, run.StatusSummary.Done, run.StatusSummary.Error)
	fmt.Printf("Wrote %s\n", workspace.RunPath(run.RunID))
	return nil
}

func prepareEvalRun(workspace eval.Workspace) (*eval.Run, bool, error) {
	if evalResumeRun != "" {
		run, err := workspace.LoadRun(evalResumeRun)
		if err != nil {
			return nil, false, err
		}
		return run, true, nil
	}

	corpus, err := workspace.LoadCorpus()
	if err != nil {
		return nil, false, err
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
	return run, false, nil
}

func normalizeResumablePapers(run *eval.Run) {
	for idx := range run.Papers {
		paper := &run.Papers[idx]
		if paper.Status == eval.RunStatusRunning {
			paper.Status = eval.RunStatusPending
			paper.FinishedAt = nil
		}
	}
}

func selectRunnablePapers(run *eval.Run, retryErrors bool) []int {
	selected := []int{}
	for idx, paper := range run.Papers {
		switch paper.Status {
		case eval.RunStatusPending:
			selected = append(selected, idx)
		case eval.RunStatusError:
			if retryErrors {
				selected = append(selected, idx)
			}
		}
	}
	return selected
}

func analyzeEvalPaper(analyzer *analyzer, runID, corpusRoot, paperID, venueID, relativePath string) (*eval.PaperResult, error) {
	pdfPath := filepath.Join(corpusRoot, filepath.FromSlash(relativePath))
	prepared, err := analyzer.prepareDocument(pdfPath, true)
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
		TotalEntries:  prepared.entryCount,
		Entries:       make([]eval.EntryResult, 0, prepared.entryCount),
	}

	for i := 1; i <= prepared.entryCount; i++ {
		view, err := analyzer.analyzeEntry(prepared, i)
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
		result.Entries = append(result.Entries, toEvalEntryResult(paperID, venueID, view))
	}

	return result, nil
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
