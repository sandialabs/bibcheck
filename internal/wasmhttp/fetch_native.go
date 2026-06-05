// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build !js || !wasm

package wasmhttp

import "net/http"

func ConfigureRequest(_ *http.Request) {}
