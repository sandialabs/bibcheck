// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"fmt"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

func (a *app) renderAnalysis() vecty.ComponentOrHTML {
	return elem.Body(
		elem.Main(
			vecty.Markup(vecty.Class("analysis-shell")),
			elem.Header(
				vecty.Markup(vecty.Class("analysis-header")),
				elem.Div(
					elem.Heading1(vecty.Text("Bibliography Analysis")),
					elem.Paragraph(vecty.Text(fmt.Sprintf("%s using %s", a.filename, providerText(a.state.Provider)))),
				),
				elem.Button(
					vecty.Markup(
						prop.Type(prop.TypeButton),
						event.Click(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.reset()
						}),
					),
					vecty.Text("New PDF"),
				),
			),
			elem.Section(
				vecty.Markup(vecty.Class("status-band")),
				elem.Div(
					elem.Span(vecty.Markup(vecty.Class("phase")), vecty.Text(nonEmpty(a.state.Phase, "Starting"))),
					elem.Span(vecty.Text(progressText(a.state))),
				),
				elem.Progress(vecty.Markup(vecty.Attribute("max", maxProgress(a.state)), vecty.Attribute("value", valueProgress(a.state)))),
			),
			vecty.If(a.state.Error != "",
				elem.Div(vecty.Markup(vecty.Class("error")), vecty.Text(a.state.Error)),
			),
			elem.Section(
				vecty.Markup(vecty.Class("entries")),
				a.renderEntries(),
			),
		),
		renderFooter(),
	)
}

func (a *app) renderEntries() vecty.MarkupOrChild {
	if len(a.state.Entries) == 0 {
		return elem.Div(vecty.Markup(vecty.Class("empty-state")), vecty.Text("Preparing bibliography."))
	}

	items := make(vecty.List, 0, len(a.state.Entries))
	for _, entry := range a.state.Entries {
		items = append(items, renderEntry(entry))
	}
	return items
}
