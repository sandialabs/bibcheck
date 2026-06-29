// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"strings"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	"github.com/sandialabs/bibcheck/config"
)

func (a *app) renderLanding() vecty.ComponentOrHTML {
	dropClasses := []string{"drop-target"}
	if a.dragging {
		dropClasses = append(dropClasses, "dragging")
	}
	if a.filename != "" {
		dropClasses = append(dropClasses, "selected")
	}

	return elem.Body(
		elem.Main(
			vecty.Markup(vecty.Class("shell")),
			elem.Section(
				vecty.Markup(vecty.Class("landing")),
				elem.Heading1(vecty.Text("Bibcheck")),
				elem.Div(
					vecty.Markup(vecty.Class("form-grid")),
					vecty.If(showShirtyKey,
						elem.Label(
							vecty.Markup(vecty.Class("field")),
							elem.Span(vecty.Text("Shirty API key")),
							elem.Input(
								vecty.Markup(
									prop.Type(prop.TypePassword),
									prop.Placeholder("Paste Shirty API key"),
									prop.Value(a.shirtyKey),
									event.Input(func(e *vecty.Event) {
										a.shirtyKey = e.Target.Get("value").String()
										saveLocalStorage(shirtyKeyStorageKey, strings.TrimSpace(a.shirtyKey))
										a.errorMessage = ""
										vecty.Rerender(a)
									}),
								),
							),
						),
					),
					vecty.If(showOpenRouterKey,
						elem.Label(
							vecty.Markup(vecty.Class("field")),
							elem.Span(vecty.Text("OpenRouter API key")),
							elem.Input(
								vecty.Markup(
									prop.Type(prop.TypePassword),
									prop.Placeholder("Paste OpenRouter API key"),
									prop.Value(a.openRouterKey),
									event.Input(func(e *vecty.Event) {
										a.openRouterKey = e.Target.Get("value").String()
										a.errorMessage = ""
										vecty.Rerender(a)
									}),
								),
							),
						),
					),
				),
				elem.Details(
					vecty.Markup(vecty.Class("advanced-options")),
					elem.Summary(vecty.Text("Advanced options")),
					elem.Label(
						vecty.Markup(vecty.Class("field")),
						elem.Span(vecty.Text("Bibliography entry")),
						elem.Input(
							vecty.Markup(
								prop.Type(prop.TypeNumber),
								vecty.Attribute("min", "1"),
								vecty.Attribute("step", "1"),
								prop.Placeholder("All entries"),
								prop.Value(a.entry),
								event.Input(func(e *vecty.Event) {
									a.entry = e.Target.Get("value").String()
									a.errorMessage = ""
									vecty.Rerender(a)
								}),
							),
						),
					),
					vecty.If(showShirtyKey,
						elem.Label(
							vecty.Markup(vecty.Class("field")),
							elem.Span(vecty.Text("Shirty base URL (e.g. https://shirty.sandia.gov/api/v1)")),
							elem.Input(
								vecty.Markup(
									prop.Type(prop.TypeText),
									prop.Placeholder(config.DefaultShirtyBaseURL),
									prop.Value(a.shirtyBaseURL),
									event.Input(func(e *vecty.Event) {
										a.shirtyBaseURL = e.Target.Get("value").String()
										a.errorMessage = ""
										vecty.Rerender(a)
									}),
								),
							),
						),
					),
				),
				elem.Div(
					vecty.Markup(
						vecty.Class(dropClasses...),
						event.DragEnter(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.dragging = true
							vecty.Rerender(a)
						}),
						event.DragOver(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.dragging = true
							vecty.Rerender(a)
						}),
						event.DragLeave(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.dragging = false
							vecty.Rerender(a)
						}),
						event.Drop(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.dragging = false
							files := e.Value.Get("dataTransfer").Get("files")
							a.loadFileList(files)
						}),
					),
					elem.Input(
						vecty.Markup(
							prop.ID("pdf-file"),
							prop.Type(prop.TypeFile),
							vecty.Attribute("accept", "application/pdf,.pdf"),
							event.Change(func(e *vecty.Event) {
								a.loadFileList(e.Target.Get("files"))
							}),
						),
					),
					elem.Label(
						vecty.Markup(prop.For("pdf-file")),
						elem.Strong(vecty.Text(dropTitle(a.filename))),
						elem.Span(vecty.Text(dropSubtitle(a.filename))),
					),
				),
				vecty.If(a.errorMessage != "",
					elem.Div(vecty.Markup(vecty.Class("error")), vecty.Text(a.errorMessage)),
				),
				elem.Button(
					vecty.Markup(
						vecty.Class("primary-action"),
						prop.Type(prop.TypeButton),
						prop.Disabled(!a.ready()),
						event.Click(func(e *vecty.Event) {
							e.Value.Call("preventDefault")
							a.start()
						}),
					),
					vecty.Text("Analyze PDF"),
				),
			),
		),
		renderFooter(),
	)
}

func dropTitle(filename string) string {
	if filename != "" {
		return filename
	}
	return "Drop a PDF here"
}

func dropSubtitle(filename string) string {
	if filename != "" {
		return "PDF ready for analysis"
	}
	return "or choose a file"
}
