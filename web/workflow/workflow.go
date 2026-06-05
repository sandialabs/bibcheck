// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
)

type ProviderKind string

const (
	ProviderNone       ProviderKind = ""
	ProviderShirty     ProviderKind = "Shirty"
	ProviderOpenRouter ProviderKind = "OpenRouter"
)

type Keys struct {
	ShirtyAPIKey     string
	OpenRouterAPIKey string
}

type EntryState struct {
	ID             string
	TextStatus     string
	Text           string
	AnalysisStatus string
	Analysis       string
	Error          string
}

type State struct {
	Provider  ProviderKind
	Phase     string
	Total     int
	Completed int
	Entries   []EntryState
	Error     string
}

type Progress func(State)

type Provider interface {
	PrepareBibliographyContent([]byte) (*documents.Bibliography, error)
	EntryFromBibliography(*documents.Bibliography, int) (string, error)
	entries.Classifier
	entries.Parser
	documents.MetaExtractor
	Summarize(*lookup.Result) (bool, string, error)
}

type Counter interface {
	CountBibliographyEntries(*documents.Bibliography) (int, error)
}

type Runtime struct {
	Kind     ProviderKind
	Provider Provider
	Counter  Counter
}

type shirtyCounter struct {
	client *shirty.Workflow
}

func (c shirtyCounter) CountBibliographyEntries(b *documents.Bibliography) (int, error) {
	return c.client.NumBibEntries(b)
}

type openRouterCounter struct {
	client *openrouter.Client
}

func (c openRouterCounter) CountBibliographyEntries(b *documents.Bibliography) (int, error) {
	return c.client.NumBibliographyEntries(b)
}

func NewRuntime(keys Keys) (*Runtime, error) {
	shirtyKey := strings.TrimSpace(keys.ShirtyAPIKey)
	openRouterKey := strings.TrimSpace(keys.OpenRouterAPIKey)

	if shirtyKey != "" {
		client := shirty.NewWorkflow(shirtyKey, shirty.WithAuditEnabled(false))
		return &Runtime{
			Kind:     ProviderShirty,
			Provider: client,
			Counter:  shirtyCounter{client: client},
		}, nil
	}

	if openRouterKey != "" {
		client := openrouter.NewClient(openRouterKey)
		return &Runtime{
			Kind:     ProviderOpenRouter,
			Provider: client,
			Counter:  openRouterCounter{client: client},
		}, nil
	}

	return nil, errors.New("provide a Shirty or OpenRouter API key")
}

