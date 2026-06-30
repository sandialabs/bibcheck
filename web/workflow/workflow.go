// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	analysisrunner "github.com/sandialabs/bibcheck/analysis"
	"github.com/sandialabs/bibcheck/config"
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
	ShirtyBaseURL    string
	OpenRouterAPIKey string
}

type EntryState struct {
	ID             string
	TextStatus     string
	Text           string
	AnalysisStatus string
	SummaryStatus  string
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

type Options struct {
	Entry   int
	Workers int
}

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
	shirtyBaseURL := strings.TrimSpace(keys.ShirtyBaseURL)
	openRouterKey := strings.TrimSpace(keys.OpenRouterAPIKey)

	if shirtyKey != "" {
		if shirtyBaseURL == "" {
			shirtyBaseURL = config.DefaultShirtyBaseURL
		}
		client := shirty.NewWorkflow(shirtyKey, shirtyBaseURL, shirty.WithAuditEnabled(false))
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
	return AnalyzePDFWithOptions(ctx, rt, pdf, Options{}, progress)
}

func AnalyzePDFWithOptions(ctx context.Context, rt *Runtime, pdf []byte, options Options, progress Progress) State {
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

	entryIDs := []int{options.Entry}
	if options.Entry < 1 {
		state.Phase = "Counting entries"
		emit(progress, state)
		count, err := rt.Counter.CountBibliographyEntries(bibliography)
		if err != nil {
			return fail(progress, state, fmt.Errorf("count bibliography entries: %w", err))
		}
		if count < 1 {
			return fail(progress, state, fmt.Errorf("expected at least one bibliography entry, found %d", count))
		}

		entryIDs = make([]int, count)
		for i := range entryIDs {
			entryIDs[i] = i + 1
		}
	}

	state.Total = len(entryIDs)
	state.Phase = "Processing entries"
	runnerResult, err := analysisrunner.Run(ctx, analysisrunner.Config{
		EntryIDs: entryIDs,
		Workers:  options.Workers,
		Extract: func(id int) (string, error) {
			return rt.Provider.EntryFromBibliography(bibliography, id)
		},
		Lookup: func(text string) (*lookup.Result, error) {
			return lookup.Entry(text, "auto", rt.Provider, rt.Provider, rt.Provider, nil)
		},
		Summarize: func(result *lookup.Result) (analysisrunner.Summary, error) {
			mismatch, comment, err := rt.Provider.Summarize(result)
			return analysisrunner.Summary{Mismatch: mismatch, Comment: comment}, err
		},
		Progress: func(snapshot analysisrunner.Snapshot) {
			state = stateFromSnapshot(state, snapshot)
			emit(progress, state)
		},
	})
	state = stateFromSnapshot(state, runnerResult)
	if err != nil {
		return fail(progress, state, err)
	}

	state.Phase = "Done"
	emit(progress, state)
	return state
}

func stateFromSnapshot(state State, snapshot analysisrunner.Snapshot) State {
	state.Completed = snapshot.Completed
	state.Entries = make([]EntryState, len(snapshot.Entries))
	for i, entry := range snapshot.Entries {
		view := EntryState{
			ID:             fmt.Sprintf("%d", entry.ID),
			TextStatus:     webStatus(entry.ExtractionStatus),
			Text:           entry.Text,
			AnalysisStatus: webStatus(entry.LookupStatus),
			SummaryStatus:  webStatus(entry.SummaryStatus),
		}
		if entry.ExtractionError != nil {
			view.Error = fmt.Sprintf("extract entry: %v", entry.ExtractionError)
		} else if entry.LookupError != nil {
			view.Error = fmt.Sprintf("analyze entry: %v", entry.LookupError)
		} else if entry.SummaryError != nil {
			entry.Result.Summary.Error = entry.SummaryError
		}
		if entry.Result != nil {
			if entry.SummaryStatus == analysisrunner.StatusCompleted {
				entry.Result.Summary.Status = lookup.SearchStatusDone
				entry.Result.Summary.Matches = !entry.Summary.Mismatch
				entry.Result.Summary.Comment = entry.Summary.Comment
			}
			view.Analysis = FormatAnalysis(entry.Result)
		}
		state.Entries[i] = view
	}
	return state
}

func webStatus(status analysisrunner.Status) string {
	switch status {
	case analysisrunner.StatusActive:
		return "active"
	case analysisrunner.StatusCompleted:
		return "completed"
	case analysisrunner.StatusError:
		return "error"
	default:
		return "pending"
	}
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
