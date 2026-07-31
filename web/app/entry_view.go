// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"github.com/maxence-charriere/go-app/v11/pkg/app"
	"github.com/sandialabs/bibcheck/web/workflow"
)

func renderEntry(entry workflow.EntryState) app.UI {
	status := entryStatus(entry)
	return app.Article().Class("entry-card").Body(
		app.Header().Body(
			app.H2().Body(
				app.Text("Entry "+entry.ID),
			),
			app.Span().Class("status", statusClass(status)).Body(
				app.Text(status),
			),
		),
		app.Div().Class("entry-columns").Body(
			renderExtractedText(entry),
			renderLookupCards(entry),
			renderSummary(entry),
		),
	)
}

func renderExtractedText(entry workflow.EntryState) app.UI {
	return app.Div().Class("entry-pane", "entry-pane-text", panelStatusClass(entry.TextStatus)).Body(
		app.H3().Body(
			app.Text("Extracted entry"),
		),
		app.Pre().Body(
			app.Text(nonEmpty(entry.Text, statusCopy(entry.TextStatus))),
		),
	)
}

func renderLookupCards(entry workflow.EntryState) app.UI {
	return app.Div().Class("entry-pane", "lookup-pane", panelStatusClass(entry.AnalysisStatus)).Body(
		app.H3().Body(
			app.Text("Lookups"),
		),
		renderLookupCardList(entry),
	)
}

func renderLookupCardList(entry workflow.EntryState) app.UI {
	if len(entry.LookupCards) == 0 {
		return app.Div().Class("empty-card").Body(
			app.Text(lookupFallback(entry)),
		)
	}
	cards := make([]app.UI, 0, len(entry.LookupCards))
	for _, card := range entry.LookupCards {
		cards = append(cards, renderLookupCard(card))
	}
	return app.Div().Class("lookup-cards").Body(cards...)
}

func renderLookupCard(card workflow.LookupCard) app.UI {
	contents := []app.UI{
		app.Div().Class("lookup-card-header").Body(
			app.Strong().Body(
				app.Text(card.Name),
			),
			app.Span().Class("lookup-status").Body(
				app.Text(statusLabel(card.Status)),
			),
		),
	}
	if card.Detail != "" {
		contents = append(contents,
			app.Pre().Body(
				app.Text(card.Detail),
			),
		)
	}
	return app.Div().Class("lookup-card", lookupStatusClass(card.Status)).Body(contents...)
}

func renderSummary(entry workflow.EntryState) app.UI {
	summary := entry.Summary
	if summary.Status == "" {
		summary.Status = entry.AnalysisStatus
	}
	return app.Div().Class("entry-pane", "summary-pane", summaryStatusClass(summary.Status)).Body(
		app.H3().Body(
			app.Text("Analysis summary"),
		),
		renderSummaryCard(entry, summary),
	)
}

func renderSummaryCard(entry workflow.EntryState, summary workflow.SummaryView) app.UI {
	if summary.Status == "pending" {
		return app.Div().Class("empty-card").Body(
			app.Text(statusCopy(summary.Status)),
		)
	}
	return app.Div().Class("summary-card", summaryStatusClass(summary.Status)).Body(
		app.Div().Class("summary-card-header").Body(
			app.Strong().Body(
				app.Text(summaryTitle(summary.Status)),
			),
		),
		app.Pre().Body(
			app.Text(nonEmpty(summary.Comment, summaryFallback(entry, summary.Status))),
		),
	)
}
