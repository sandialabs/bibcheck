// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"strings"

	"github.com/maxence-charriere/go-app/v11/pkg/app"
	"github.com/sandialabs/bibcheck/config"
)

func (a *bibcheckApp) renderLanding() app.UI {
	dropClasses := []string{"drop-target"}
	if a.dragging {
		dropClasses = append(dropClasses, "dragging")
	}
	if a.filename != "" {
		dropClasses = append(dropClasses, "selected")
	}

	fields := []app.UI{}
	if showShirtyKey {
		fields = append(fields,
			app.Label().Class("field").Body(
				app.Span().Body(
					app.Text("Shirty API key"),
				),
				app.Input().
					Type("password").
					Placeholder("Paste Shirty API key").
					Value(a.shirtyKey).
					OnInput(func(ctx app.Context, e app.Event) {
						a.shirtyKey = e.Get("target").Get("value").String()
						_ = ctx.LocalStorage().Set(shirtyKeyStorageKey, strings.TrimSpace(a.shirtyKey))
						a.errorMessage = ""
					}),
			),
		)
	}
	if showOpenRouterKey {
		fields = append(fields,
			app.Label().Class("field").Body(
				app.Span().Body(
					app.Text("OpenRouter API key"),
				),
				app.Input().
					Type("password").
					Placeholder("Paste OpenRouter API key").
					Value(a.openRouterKey).
					OnInput(func(_ app.Context, e app.Event) {
						a.openRouterKey = e.Get("target").Get("value").String()
						a.errorMessage = ""
					}),
			),
		)
	}

	advanced := []app.UI{
		app.Summary().Body(
			app.Text("Advanced options"),
		),
		app.Label().Class("field").Body(
			app.Span().Body(
				app.Text("Bibliography entry"),
			),
			app.Input().
				Type("number").
				Min(1).
				Step(1).
				Placeholder("All entries").
				Value(a.entry).
				OnInput(func(_ app.Context, e app.Event) {
					a.entry = e.Get("target").Get("value").String()
					a.errorMessage = ""
				}),
		),
	}
	if showShirtyKey {
		advanced = append(advanced,
			app.Label().Class("field").Body(
				app.Span().Body(
					app.Text("Shirty base URL (e.g. https://shirty.sandia.gov/api/v1)"),
				),
				app.Input().
					Type("text").
					Placeholder(config.DefaultShirtyBaseURL).
					Value(a.shirtyBaseURL).
					OnInput(func(_ app.Context, e app.Event) {
						a.shirtyBaseURL = e.Get("target").Get("value").String()
						a.errorMessage = ""
					}),
			),
		)
	}

	contents := []app.UI{
		app.H1().Body(
			app.Text("Bibcheck"),
		),
		app.Div().Class("form-grid").Body(fields...),
		app.Details().Class("advanced-options").Body(advanced...),
		a.fileDropTarget(dropClasses),
	}
	if a.errorMessage != "" {
		contents = append(contents,
			app.Div().Class("error").Body(
				app.Text(a.errorMessage),
			),
		)
	}
	contents = append(contents,
		app.Button().
			Class("primary-action").
			Type("button").
			Disabled(!a.ready()).
			OnClick(func(_ app.Context, e app.Event) {
				e.PreventDefault()
				a.start()
			}).
			Body(
				app.Text("Analyze PDF"),
			),
	)

	return app.Div().Class("app-page").Body(
		app.Main().Class("shell").Body(
			app.Section().Class("landing").Body(contents...),
		),
		renderFooter(),
	)
}

func (a *bibcheckApp) fileDropTarget(classes []string) app.UI {
	return app.Div().
		Class(classes...).
		OnDragEnter(func(_ app.Context, e app.Event) {
			e.PreventDefault()
			a.dragging = true
		}).
		OnDragOver(func(_ app.Context, e app.Event) {
			e.PreventDefault()
			a.dragging = true
		}).
		OnDragLeave(func(_ app.Context, e app.Event) {
			e.PreventDefault()
			a.dragging = false
		}).
		OnDrop(func(ctx app.Context, e app.Event) {
			e.PreventDefault()
			a.dragging = false
			a.loadFileList(ctx, e.Get("dataTransfer").Get("files"))
		}).
		Body(
			app.Input().
				ID("pdf-file").
				Type("file").
				Accept("application/pdf,.pdf").
				OnChange(func(ctx app.Context, e app.Event) {
					a.loadFileList(ctx, e.Get("target").Get("files"))
				}),
			app.Label().For("pdf-file").Body(
				app.Strong().Body(
					app.Text(dropTitle(a.filename)),
				),
				app.Span().Body(
					app.Text(dropSubtitle(a.filename)),
				),
			),
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
