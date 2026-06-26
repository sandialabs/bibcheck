// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/web/workflow"
)

const shirtyKeyStorageKey = "bibcheck.shirty_api_key"

type app struct {
	vecty.Core

	shirtyKey     string
	shirtyBaseURL string
	openRouterKey string
	entry         string
	filename      string
	pdf           []byte
	dragging      bool
	running       bool
	errorMessage  string
	state         workflow.State
}

func main() {
	vecty.SetTitle("Bibcheck")
	vecty.RenderBody(newApp())
	select {}
}

func newApp() *app {
	a := &app{shirtyBaseURL: config.DefaultShirtyBaseURL}
	if showShirtyKey {
		a.shirtyKey = loadLocalStorage(shirtyKeyStorageKey)
	}
	return a
}

func (a *app) Render() vecty.ComponentOrHTML {
	if a.running || a.state.Phase != "" {
		return a.renderAnalysis()
	}
	return a.renderLanding()
}

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
							elem.Span(vecty.Text("Shirty base URL")),
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
		items = append(items, elem.Article(
			vecty.Markup(vecty.Class("entry-card")),
			elem.Header(
				elem.Heading2(vecty.Text("Entry "+entry.ID)),
				elem.Span(vecty.Markup(vecty.Class("status", entry.AnalysisStatus)), vecty.Text(statusText(entry))),
			),
			elem.Div(
				vecty.Markup(vecty.Class("entry-columns")),
				elem.Div(
					vecty.Markup(vecty.Class("entry-pane")),
					elem.Heading3(vecty.Text("Extracted text")),
					elem.Preformatted(vecty.Text(nonEmpty(entry.Text, statusCopy(entry.TextStatus)))),
				),
				elem.Div(
					vecty.Markup(vecty.Class("entry-pane")),
					elem.Heading3(vecty.Text("Analysis")),
					elem.Preformatted(vecty.Text(nonEmpty(entry.Analysis, nonEmpty(entry.Error, statusCopy(entry.AnalysisStatus))))),
				),
			),
		))
	}
	return items
}

func (a *app) ready() bool {
	return len(a.pdf) > 0 && (shirtyKey(a) != "" || openRouterKey(a) != "")
}

func (a *app) start() {
	entry, err := selectedEntry(a.entry)
	if err != nil {
		a.errorMessage = err.Error()
		vecty.Rerender(a)
		return
	}

	rt, err := workflow.NewRuntime(workflow.Keys{
		ShirtyAPIKey:     shirtyKey(a),
		ShirtyBaseURL:    shirtyBaseURL(a),
		OpenRouterAPIKey: openRouterKey(a),
	})
	if err != nil {
		a.errorMessage = err.Error()
		vecty.Rerender(a)
		return
	}

	a.running = true
	a.errorMessage = ""
	a.state = workflow.State{Provider: rt.Kind, Phase: "Starting"}
	vecty.Rerender(a)

	pdf := append([]byte(nil), a.pdf...)
	options := workflow.Options{Entry: entry}
	go workflow.AnalyzePDFWithOptions(context.Background(), rt, pdf, options, func(state workflow.State) {
		a.state = state
		vecty.Rerender(a)
	})
}

func shirtyKey(a *app) string {
	if !showShirtyKey {
		return ""
	}
	return strings.TrimSpace(a.shirtyKey)
}

func shirtyBaseURL(a *app) string {
	if !showShirtyKey {
		return ""
	}
	return strings.TrimSpace(a.shirtyBaseURL)
}

func openRouterKey(a *app) string {
	if !showOpenRouterKey {
		return ""
	}
	return strings.TrimSpace(a.openRouterKey)
}

func loadLocalStorage(key string) string {
	defer func() {
		_ = recover()
	}()
	storage := js.Global().Get("localStorage")
	if !storage.Truthy() {
		return ""
	}
	value := storage.Call("getItem", key)
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}

func saveLocalStorage(key, value string) {
	defer func() {
		_ = recover()
	}()
	storage := js.Global().Get("localStorage")
	if !storage.Truthy() {
		return
	}
	if value == "" {
		storage.Call("removeItem", key)
		return
	}
	storage.Call("setItem", key, value)
}

func (a *app) reset() {
	a.filename = ""
	a.pdf = nil
	a.running = false
	a.errorMessage = ""
	a.state = workflow.State{}
	vecty.Rerender(a)
}

func (a *app) loadFileList(files js.Value) {
	if !files.Truthy() || files.Get("length").Int() < 1 {
		return
	}
	file := files.Index(0)
	name := file.Get("name").String()
	if !strings.HasSuffix(strings.ToLower(name), ".pdf") && file.Get("type").String() != "application/pdf" {
		a.errorMessage = "Select a PDF file."
		vecty.Rerender(a)
		return
	}

	reader := js.Global().Get("FileReader").New()
	var onload js.Func
	var onerror js.Func
	onload = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer onload.Release()
		defer onerror.Release()

		array := js.Global().Get("Uint8Array").New(reader.Get("result"))
		data := make([]byte, array.Get("byteLength").Int())
		js.CopyBytesToGo(data, array)

		a.filename = name
		a.pdf = data
		a.errorMessage = ""
		vecty.Rerender(a)
		return nil
	})
	onerror = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer onload.Release()
		defer onerror.Release()
		a.errorMessage = "Could not read the selected PDF."
		vecty.Rerender(a)
		return nil
	})
	reader.Set("onload", onload)
	reader.Set("onerror", onerror)
	reader.Call("readAsArrayBuffer", file)
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

func selectedEntry(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	entry, err := strconv.Atoi(trimmed)
	if err != nil || entry < 1 {
		return 0, fmt.Errorf("Bibliography entry must be a positive whole number.")
	}
	return entry, nil
}

func providerText(provider workflow.ProviderKind) string {
	if provider == "" {
		return "selected provider"
	}
	return string(provider)
}

func progressText(state workflow.State) string {
	if state.Total == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d analyzed", state.Completed, state.Total)
}

func maxProgress(state workflow.State) int {
	if state.Total < 1 {
		return 1
	}
	return state.Total
}

func valueProgress(state workflow.State) int {
	if state.Completed < 0 {
		return 0
	}
	return state.Completed
}

func statusText(entry workflow.EntryState) string {
	if entry.AnalysisStatus != "pending" {
		return entry.AnalysisStatus
	}
	return entry.TextStatus
}

func statusCopy(status string) string {
	switch status {
	case "active":
		return "Working..."
	case "completed":
		return ""
	case "error":
		return "Error"
	default:
		return "Pending"
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
