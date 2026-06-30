// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/sandialabs/bibcheck/web/workflow"
)

func renderEntry(entry workflow.EntryState) vecty.ComponentOrHTML {
	return elem.Article(
		vecty.Markup(vecty.Class("entry-card")),
		elem.Header(
			elem.Heading2(vecty.Text("Entry "+entry.ID)),
			elem.Span(vecty.Markup(vecty.Class("status", statusClass(entry.AnalysisStatus))), vecty.Text(statusText(entry))),
		),
		elem.Div(
			vecty.Markup(vecty.Class("entry-columns")),
			renderExtractedText(entry),
			renderLookupCards(entry),
			renderSummary(entry),
		),
	)
}

func renderExtractedText(entry workflow.EntryState) vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Class("entry-pane", "entry-pane-text", panelStatusClass(entry.TextStatus))),
		elem.Heading3(vecty.Text("Extracted entry")),
		elem.Preformatted(vecty.Text(nonEmpty(entry.Text, statusCopy(entry.TextStatus)))),
	)
}

func renderLookupCards(entry workflow.EntryState) vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Class("entry-pane", "lookup-pane", panelStatusClass(entry.AnalysisStatus))),
		elem.Heading3(vecty.Text("Lookups")),
		renderLookupCardList(entry),
	)
}

func renderLookupCardList(entry workflow.EntryState) vecty.MarkupOrChild {
	if len(entry.LookupCards) == 0 {
		return elem.Div(vecty.Markup(vecty.Class("empty-card")), vecty.Text(lookupFallback(entry)))
	}

	cards := make(vecty.List, 0, len(entry.LookupCards))
	for _, card := range entry.LookupCards {
		cards = append(cards, renderLookupCard(card))
	}
	return elem.Div(vecty.Markup(vecty.Class("lookup-cards")), cards)
}

func renderLookupCard(card workflow.LookupCard) vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(vecty.Class("lookup-card", lookupStatusClass(card.Status))),
		elem.Div(
			vecty.Markup(vecty.Class("lookup-card-header")),
			elem.Strong(vecty.Text(card.Name)),
			elem.Span(vecty.Markup(vecty.Class("lookup-status")), vecty.Text(statusLabel(card.Status))),
		),
		vecty.If(card.Detail != "",
			elem.Preformatted(vecty.Text(card.Detail)),
		),
	)
}

func renderSummary(entry workflow.EntryState) vecty.ComponentOrHTML {
	summary := entry.Summary
	if summary.Status == "" {
		summary.Status = entry.AnalysisStatus
	}
	return elem.Div(
		vecty.Markup(vecty.Class("entry-pane", "summary-pane", summaryStatusClass(summary.Status))),
		elem.Heading3(vecty.Text("Analysis summary")),
		elem.Div(
			vecty.Markup(vecty.Class("summary-card", summaryStatusClass(summary.Status))),
			elem.Div(
				vecty.Markup(vecty.Class("summary-card-header")),
				elem.Strong(vecty.Text(summaryTitle(summary.Status))),
			),
			elem.Preformatted(vecty.Text(nonEmpty(summary.Comment, summaryFallback(entry, summary.Status)))),
		),
	)
}
