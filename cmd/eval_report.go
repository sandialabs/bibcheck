// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/eval"
)

func runEvalReportCommand(runID string) error {
	workspaceRoot, err := filepath.Abs(evalWorkspace)
	if err != nil {
		return fmt.Errorf("resolve eval workspace %q: %w", evalWorkspace, err)
	}

	workspace := eval.NewWorkspace(workspaceRoot)
	run, err := workspace.LoadRun(runID)
	if err != nil {
		return err
	}

	report, err := buildSummaryReport(workspace, run)
	if err != nil {
		return err
	}
	if err := workspace.SaveSummaryReport(report); err != nil {
		return err
	}

	rendered, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("render report json: %w", err)
	}
	fmt.Println(string(rendered))
	return nil
}

func buildSummaryReport(workspace eval.Workspace, run *eval.Run) (*eval.SummaryReport, error) {
	report := &eval.SummaryReport{
		FormatVersion: eval.FormatVersion,
		RunID:         run.RunID,
		GeneratedAt:   time.Now().UTC(),
		Papers:        []eval.PaperSummary{},
		Venues:        []eval.VenueSummary{},
	}

	venueIndex := map[string]*eval.VenueSummary{}

	for _, paper := range run.Papers {
		if paper.Status != eval.RunStatusDone {
			continue
		}

		result, err := workspace.LoadPaperResult(run.RunID, paper.PaperID)
		if err != nil {
			return nil, err
		}

		paperSummary := summarizePaperResult(result)
		report.Papers = append(report.Papers, paperSummary)
		accumulatePaperIntoReport(report, paperSummary)

		venueSummary := venueIndex[paperSummary.VenueID]
		if venueSummary == nil {
			report.Venues = append(report.Venues, eval.VenueSummary{
				VenueID: paperSummary.VenueID,
			})
			venueSummary = &report.Venues[len(report.Venues)-1]
			venueIndex[paperSummary.VenueID] = venueSummary
		}
		accumulatePaperIntoVenue(venueSummary, paperSummary)
	}

	finalizeCoverage(&report.ReviewCoverage, totalEntries(report.EntryCounts))
	for idx := range report.Papers {
		finalizeCoverage(&report.Papers[idx].ReviewCoverage, report.Papers[idx].TotalEntries)
	}
	for idx := range report.Venues {
		finalizeCoverage(&report.Venues[idx].ReviewCoverage, report.Venues[idx].TotalEntries)
	}

	slices.SortFunc(report.Papers, func(a, b eval.PaperSummary) int {
		return strings.Compare(a.PaperID, b.PaperID)
	})
	slices.SortFunc(report.Venues, func(a, b eval.VenueSummary) int {
		return strings.Compare(a.VenueID, b.VenueID)
	})

	return report, nil
}

func summarizePaperResult(result *eval.PaperResult) eval.PaperSummary {
	summary := eval.PaperSummary{
		PaperID:      result.PaperID,
		VenueID:      result.VenueID,
		TotalEntries: len(result.Entries),
	}

	for _, entry := range result.Entries {
		accumulateEntryCounts(&summary.EntryCounts, entry.FinalDecision)
		accumulateSummaryState(&summary.SummaryStates, entry.SummaryState)
	}
	finalizeRates(&summary.EntryCounts, summary.TotalEntries, &summary.MatchFoundRate, &summary.NoMatchRate, &summary.ErrorRate)
	finalizeCoverage(&summary.ReviewCoverage, summary.TotalEntries)
	return summary
}

func accumulatePaperIntoReport(report *eval.SummaryReport, paper eval.PaperSummary) {
	report.EntryCounts.MatchFound += paper.EntryCounts.MatchFound
	report.EntryCounts.NoMatch += paper.EntryCounts.NoMatch
	report.EntryCounts.Error += paper.EntryCounts.Error

	report.SummaryStateCounts.OK += paper.SummaryStates.OK
	report.SummaryStateCounts.Review += paper.SummaryStates.Review
	report.SummaryStateCounts.Error += paper.SummaryStates.Error
	report.SummaryStateCounts.Unknown += paper.SummaryStates.Unknown
}

func accumulatePaperIntoVenue(venue *eval.VenueSummary, paper eval.PaperSummary) {
	venue.PaperCount++
	venue.TotalEntries += paper.TotalEntries
	venue.EntryCounts.MatchFound += paper.EntryCounts.MatchFound
	venue.EntryCounts.NoMatch += paper.EntryCounts.NoMatch
	venue.EntryCounts.Error += paper.EntryCounts.Error
	venue.SummaryStates.OK += paper.SummaryStates.OK
	venue.SummaryStates.Review += paper.SummaryStates.Review
	venue.SummaryStates.Error += paper.SummaryStates.Error
	venue.SummaryStates.Unknown += paper.SummaryStates.Unknown
	finalizeRates(&venue.EntryCounts, venue.TotalEntries, &venue.MatchFoundRate, &venue.NoMatchRate, &venue.ErrorRate)
}

func accumulateEntryCounts(counts *eval.DecisionCounts, decision string) {
	switch decision {
	case "match_found":
		counts.MatchFound++
	case "error":
		counts.Error++
	default:
		counts.NoMatch++
	}
}

func accumulateSummaryState(counts *eval.SummaryStateCounts, state string) {
	switch state {
	case "ok":
		counts.OK++
	case "review":
		counts.Review++
	case "error":
		counts.Error++
	default:
		counts.Unknown++
	}
}

func finalizeRates(counts *eval.DecisionCounts, total int, matchFoundRate, noMatchRate, errorRate *float64) {
	if total == 0 {
		return
	}
	*matchFoundRate = float64(counts.MatchFound) / float64(total)
	*noMatchRate = float64(counts.NoMatch) / float64(total)
	*errorRate = float64(counts.Error) / float64(total)
}

func finalizeCoverage(coverage *eval.ReviewCoverage, total int) {
	coverage.UnreviewedEntries = total - coverage.ReviewedEntries
	if total == 0 {
		return
	}
	coverage.ReviewedFraction = float64(coverage.ReviewedEntries) / float64(total)
}

func totalEntries(counts eval.DecisionCounts) int {
	return counts.MatchFound + counts.NoMatch + counts.Error
}
