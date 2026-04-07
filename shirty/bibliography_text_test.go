// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"strings"
	"testing"
)

func TestBibliographyTextReferencesHeading(t *testing.T) {
	text := strings.Join([]string{
		"Title of Paper",
		"Introduction",
		"Some body text.",
		"References",
		"[1] First entry.",
		"[2] Second entry.",
	}, "\n")

	got := bibliographyText(text)

	if !strings.HasPrefix(got, "References\n[1] First entry.") {
		t.Fatalf("expected bibliography slice, got %q", got)
	}
}

func TestBibliographyTextNumberedHeading(t *testing.T) {
	text := strings.Join([]string{
		"Body text",
		"4 References",
		"[1] First entry.",
	}, "\n")

	got := bibliographyText(text)

	if !strings.HasPrefix(got, "4 References\n[1] First entry.") {
		t.Fatalf("expected numbered bibliography heading, got %q", got)
	}
}

func TestBibliographyTextFallsBackToFullText(t *testing.T) {
	text := "Body text only\nNo bibliography heading here."

	got := bibliographyText(text)

	if got != text {
		t.Fatalf("expected full text fallback, got %q", got)
	}
}
