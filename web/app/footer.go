// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"github.com/maxence-charriere/go-app/v11/pkg/app"
	"github.com/sandialabs/bibcheck/version"
)

func renderFooter() app.UI {
	items := []app.UI{
		app.Div().Class("footer-item").Body(
			app.A().
				Href("https://github.com/sandialabs/bibcheck").
				Attr("target", "_blank").
				Attr("rel", "noopener noreferrer").
				Body(
					app.Text("sandialabs/bibcheck"),
				),
			app.Text(" · "+version.String()),
		),
	}
	if showHowItWorksLink {
		items = append(items,
			app.Div().Class("footer-item").Body(
				app.A().
					Href("https://github.com/sandialabs/bibcheck/blob/main/docs/snl-how-it-works.md").
					Attr("target", "_blank").
					Attr("rel", "noopener noreferrer").
					Body(
						app.Text("How it Works"),
					),
			),
		)
	}
	items = append(items,
		app.Div().Class("footer-item").Body(
			app.Text("(c) 2025 National Technology and Engineering Solutions of Sandia"),
		),
		app.Div().Class("footer-item").Body(
			app.Text("Point of contact: Carl Pearson <cwpears@sandia.gov>"),
		),
	)

	return app.Footer().Class("app-footer").Body(items...)
}
