// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package bibliography

import (
	"regexp"
	"strconv"
	"strings"
)

// SplitEntry is one bibliography entry recovered from raw text.
type SplitEntry struct {
	ID   int    // numeric marker value when available, else ordinal (1-based)
	Key  string // raw marker key, e.g. "1" or "Smith97"; "" when no marker present
	Text string // trimmed entry text, may span multiple lines
}

var (
	lineNumericMarker = regexp.MustCompile(`(?m)^[ \t]*\[(\d{1,4})\]`)
	inlineNumericRe   = regexp.MustCompile(`\[(\d{1,4})\]`)
	lineAlphaMarker   = regexp.MustCompile(`(?m)^[ \t]*\[([A-Za-z][A-Za-z0-9:+./-]{0,30})\]`)
	blankLineRe       = regexp.MustCompile(`\n[ \t]*\n+`)
)

// SplitEntries splits raw bibliography text into individual entries without
// any LLM assistance. The text is first passed through ReduceText, then split
// by the first strategy that succeeds:
//
//  1. numeric bracket markers at line starts, e.g. "[1] ...\n[2] ..."
//  2. inline numeric bracket markers forming a +1 chain, for single-line
//     input like "[1] ... [2] ...". An inline citation that happens to equal
//     the next expected number and appears before the real marker will
//     mis-split; sequence-breaking citations (e.g. "[12]" inside entry 2)
//     are ignored correctly.
//  3. alphanumeric bracket keys at line starts, e.g. "[Smith97] ..."
//  4. blank-line separated blocks
//  5. one entry per non-empty line
//
// Marker text is stripped from Text and preserved in Key. IDs come from the
// numeric markers when they are unique, otherwise entries are numbered in
// order of appearance.
func SplitEntries(text string) []SplitEntry {
	text = ReduceText(text)
	// ReduceText keeps the heading line; drop it so markers anchor at offset 0
	if first, rest, found := strings.Cut(text, "\n"); found && heading.MatchString(strings.TrimSpace(first)) {
		text = strings.TrimSpace(rest)
	}
	if text == "" {
		return nil
	}

	lineMatches := lineNumericMarker.FindAllStringSubmatchIndex(text, -1)
	if len(lineMatches) >= 2 {
		return splitAtMarkers(text, lineMatches)
	}
	if entries := splitAtMarkers(text, inlineNumericChain(text)); len(entries) >= 2 {
		return entries
	}
	// a single marker anchored at the very start still wins, so a lone
	// wrapped entry like "[1] Foo,\n  bar" is not split at line breaks
	if len(lineMatches) == 1 && lineMatches[0][0] == 0 {
		return splitAtMarkers(text, lineMatches)
	}
	if matches := lineAlphaMarker.FindAllStringSubmatchIndex(text, -1); len(matches) >= 2 ||
		(len(matches) == 1 && matches[0][0] == 0) {
		return splitAtMarkers(text, matches)
	}
	if entries := splitBlocks(blankLineRe.Split(text, -1)); len(entries) >= 2 {
		return entries
	}
	if entries := splitBlocks(strings.Split(text, "\n")); len(entries) >= 2 {
		return entries
	}
	return []SplitEntry{{ID: 1, Text: text}}
}

// inlineNumericChain finds inline numeric markers forming a sequential +1
// chain: the first marker anywhere in the text is accepted, then the earliest
// later marker numbered prev+1, and so on.
func inlineNumericChain(text string) [][]int {
	matches := inlineNumericRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) < 2 {
		return nil
	}
	chain := [][]int{matches[0]}
	prev, _ := strconv.Atoi(text[matches[0][2]:matches[0][3]])
	for _, m := range matches[1:] {
		n, _ := strconv.Atoi(text[m[2]:m[3]])
		if n == prev+1 {
			chain = append(chain, m)
			prev = n
		}
	}
	return chain
}

// splitAtMarkers cuts text at each regex match, stripping the marker itself.
// Text before the first marker is treated as preamble and discarded. Numeric
// keys become entry IDs when unique; otherwise IDs are ordinal.
func splitAtMarkers(text string, matches [][]int) []SplitEntry {
	if len(matches) == 0 {
		return nil
	}

	entries := make([]SplitEntry, 0, len(matches))
	for i, m := range matches {
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := strings.TrimSpace(text[m[1]:end])
		if body == "" {
			continue
		}
		entries = append(entries, SplitEntry{Key: text[m[2]:m[3]], Text: body})
	}

	ids := make(map[int]bool, len(entries))
	numeric := true
	for _, e := range entries {
		n, err := strconv.Atoi(e.Key)
		if err != nil || ids[n] {
			numeric = false
			break
		}
		ids[n] = true
	}
	for i := range entries {
		if numeric {
			entries[i].ID, _ = strconv.Atoi(entries[i].Key)
		} else {
			entries[i].ID = i + 1
		}
	}
	return entries
}

func splitBlocks(blocks []string) []SplitEntry {
	entries := []SplitEntry{}
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		entries = append(entries, SplitEntry{ID: len(entries) + 1, Text: block})
	}
	return entries
}
