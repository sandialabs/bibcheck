// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func DiscoverCorpus(corpusRoot string) (*Corpus, error) {
	absRoot, err := filepath.Abs(corpusRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve corpus root %q: %w", corpusRoot, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat corpus root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("corpus root %q is not a directory", absRoot)
	}

	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, fmt.Errorf("read corpus root %q: %w", absRoot, err)
	}

	corpus := &Corpus{
		FormatVersion: FormatVersion,
		GeneratedAt:   time.Now().UTC(),
		CorpusRoot:    absRoot,
		Venues:        []Venue{},
		Papers:        []PaperRef{},
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		venue, err := discoverVenue(absRoot, entry.Name())
		if err != nil {
			return nil, err
		}
		if len(venue.Papers) == 0 {
			continue
		}

		corpus.Venues = append(corpus.Venues, venue)
		corpus.Papers = append(corpus.Papers, venue.Papers...)
	}

	slices.SortFunc(corpus.Venues, func(a, b Venue) int {
		return strings.Compare(a.ID, b.ID)
	})
	slices.SortFunc(corpus.Papers, func(a, b PaperRef) int {
		return strings.Compare(a.ID, b.ID)
	})

	return corpus, nil
}

func discoverVenue(corpusRoot, venueName string) (Venue, error) {
	venuePath := filepath.Join(corpusRoot, venueName)
	entries, err := os.ReadDir(venuePath)
	if err != nil {
		return Venue{}, fmt.Errorf("read venue %q: %w", venuePath, err)
	}

	venue := Venue{
		ID:           venueName,
		Name:         venueName,
		RelativePath: filepath.ToSlash(venueName),
		Papers:       []PaperRef{},
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			continue
		}

		relativePath := filepath.ToSlash(filepath.Join(venueName, entry.Name()))
		venue.Papers = append(venue.Papers, PaperRef{
			ID:           relativePath,
			VenueID:      venue.ID,
			RelativePath: relativePath,
			FileName:     entry.Name(),
		})
	}

	slices.SortFunc(venue.Papers, func(a, b PaperRef) int {
		return strings.Compare(a.ID, b.ID)
	})

	return venue, nil
}
