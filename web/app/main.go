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
