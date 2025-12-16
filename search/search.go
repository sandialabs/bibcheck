// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package search

import "github.com/sandialabs/bibcheck/entries"

type Searcher interface {
	SearchEntry(text string) (bool, string, error)

	// search for existence of website
	// returns (exists, comment, error)
	SearchOnline(website *entries.Online) (bool, string, error)

	// search for existence of software
	// returns (exists, comment, error)
	SearchSoftware(software *entries.Software) (bool, string, error)
}
