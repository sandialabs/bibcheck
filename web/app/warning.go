// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import "github.com/maxence-charriere/go-app/v11/pkg/app"

func (a *bibcheckApp) renderWarning() app.UI {
	dismiss := func(_ app.Context, _ app.Event) {
		a.warningRead = true
	}

	return app.Main().
		Class("warning-shell").
		OnClick(dismiss).
		OnKeyDown(dismiss).
		Body(
			app.Section().Class("warning-page").Body(
				app.H1().Body(
					app.Text("!! UUI/UUR Only !!"),
				),
				app.P().Body(
					app.Text("This application communicates with external resources. Any uploaded document must be UUI or UUR."),
				),
				app.P().Class("warning-prompt").Body(
					app.Text("Click anywhere or press any key to continue"),
				),
			),
		)
}
