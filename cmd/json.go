// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import "encoding/json"

type jsonSourceView struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type jsonEntryView struct {
	Number         int              `json:"number"`
	OriginalText   string           `json:"original_text"`
	PrimaryMessage string           `json:"primary_message"`
	PrimarySource  string           `json:"primary_source"`
	SummaryState   summaryState     `json:"summary_state"`
	SummaryComment string           `json:"summary_comment"`
	Sources        []jsonSourceView `json:"sources"`
}

type jsonSummaryCounts struct {
	OK      int `json:"ok"`
	Review  int `json:"review"`
	Error   int `json:"error"`
	Unknown int `json:"unknown"`
}

type jsonDocumentView struct {
	Format             string            `json:"format"`
	TotalEntries       int               `json:"total_entries"`
	ShownEntries       int               `json:"shown_entries"`
	HiddenOKEntries    int               `json:"hidden_ok_entries"`
	SummaryCounts      jsonSummaryCounts `json:"summary_counts"`
	PrimaryMatchCounts map[string]int    `json:"primary_match_counts"`
	Entries            []jsonEntryView   `json:"entries"`
}

func renderJSONDocument(doc documentView, views []entryView, carelessHideOK bool, singleEntry bool) (string, error) {
	payload := jsonDocumentView{
		Format:          string(outputFormatJSON),
		TotalEntries:    doc.total,
		ShownEntries:    doc.shown,
		HiddenOKEntries: doc.hiddenOK,
		SummaryCounts: jsonSummaryCounts{
			OK:      doc.explicitOK,
			Review:  doc.review,
			Error:   doc.errors,
			Unknown: doc.unknown,
		},
		PrimaryMatchCounts: doc.sourceCounts,
		Entries:            []jsonEntryView{},
	}

	for _, view := range views {
		if shouldHideEntry(view, carelessHideOK, singleEntry) {
			continue
		}
		payload.Entries = append(payload.Entries, jsonEntryView{
			Number:         view.number,
			OriginalText:   view.originalText,
			PrimaryMessage: view.primaryMessage,
			PrimarySource:  view.primarySource,
			SummaryState:   view.summaryState,
			SummaryComment: view.summaryComment,
			Sources:        toJSONSources(view.sources),
		})
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

func toJSONSources(sources []sourceView) []jsonSourceView {
	out := make([]jsonSourceView, 0, len(sources))
	for _, source := range sources {
		out = append(out, jsonSourceView{
			Name:   source.name,
			Status: source.status,
			Detail: source.detail,
		})
	}
	return out
}
