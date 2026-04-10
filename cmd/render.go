// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"strings"

	prettytext "github.com/jedib0t/go-pretty/v6/text"
	"github.com/sandialabs/bibcheck/lookup"
)

type summaryState string

const (
	summaryStateError   summaryState = "error"
	summaryStateOK      summaryState = "ok"
	summaryStateReview  summaryState = "review"
	summaryStateUnknown summaryState = "unknown"
)

type summaryOutcome struct {
	mismatch bool
	comment  string
	err      error
}

type sourceView struct {
	name   string
	status string
	detail string
}

// entryView is where rlookup.Result data gets translated into a display-oriented form.
type entryView struct {
	number         int
	originalText   string
	summaryState   summaryState
	summaryComment string
	sources        []sourceView
}

type documentView struct {
	total      int
	shown      int
	hiddenOK   int
	review     int
	errors     int
	explicitOK int
	unknown    int
}

func buildEntryView(number int, lr *lookup.Result, outcome summaryOutcome) entryView {
	view := entryView{
		number:       number,
		originalText: lr.Text,
		sources: []sourceView{
			buildDOISourceView(lr),
			buildOSTISourceView(lr),
			buildArxivSourceView(lr),
			buildElsevierSourceView(lr),
			buildCrossrefSourceView(lr),
			buildOnlineSourceView(lr),
		},
	}

	switch {
	case outcome.err != nil:
		view.summaryState = summaryStateError
		view.summaryComment = fmt.Sprintf("summary error: %v", outcome.err)
	case outcome.comment != "":
		view.summaryComment = outcome.comment
		if outcome.mismatch {
			view.summaryState = summaryStateReview
		} else {
			view.summaryState = summaryStateOK
		}
	default:
		view.summaryState = deriveSummaryStateFromSources(lr)
	}

	return view
}

func buildDocumentView(views []entryView, carelessHideOK bool) documentView {
	doc := documentView{
		total: len(views),
	}

	for _, view := range views {
		if shouldHideEntry(view, carelessHideOK, false) {
			doc.hiddenOK++
		} else {
			doc.shown++
		}

		switch view.summaryState {
		case summaryStateOK:
			doc.explicitOK++
		case summaryStateReview:
			doc.review++
		case summaryStateError:
			doc.errors++
		default:
			doc.unknown++
		}
	}

	return doc
}

func shouldHideEntry(view entryView, carelessHideOK bool, singleEntry bool) bool {
	if singleEntry || !carelessHideOK {
		return false
	}
	return view.summaryState == summaryStateOK
}

func renderDocument(doc documentView, views []entryView, carelessHideOK bool, singleEntry bool) string {
	var b strings.Builder

	if !singleEntry {
		fmt.Fprintf(&b, "Analyzed %d bibliographic entries\n", doc.total)
		fmt.Fprintf(&b, "Showing %d entries", doc.shown)
		if carelessHideOK {
			fmt.Fprintf(&b, " (%d hidden by --careless-hide-ok)", doc.hiddenOK)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "Summary states: review=%d error=%d ok=%d unknown=%d\n", doc.review, doc.errors, doc.explicitOK, doc.unknown)
		b.WriteString("\n")
	}

	rendered := 0
	for _, view := range views {
		if shouldHideEntry(view, carelessHideOK, singleEntry) {
			continue
		}
		if rendered > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderEntry(view))
		rendered++
	}

	if rendered == 0 {
		b.WriteString("No entries to display.\n")
	}

	return b.String()
}

func renderEntry(view entryView) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Entry %d [%s]\n", view.number, strings.ToUpper(string(view.summaryState)))
	b.WriteString(renderLabeledBlock("Original", view.originalText))
	if view.summaryComment != "" {
		b.WriteString(renderLabeledBlock("Summary", view.summaryComment))
	}
	b.WriteString(renderSourceBlock("Lookups", view.sources))

	return b.String()
}

