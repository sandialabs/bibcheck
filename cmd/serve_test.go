// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFetchHandlerFetchesUpstreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("<title>ok</title>")); err != nil {
			t.Fatalf("write upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/fetch?url="+url.QueryEscape(upstream.URL), nil)
	resp := httptest.NewRecorder()

	fetchHandler(1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "text/html" {
		t.Fatalf("expected content type text/html, got %q", got)
	}
	if got := resp.Body.String(); got != "<title>ok</title>" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestFetchHandlerRejectsOversizedUpstreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("too large")); err != nil {
			t.Fatalf("write upstream response: %v", err)
		}
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/fetch?url="+url.QueryEscape(upstream.URL), nil)
	resp := httptest.NewRecorder()

	fetchHandler(3).ServeHTTP(resp, req)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "exceeds 3 bytes") {
		t.Fatalf("unexpected response body: %q", resp.Body.String())
	}
}

func TestFetchHandlerRejectsUnsupportedURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/fetch?url="+url.QueryEscape("file:///etc/passwd"), nil)
	resp := httptest.NewRecorder()

	fetchHandler(1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
}