func AnalyzePDF(ctx context.Context, rt *Runtime, pdf []byte, progress Progress) State {
	if rt == nil || rt.Provider == nil || rt.Counter == nil {
		state := State{Phase: "Starting"}
		return fail(progress, state, errors.New("missing analysis runtime"))
	}

	state := State{
		Provider: rt.Kind,
		Phase:    "Preparing bibliography",
	}
	emit(progress, state)

	if err := ctx.Err(); err != nil {
		return fail(progress, state, err)
	}
	if len(pdf) == 0 {
		return fail(progress, state, errors.New("selected PDF is empty"))
	}

	bibliography, err := rt.Provider.PrepareBibliographyContent(pdf)
	if err != nil {
		return fail(progress, state, fmt.Errorf("prepare bibliography: %w", err))
	}

	state.Phase = "Counting entries"
	emit(progress, state)
	count, err := rt.Counter.CountBibliographyEntries(bibliography)
	if err != nil {
		return fail(progress, state, fmt.Errorf("count bibliography entries: %w", err))
	}
	if count < 1 {
		return fail(progress, state, fmt.Errorf("expected at least one bibliography entry, found %d", count))
	}

	state.Total = count
	state.Entries = make([]EntryState, count)
	for i := range state.Entries {
		state.Entries[i] = EntryState{
			ID:             fmt.Sprintf("%d", i+1),
			TextStatus:     "pending",
			AnalysisStatus: "pending",
		}
	}
	state.Phase = "Extracting entries"
	emit(progress, state)

	for i := range state.Entries {
		if err := ctx.Err(); err != nil {
			return fail(progress, state, err)
		}

		state.Entries[i].TextStatus = "active"
		emit(progress, state)

		text, err := rt.Provider.EntryFromBibliography(bibliography, i+1)
		if err != nil {
			state.Entries[i].TextStatus = "error"
			state.Entries[i].Error = fmt.Sprintf("extract entry: %v", err)
			emit(progress, state)
			continue
		}

		state.Entries[i].TextStatus = "completed"
		state.Entries[i].Text = text
		emit(progress, state)
	}

	state.Phase = "Analyzing entries"
	emit(progress, state)
	for i := range state.Entries {
		if err := ctx.Err(); err != nil {
			return fail(progress, state, err)
		}
		if state.Entries[i].TextStatus != "completed" {
			state.Entries[i].AnalysisStatus = "error"
			if state.Entries[i].Error == "" {
				state.Entries[i].Error = "entry text was not extracted"
			}
			emit(progress, state)
			continue
		}

		state.Entries[i].AnalysisStatus = "active"
		emit(progress, state)

		result, err := lookup.Entry(state.Entries[i].Text, "auto", rt.Provider, rt.Provider, rt.Provider, nil)
		if err != nil {
			state.Entries[i].AnalysisStatus = "error"
			state.Entries[i].Error = fmt.Sprintf("analyze entry: %v", err)
			emit(progress, state)
			continue
		}

		mismatch, comment, summaryErr := rt.Provider.Summarize(result)
		if summaryErr != nil {
			result.Summary.Error = summaryErr
		} else {
			result.Summary.Status = lookup.SearchStatusDone
			result.Summary.Matches = !mismatch
			result.Summary.Comment = comment
		}

		state.Entries[i].AnalysisStatus = "completed"
		state.Entries[i].Analysis = FormatAnalysis(result)
		state.Completed++
		emit(progress, state)
	}

	state.Phase = "Done"
	emit(progress, state)
	return state
}

func FormatAnalysis(result *lookup.Result) string {
	if result == nil {
		return ""
	}

	var b strings.Builder
	if result.Summary.Status == lookup.SearchStatusDone {
		status := "possible mismatch"
		if result.Summary.Matches {
			status = "looks okay"
		}
		fmt.Fprintf(&b, "summary: %s", status)
		if result.Summary.Comment != "" {
			fmt.Fprintf(&b, " - %s", result.Summary.Comment)
		}
		b.WriteString("\n")
	} else if result.Summary.Error != nil {
		fmt.Fprintf(&b, "summary error: %v\n", result.Summary.Error)
	}

	if result.Arxiv.Status == lookup.SearchStatusDone && result.Arxiv.Entry != nil {
		fmt.Fprintf(&b, "arxiv: %s\n", result.Arxiv.Entry.ToString())
	}
	if result.OSTI.Status == lookup.SearchStatusDone && result.OSTI.Record != nil {
		fmt.Fprintf(&b, "OSTI: %s\n", result.OSTI.Record.ToString())
	}
	if result.Crossref.Status == lookup.SearchStatusDone && result.Crossref.Work != nil {
		fmt.Fprintf(&b, "crossref: %s\n", result.Crossref.Work.ToString())
	}
	if result.DOIOrg.Status == lookup.SearchStatusDone && result.DOIOrg.Found {
		fmt.Fprintf(&b, "doi.org: exists\n")
	}
	if result.Online.Status == lookup.SearchStatusDone && result.Online.Metadata != nil {
		fmt.Fprintf(&b, "URL: %s\n", result.Online.Metadata.ToString())
	}
	if b.Len() == 0 {
		return "No matching metadata found."
	}
	return strings.TrimSpace(b.String())
}

func emit(progress Progress, state State) {
	if progress != nil {
		progress(cloneState(state))
	}
}

func fail(progress Progress, state State, err error) State {
	state.Phase = "Error"
	state.Error = err.Error()
	emit(progress, state)
	return state
}

func cloneState(state State) State {
	entries := make([]EntryState, len(state.Entries))
	copy(entries, state.Entries)
	state.Entries = entries
	return state
}
