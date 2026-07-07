// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"net/url"
	"strings"
	"testing"
)

func TestEncodeQueryLeavesMailtoAtSignUnescaped(t *testing.T) {
	params := url.Values{
		"mailto":              {"user@example.com"},
		"query.bibliographic": {"author@example.com"},
		"rows":                {"2"},
	}

	query := encodeQuery(params)

	if !strings.Contains(query, "mailto=user@example.com") {
		t.Fatalf("mailto @ is encoded in query %q", query)
	}
	if !strings.Contains(query, "query.bibliographic=author%40example.com") {
		t.Fatalf("non-mailto @ is not encoded in query %q", query)
	}
}
