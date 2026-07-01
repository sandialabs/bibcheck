// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/prop"
	"github.com/sandialabs/bibcheck/version"
)

func renderFooter() vecty.ComponentOrHTML {
	return elem.Footer(
		vecty.Markup(vecty.Class("app-footer")),
		elem.Div(
			vecty.Markup(vecty.Class("footer-item")),
			elem.Anchor(
				vecty.Markup(
					prop.Href("https://github.com/sandialabs/bibcheck"),
					vecty.Attribute("target", "_blank"),
					vecty.Attribute("rel", "noopener noreferrer"),
				),
				vecty.Text("sandialabs/bibcheck"),
			),
			vecty.Text(" · "+version.String()),
		),
		elem.Div(
			vecty.Markup(vecty.Class("footer-item")),
			vecty.Text("(c) 2025 National Technology and Engineering Solutions of Sandia"),
		),
		elem.Div(
			vecty.Markup(vecty.Class("footer-item")),
			vecty.Text("Point of contact: Carl Pearson <cwpears@sandia.gov>"),
		),
	)
}
