// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package workflow

import (
	"strings"
	"testing"

	"github.com/sandialabs/bibcheck/lookup"
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
