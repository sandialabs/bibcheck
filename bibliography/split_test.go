// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package bibliography

import "testing"

func assertEntries(t *testing.T, got, want []SplitEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d entries, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d: expected %+v, got %+v", i, want[i], got[i])
		}
	}
}

func TestSplitEntriesLineStartNumericMarkers(t *testing.T) {
	input := "[1] A. Author, Some Title,\n    arXiv:2103.11991, 2021.\n[2] B. Author, Another Title, 2017.\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "A. Author, Some Title,\n    arXiv:2103.11991, 2021."},
		{ID: 2, Key: "2", Text: "B. Author, Another Title, 2017."},
	})
}

func TestSplitEntriesInlineSingleLine(t *testing.T) {
	input := "[1] Authors, 2021, arXiv:2103.11991 [2] Authors, 2017, https://arxiv.org/abs/1706.03762"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "Authors, 2021, arXiv:2103.11991"},
		{ID: 2, Key: "2", Text: "Authors, 2017, https://arxiv.org/abs/1706.03762"},
	})
}

func TestSplitEntriesInlineCitationNotSplit(t *testing.T) {
	// "[5]" breaks the +1 chain, so it stays inside entry 2
	input := "[1] First entry [2] Second entry, see also [5] for details [3] Third entry"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "First entry"},
		{ID: 2, Key: "2", Text: "Second entry, see also [5] for details"},
		{ID: 3, Key: "3", Text: "Third entry"},
	})
}

func TestSplitEntriesLineStartCitationNotSplit(t *testing.T) {
	input := "[1] First entry, see\n[3] mid-text does not match because markers use line starts only when wrapped\n[2] Second entry"
	got := SplitEntries(input)
	// [3] is at a line start, so it does split; captured IDs are used as-is
	assertEntries(t, got, []SplitEntry{
		{ID: 1, Key: "1", Text: "First entry, see"},
		{ID: 3, Key: "3", Text: "mid-text does not match because markers use line starts only when wrapped"},
		{ID: 2, Key: "2", Text: "Second entry"},
	})
}

func TestSplitEntriesStripsHeadingPreamble(t *testing.T) {
	input := "Body text about things.\n\nReferences\n[1] First entry\n[2] Second entry\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "First entry"},
		{ID: 2, Key: "2", Text: "Second entry"},
	})
}

func TestSplitEntriesAlphanumericKeys(t *testing.T) {
	input := "[Smith97] J. Smith, Old Paper, 1997.\n[GKS+04] Several Authors, Group Paper, 2004.\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "Smith97", Text: "J. Smith, Old Paper, 1997."},
		{ID: 2, Key: "GKS+04", Text: "Several Authors, Group Paper, 2004."},
	})
}

func TestSplitEntriesBlankLineBlocks(t *testing.T) {
	input := "A. Author, First Title,\n2021.\n\nB. Author, Second Title,\n2022.\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Text: "A. Author, First Title,\n2021."},
		{ID: 2, Text: "B. Author, Second Title,\n2022."},
	})
}

func TestSplitEntriesPerLineFallback(t *testing.T) {
	input := "A. Author, First Title, 2021.\nB. Author, Second Title, 2022.\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Text: "A. Author, First Title, 2021."},
		{ID: 2, Text: "B. Author, Second Title, 2022."},
	})
}

func TestSplitEntriesSingleWrappedEntryWithMarker(t *testing.T) {
	input := "[1] A. Author, Only Entry,\n    arXiv:2103.11991, 2021."
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "A. Author, Only Entry,\n    arXiv:2103.11991, 2021."},
	})
}

func TestSplitEntriesSingleEntryNoMarker(t *testing.T) {
	input := "A. Author, Only Entry, 2021."
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Text: "A. Author, Only Entry, 2021."},
	})
}

func TestSplitEntriesMarkersNotStartingAtOne(t *testing.T) {
	input := "[7] Seventh entry\n[8] Eighth entry\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 7, Key: "7", Text: "Seventh entry"},
		{ID: 8, Key: "8", Text: "Eighth entry"},
	})
}

func TestSplitEntriesDuplicateMarkersFallBackToOrdinals(t *testing.T) {
	input := "[1] First entry\n[1] Also first, apparently\n[2] Second entry\n"
	assertEntries(t, SplitEntries(input), []SplitEntry{
		{ID: 1, Key: "1", Text: "First entry"},
		{ID: 2, Key: "1", Text: "Also first, apparently"},
		{ID: 3, Key: "2", Text: "Second entry"},
	})
}

func TestSplitEntriesEmptyInput(t *testing.T) {
	if got := SplitEntries("  \n \n"); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
