// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sandialabs/bibcheck/internal/wasmhttp"
)

func TestFetchResponseErrorReportsProxyErrorBody(t *testing.T) {
	resp := &http.Response{
		Status:     "504 Gateway Timeout",
		StatusCode: http.StatusGatewayTimeout,
		Header: http.Header{
			wasmhttp.FetchResultHeader: []string{wasmhttp.FetchResultProxyError},
		},
		Body: io.NopCloser(strings.NewReader("upstream request timed out after 15s\n")),
	}

	err := fetchResponseError(resp, true)
	if err == nil || err.Error() != "/api/fetch error: upstream request timed out after 15s" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchResponseErrorPreservesUpstreamErrorProvenance(t *testing.T) {
	resp := &http.Response{
		Status:     "502 Bad Gateway",
		StatusCode: http.StatusBadGateway,
		Header: http.Header{
			wasmhttp.FetchResultHeader: []string{wasmhttp.FetchResultUpstream},
		},
		Body: io.NopCloser(strings.NewReader("upstream body")),
	}

	err := fetchResponseError(resp, true)
	if err == nil || err.Error() != "upstream HTTP error: 502 Bad Gateway" {
		t.Fatalf("unexpected error: %v", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read body: %v", readErr)
	}
	if got := string(body); got != "upstream body" {
		t.Fatalf("expected untouched upstream body, got %q", got)
	}
}

func TestFetchResponseErrorRejectsMissingResultMarker(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("not the requested resource")),
	}

	err := fetchResponseError(resp, true)
	if err == nil || !strings.Contains(err.Error(), "endpoint may be unavailable or misconfigured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