func renderLabeledBlock(label, value string) string {
	if value == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	var b strings.Builder
	if len(lines) == 1 {
		fmt.Fprintf(&b, "%s: %s\n", label, lines[0])
		return b.String()
	}

	fmt.Fprintf(&b, "%s:\n", label)
	for _, line := range lines {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	return b.String()
}

func deriveSummaryStateFromSources(lr *lookup.Result) summaryState {
	if lr.OSTI.Error != nil || lr.Arxiv.Error != nil || lr.Elsevier.Error != nil || lr.Crossref.Error != nil || lr.Online.Error != nil || lr.DOIOrg.Error != nil {
		return summaryStateError
	}
	if lr.OSTI.Record != nil || lr.Arxiv.Entry != nil || lr.Elsevier.Result != nil || lr.Crossref.Work != nil || lr.Online.Metadata != nil || lr.DOIOrg.Found {
		return summaryStateUnknown
	}
	return summaryStateReview
}

func renderSourceBlock(label string, sources []sourceView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", label)
	for _, source := range sources {
		fmt.Fprintf(&b, "  %s: %s", source.name, colorizeSourceStatus(source.status))
		if source.detail != "" {
			fmt.Fprintf(&b, " (%s)", source.detail)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func colorizeSourceStatus(status string) string {
	switch status {
	case "error":
		return prettytext.FgRed.Sprint(status)
	case "skipped":
		return prettytext.FgYellow.Sprint(status)
	default:
		return status
	}
}

func buildDOISourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "DOI", status: "skipped"}
	switch {
	case lr.DOIOrg.Found:
		view.status = "found"
		view.detail = "exists"
	case lr.DOIOrg.Error != nil:
		view.status = "error"
		view.detail = lr.DOIOrg.Error.Error()
	case lr.DOIOrg.ID != "":
		view.status = "not-found"
		view.detail = lr.DOIOrg.ID
	}
	return view
}

func buildOSTISourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "OSTI", status: "skipped"}
	switch {
	case lr.OSTI.Record != nil:
		view.status = "found"
		view.detail = lr.OSTI.Record.ToString()
	case lr.OSTI.Error != nil:
		view.status = "error"
		view.detail = lr.OSTI.Error.Error()
	case lr.OSTI.ID != "":
		view.status = "not-found"
		view.detail = lr.OSTI.ID
	}
	return view
}

func buildArxivSourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "arXiv", status: "skipped"}
	switch {
	case lr.Arxiv.Entry != nil:
		view.status = "found"
		view.detail = lr.Arxiv.Entry.ToString()
	case lr.Arxiv.Error != nil:
		view.status = "error"
		view.detail = lr.Arxiv.Error.Error()
	case lr.Arxiv.ID != "":
		view.status = "not-found"
		view.detail = lr.Arxiv.ID
	}
	return view
}

func buildElsevierSourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "Elsevier", status: "skipped"}
	switch {
	case lr.Elsevier.Result != nil:
		view.status = "matched"
		view.detail = lr.Elsevier.Result.ToString()
	case lr.Elsevier.Error != nil:
		view.status = "error"
		view.detail = lr.Elsevier.Error.Error()
	case lr.Elsevier.Status == lookup.SearchStatusDone:
		view.status = "no-match"
	}
	return view
}

func buildCrossrefSourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "Crossref", status: "skipped"}
	switch {
	case lr.Crossref.Work != nil:
		view.status = "matched"
		view.detail = lr.Crossref.Work.ToString()
	case lr.Crossref.Error != nil:
		view.status = "error"
		view.detail = lr.Crossref.Error.Error()
	case lr.Crossref.Comment != "":
		view.status = "no-match"
		view.detail = lr.Crossref.Comment
	case lr.Crossref.Status == lookup.SearchStatusDone:
		view.status = "no-match"
	}
	return view
}

func buildOnlineSourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "Online", status: "skipped"}
	switch {
	case lr.Online.Metadata != nil:
		view.status = "found"
		view.detail = lr.Online.Metadata.ToString()
	case lr.Online.Error != nil:
		view.status = "error"
		view.detail = lr.Online.Error.Error()
	case lr.Online.Status == lookup.SearchStatusDone:
		view.status = "not-found"
	}
	return view
}
