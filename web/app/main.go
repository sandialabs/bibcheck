// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/maxence-charriere/go-app/v11/pkg/app"
	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/web/workflow"
)

const shirtyKeyStorageKey = "bibcheck.shirty_api_key"

type bibcheckApp struct {
	app.Compo

	shirtyKey     string
	shirtyBaseURL string
	openRouterKey string
	entry         string
	filename      string
	pdf           []byte
	dragging      bool
	running       bool
	cancelRun     context.CancelFunc
	runID         uint64
	errorMessage  string
	state         workflow.State
	warningRead   bool
	uiContext     app.Context
}

func main() {
	app.Route("/", func() app.Composer { return newApp() })
	app.RunWhenOnBrowser()
}

func newApp() *bibcheckApp {
	return &bibcheckApp{
		shirtyBaseURL: config.DefaultShirtyBaseURL,
		warningRead:   !showWarningPage,
	}
}

func (a *bibcheckApp) OnMount(ctx app.Context) {
	a.uiContext = ctx
	if showShirtyKey {
		_ = ctx.LocalStorage().Get(shirtyKeyStorageKey, &a.shirtyKey)
		ctx.Update()
	}
}

func (a *bibcheckApp) Render() app.UI {
	if !a.warningRead {
		return a.renderWarning()
	}
	if a.running || a.state.Phase != "" {
		return a.renderAnalysis()
	}
	return a.renderLanding()
}

func (a *bibcheckApp) ready() bool {
	return len(a.pdf) > 0 && (shirtyAPIKey(a) != "" || openRouterAPIKey(a) != "")
}

func (a *bibcheckApp) start() {
	entry, err := selectedEntry(a.entry)
	if err != nil {
		a.errorMessage = err.Error()
		return
	}

	rt, err := workflow.NewRuntime(workflow.Keys{
		ShirtyAPIKey:     shirtyAPIKey(a),
		ShirtyBaseURL:    shirtyBaseURL(a),
		OpenRouterAPIKey: openRouterAPIKey(a),
	})
	if err != nil {
		a.errorMessage = err.Error()
		return
	}

	if a.cancelRun != nil {
		a.cancelRun()
	}
	a.running = true
	a.cancelRun = nil
	a.runID++
	runID := a.runID
	a.errorMessage = ""
	a.state = workflow.State{Provider: rt.Kind, Phase: "Starting"}
	pdf := append([]byte(nil), a.pdf...)
	runCtx, cancel := context.WithCancel(context.Background())
	a.cancelRun = cancel

	a.uiContext.Async(func() {
		workflow.AnalyzePDFWithOptions(runCtx, rt, pdf, workflow.Options{Entry: entry}, func(state workflow.State) {
			a.uiContext.Dispatch(func(app.Context) {
				if runID == a.runID {
					a.state = state
				}
			})
		})
	})
}

func shirtyAPIKey(a *bibcheckApp) string {
	if !showShirtyKey {
		return ""
	}
	return strings.TrimSpace(a.shirtyKey)
}

func shirtyBaseURL(a *bibcheckApp) string {
	if !showShirtyKey {
		return ""
	}
	return strings.TrimSpace(a.shirtyBaseURL)
}

func openRouterAPIKey(a *bibcheckApp) string {
	if !showOpenRouterKey {
		return ""
	}
	return strings.TrimSpace(a.openRouterKey)
}

func (a *bibcheckApp) reset() {
	if a.cancelRun != nil {
		a.cancelRun()
		a.cancelRun = nil
	}
	a.runID++
	a.filename = ""
	a.pdf = nil
	a.running = false
	a.errorMessage = ""
	a.state = workflow.State{}
}

func (a *bibcheckApp) loadFileList(ctx app.Context, files app.Value) {
	if !files.Truthy() || files.Length() < 1 {
		return
	}
	file := files.Index(0)
	name := file.Get("name").String()
	if !strings.HasSuffix(strings.ToLower(name), ".pdf") && file.Get("type").String() != "application/pdf" {
		a.errorMessage = "Select a PDF file."
		return
	}

	reader := app.Window().Get("FileReader").New()
	var onload, onerror app.Func
	onload = app.FuncOf(func(_ app.Value, _ []app.Value) any {
		defer onload.Release()
		defer onerror.Release()
		array := app.Window().Get("Uint8Array").New(reader.Get("result"))
		data := make([]byte, array.Get("byteLength").Int())
		app.CopyBytesToGo(data, array)
		ctx.Dispatch(func(app.Context) {
			a.filename = name
			a.pdf = data
			a.errorMessage = ""
		})
		return nil
	})
	onerror = app.FuncOf(func(_ app.Value, _ []app.Value) any {
		defer onload.Release()
		defer onerror.Release()
		ctx.Dispatch(func(app.Context) { a.errorMessage = "Could not read the selected PDF." })
		return nil
	})
	reader.Set("onload", onload)
	reader.Set("onerror", onerror)
	reader.Call("readAsArrayBuffer", file)
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
