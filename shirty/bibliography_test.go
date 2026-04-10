// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import "testing"

func TestBibliographyPageRange(t *testing.T) {
	start, end, ok := bibliographyPageRange([]bool{false, true, false, true, false})
	if !ok {
		t.Fatalf("expected bibliography range")
	}
	if start != 2 || end != 4 {
		t.Fatalf("expected range 2-4, got %d-%d", start, end)
	}
}

func TestBibliographyPageRangeNoMatches(t *testing.T) {
	start, end, ok := bibliographyPageRange([]bool{false, false, false})
	if ok {
		t.Fatalf("expected no bibliography range, got %d-%d", start, end)
	}
	if start != 0 || end != 0 {
		t.Fatalf("expected zero range, got %d-%d", start, end)
	}
}
