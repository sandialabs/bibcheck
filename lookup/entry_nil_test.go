// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup_test

import (
	"testing"

	"github.com/sandialabs/bibcheck/lookup"
)

// Entry must not panic when no LLM-backed parser or metadata extractor is
// configured, e.g. the key-free `bibcheck text` path.
func Test_Entry_NilParserSkipsOnline(t *testing.T) {
	text := "A. Author, Some Title Without Any Identifiers, Some Venue, 2021."

	EA, err := lookup.Entry(text, "", nil, nil, nil, &lookup.EntryConfig{})
	if err != nil {
		t.Fatalf("Entry error: %v", err)
	}
	if EA.Online.Status != lookup.SearchStatusNotAttempted {
		t.Fatalf("expected online lookup not attempted, got status %q", EA.Online.Status)
	}
	if EA.Online.Error != nil {
		t.Fatalf("expected no online error, got %v", EA.Online.Error)
	}
}
