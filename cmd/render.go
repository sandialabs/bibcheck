// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"strings"

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
	primaryMessage string
	primarySource  string
	summaryState   summaryState
	summaryComment string
	sources        []sourceView
}

type documentView struct {
	total        int
	shown        int
	hiddenOK     int
	review       int
	errors       int
	explicitOK   int
	unknown      int
	sourceCounts map[string]int
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
		view.primaryMessage = view.summaryComment
	case outcome.comment != "":
		view.summaryComment = outcome.comment
		if outcome.mismatch {
			view.summaryState = summaryStateReview
			view.primaryMessage = outcome.comment
		} else {
			view.summaryState = summaryStateOK
			view.primaryMessage = outcome.comment
		}
	default:
		view.summaryState = deriveSummaryStateFromSources(lr)
		view.primaryMessage = defaultPrimaryMessage(view.summaryState, lr)
	}

	view.primarySource = derivePrimarySource(lr)
	return view
}

func buildDocumentView(views []entryView, carelessHideOK bool) documentView {
	doc := documentView{
		total:        len(views),
		sourceCounts: map[string]int{},
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

		if view.primarySource != "" {
			doc.sourceCounts[view.primarySource]++
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
		if len(doc.sourceCounts) > 0 {
			b.WriteString("Primary matches: ")
			b.WriteString(renderSourceCounts(doc.sourceCounts))
			b.WriteString("\n")
		}
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
	if view.primaryMessage != "" {
		b.WriteString(renderLabeledBlock("Result", view.primaryMessage))
	}
	if view.primarySource != "" {
		fmt.Fprintf(&b, "Primary match: %s\n", view.primarySource)
	}
	fmt.Fprintf(&b, "Checks: %s\n", renderSourceLine(view.sources))
	if view.summaryComment != "" {
		b.WriteString(renderLabeledBlock("Summary", view.summaryComment))
	}

	return b.String()
}

func renderSourceLine(sources []sourceView) string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		part := source.name + " " + source.status
		if source.detail != "" {
			part += " (" + source.detail + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

func renderSourceCounts(counts map[string]int) string {
	order := []string{"OSTI", "arXiv", "Elsevier", "Crossref", "Online", "DOI"}
	parts := []string{}
	for _, name := range order {
		if counts[name] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", name, counts[name]))
		}
	}
	return strings.Join(parts, ", ")
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

func derivePrimarySource(lr *lookup.Result) string {
	switch {
	case lr.OSTI.Record != nil:
		return "OSTI"
	case lr.Arxiv.Entry != nil:
		return "arXiv"
	case lr.Elsevier.Result != nil:
		return "Elsevier"
	case lr.Crossref.Work != nil:
		return "Crossref"
	case lr.Online.Metadata != nil:
		return "Online"
	case lr.DOIOrg.Found:
		return "DOI"
	default:
		return ""
	}
}

func deriveSummaryStateFromSources(lr *lookup.Result) summaryState {
	if lr.OSTI.Error != nil || lr.Arxiv.Error != nil || lr.Elsevier.Error != nil || lr.Crossref.Error != nil || lr.Online.Error != nil || lr.DOIOrg.Error != nil {
		return summaryStateError
	}
	if derivePrimarySource(lr) != "" {
		return summaryStateUnknown
	}
	return summaryStateReview
}

func defaultPrimaryMessage(state summaryState, lr *lookup.Result) string {
	switch state {
	case summaryStateError:
		return "one or more lookups returned an error"
	case summaryStateReview:
		return "no conclusive match found"
	case summaryStateUnknown:
		if source := derivePrimarySource(lr); source != "" {
			return source + " returned a plausible match"
		}
	}
	return ""
}

func buildDOISourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "DOI", status: "skipped"}
	switch {
	case lr.DOIOrg.Found:
		view.status = "matched"
		view.detail = "exists"
	case lr.DOIOrg.Error != nil:
		view.status = "error"
		view.detail = lr.DOIOrg.Error.Error()
	case lr.DOIOrg.ID != "":
		view.status = "no-match"
		view.detail = lr.DOIOrg.ID
	}
	return view
}

func buildOSTISourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "OSTI", status: "skipped"}
	switch {
	case lr.OSTI.Record != nil:
		view.status = "matched"
		view.detail = lr.OSTI.Record.ToString()
	case lr.OSTI.Error != nil:
		view.status = "error"
		view.detail = lr.OSTI.Error.Error()
	case lr.OSTI.ID != "":
		view.status = "no-match"
		view.detail = lr.OSTI.ID
	}
	return view
}

func buildArxivSourceView(lr *lookup.Result) sourceView {
	view := sourceView{name: "arXiv", status: "skipped"}
	switch {
	case lr.Arxiv.Entry != nil:
		view.status = "matched"
		view.detail = lr.Arxiv.Entry.ToString()
	case lr.Arxiv.Error != nil:
		view.status = "error"
		view.detail = lr.Arxiv.Error.Error()
	case lr.Arxiv.ID != "":
		view.status = "no-match"
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
		view.status = "matched"
		view.detail = lr.Online.Metadata.ToString()
	case lr.Online.Error != nil:
		view.status = "error"
		view.detail = lr.Online.Error.Error()
	case lr.Online.Status == lookup.SearchStatusDone:
		view.status = "no-match"
	}
	return view
}
