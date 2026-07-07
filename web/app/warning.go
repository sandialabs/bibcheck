// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

func (a *app) renderWarning() vecty.ComponentOrHTML {
	dismiss := func(e *vecty.Event) {
		a.warningRead = true
		vecty.Rerender(a)
	}

	return elem.Body(
		vecty.Markup(
			event.Click(dismiss),
			event.KeyDown(dismiss),
		),
		elem.Main(
			vecty.Markup(vecty.Class("warning-shell")),
			elem.Section(
				vecty.Markup(vecty.Class("warning-page")),
				elem.Heading1(vecty.Text("!! UUI/UUR Only !!")),
				elem.Paragraph(vecty.Text("This application communicates with external resources. Any uploaded document must be UUI or UUR.")),
				elem.Paragraph(
					vecty.Markup(vecty.Class("warning-prompt")),
					vecty.Text("Click anywhere or press any key to continue"),
				),
			),
		),
	)
}
