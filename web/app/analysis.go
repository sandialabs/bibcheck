// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v11/pkg/app"
)

func (a *bibcheckApp) renderAnalysis() app.UI {
	contents := []app.UI{
		app.Header().Class("analysis-header").Body(
			app.Div().Body(
				app.H1().Body(
					app.Text("Bibliography Analysis"),
				),
				app.P().Body(
					app.Text(fmt.Sprintf("%s using %s", a.filename, providerText(a.state.Provider))),
				),
			),
			app.Button().
				Type("button").
				OnClick(func(_ app.Context, e app.Event) {
					e.PreventDefault()
					a.reset()
				}).
				Body(
					app.Text("New PDF"),
				),
		),
		app.Section().Class("status-band").Body(
			app.Div().Body(
				app.Span().Class("phase").Body(
					app.Text(nonEmpty(a.state.Phase, "Starting")),
				),
				app.Span().Body(
					app.Text(progressText(a.state)),
				),
			),
			app.Progress().
				Attr("max", maxProgress(a.state)).
				Attr("value", valueProgress(a.state)),
		),
	}
	if a.state.Error != "" {
		contents = append(contents,
			app.Div().Class("error").Body(
				app.Text(a.state.Error),
			),
		)
	}
	contents = append(contents,
		app.Section().Class("entries").Body(a.renderEntries()...),
	)

	return app.Div().Class("app-page").Body(
		app.Main().Class("analysis-shell").Body(contents...),
		renderFooter(),
	)
}

func (a *bibcheckApp) renderEntries() []app.UI {
	if len(a.state.Entries) == 0 {
		return nil
	}
	items := make([]app.UI, 0, len(a.state.Entries))
	for _, entry := range a.state.Entries {
		items = append(items, renderEntry(entry))
	}
	return items
}
