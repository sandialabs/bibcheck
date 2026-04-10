// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package bibliography

import "testing"

func TestReduceTextTrimsToBibliographyHeading(t *testing.T) {
	input := "Introduction\nBody\nReferences\n[1] First entry\n[2] Second entry\n"
	got := ReduceText(input)
	want := "References\n[1] First entry\n[2] Second entry"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestReduceTextFallsBackToTrimmedInput(t *testing.T) {
	input := "  No bibliography heading here\n[1] Maybe content\n  "
	got := ReduceText(input)
	want := "No bibliography heading here\n[1] Maybe content"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
