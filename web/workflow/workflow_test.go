// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/crossref"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/shirty"
)

func TestNewRuntimeRequiresKey(t *testing.T) {
	if _, err := NewRuntime(Keys{}); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestNewRuntimePrefersShirty(t *testing.T) {
	rt, err := NewRuntime(Keys{
		ShirtyAPIKey:     "shirty",
		OpenRouterAPIKey: "openrouter",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rt.Kind != ProviderShirty {
		t.Fatalf("kind = %q, want %q", rt.Kind, ProviderShirty)
	}
	if rt.CrossrefClient == nil {
		t.Fatal("missing Crossref client")
	}
}

func TestNewRuntimeUsesDefaultShirtyBaseURL(t *testing.T) {
	rt, err := NewRuntime(Keys{ShirtyAPIKey: "shirty"})
	if err != nil {
		t.Fatal(err)
	}
	client := rt.Provider.(*shirty.Workflow)
	if got := client.OpenAIClient().BaseUrl(); got != config.DefaultShirtyBaseURL {
		t.Fatalf("Shirty base URL = %q, want %q", got, config.DefaultShirtyBaseURL)
	}
}

func TestNewRuntimeUsesCustomShirtyBaseURL(t *testing.T) {
	const baseURL = "https://example.invalid/api/v1"
	rt, err := NewRuntime(Keys{
		ShirtyAPIKey:  "shirty",
		ShirtyBaseURL: " " + baseURL + " ",
	})
	if err != nil {
		t.Fatal(err)
	}
	client := rt.Provider.(*shirty.Workflow)
	if got := client.OpenAIClient().BaseUrl(); got != baseURL {
		t.Fatalf("Shirty base URL = %q, want %q", got, baseURL)
	}
}

func TestNewRuntimeUsesOpenRouterWithoutShirty(t *testing.T) {
	rt, err := NewRuntime(Keys{OpenRouterAPIKey: "openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	if rt.Kind != ProviderOpenRouter {
		t.Fatalf("kind = %q, want %q", rt.Kind, ProviderOpenRouter)
	}
}

func TestFormatAnalysisSummary(t *testing.T) {
	result := &lookup.Result{}
	result.Summary.Status = lookup.SearchStatusDone
	result.Summary.Matches = true
	result.Summary.Comment = "metadata agrees"

	got := FormatAnalysis(result)
	if !strings.Contains(got, "summary: looks okay") {
		t.Fatalf("summary missing from %q", got)
	}
	if !strings.Contains(got, "metadata agrees") {
		t.Fatalf("comment missing from %q", got)
	}
}

func TestBuildSummaryViewOKWithComment(t *testing.T) {
	result := &lookup.Result{}
	result.Summary.Status = lookup.SearchStatusDone
	result.Summary.Matches = true
	result.Summary.Comment = "metadata agrees"

	got := BuildSummaryView(result)
	if got.Status != "ok" {
		t.Fatalf("status = %q, want ok", got.Status)
	}
	if got.Comment != "metadata agrees" {
		t.Fatalf("comment = %q, want metadata agrees", got.Comment)
	}
}

func TestBuildSummaryViewError(t *testing.T) {
	result := &lookup.Result{}
	result.Summary.Error = errors.New("summary unavailable")

	got := BuildSummaryView(result)
	if got.Status != "error" {
		t.Fatalf("status = %q, want error", got.Status)
	}
	if got.Comment != "summary unavailable" {
		t.Fatalf("comment = %q, want summary unavailable", got.Comment)
	}
}

func TestBuildLookupCardsDOIFound(t *testing.T) {
	result := &lookup.Result{}
	result.DOIOrg.ID = "10.1234/example"
	result.DOIOrg.Found = true
	result.DOIOrg.Status = lookup.SearchStatusDone

	cards := BuildLookupCards(result)
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
	assertLookupCard(t, cards[0], "DOI", "found", "exists")
}

func TestBuildLookupCardsMatchedSources(t *testing.T) {
	result := &lookup.Result{}
	result.Elsevier.Status = lookup.SearchStatusDone
	result.Elsevier.Result = &elsevier.SearchResult{Title: "A useful paper"}
	result.Crossref.Status = lookup.SearchStatusDone
	result.Crossref.Work = &crossref.CrossrefWork{
		DOI:   "10.1234/match",
		Title: []string{"A useful paper"},
	}

	cards := BuildLookupCards(result)
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	assertLookupCard(t, cards[0], "Elsevier", "matched", "A useful paper")
	assertLookupCard(t, cards[1], "Crossref", "matched", "A useful paper")
}

func TestBuildLookupCardsExplicitNoMatch(t *testing.T) {
	result := &lookup.Result{}
	result.Elsevier.Status = lookup.SearchStatusDone
	result.Crossref.Status = lookup.SearchStatusDone
	result.Crossref.Comment = "no confident match"

	cards := BuildLookupCards(result)
	if len(cards) != 2 {
		t.Fatalf("len(cards) = %d, want 2", len(cards))
	}
	assertLookupCard(t, cards[0], "Elsevier", "no-match", "")
	assertLookupCard(t, cards[1], "Crossref", "no-match", "no confident match")
}

func TestBuildLookupCardsError(t *testing.T) {
	result := &lookup.Result{}
	result.DOIOrg.ID = "10.1234/example"
	result.DOIOrg.Error = errors.New("resolver failed")

	cards := BuildLookupCards(result)
	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}
	assertLookupCard(t, cards[0], "DOI", "error", "resolver failed")
}

func TestBuildLookupCardsOmitsSkippedMethods(t *testing.T) {
	cards := BuildLookupCards(&lookup.Result{})
	if len(cards) != 0 {
		t.Fatalf("len(cards) = %d, want 0", len(cards))
	}
}

func assertLookupCard(t *testing.T, got LookupCard, name, status, detailContains string) {
	t.Helper()
	if got.Name != name {
		t.Fatalf("name = %q, want %q", got.Name, name)
	}
	if got.Status != status {
		t.Fatalf("status = %q, want %q", got.Status, status)
	}
	if detailContains != "" && !strings.Contains(got.Detail, detailContains) {
		t.Fatalf("detail = %q, want to contain %q", got.Detail, detailContains)
	}
	if detailContains == "" && got.Detail != "" {
		t.Fatalf("detail = %q, want empty", got.Detail)
	}
}
