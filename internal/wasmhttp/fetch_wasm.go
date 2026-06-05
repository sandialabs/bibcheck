// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package wasmhttp

import "net/http"

func ConfigureRequest(req *http.Request) {
	// Browsers may reject attempts to set User-Agent in fetch/CORS requests.
	req.Header.Del("User-Agent")
	// Go's wasm transport maps js.fetch:* pseudo-headers to browser fetch options.
	req.Header.Add("js.fetch:mode", "cors")
}
