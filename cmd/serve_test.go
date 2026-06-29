// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sandialabs/bibcheck/internal/wasmhttp"
)

func TestServeMuxLiveness(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, livenessPath, nil)
	resp := httptest.NewRecorder()

	serveMux(t.TempDir(), 1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); got != "ok\n" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := resp.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
}

func TestServeMuxLivenessSupportsHead(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, livenessPath, nil)
	resp := httptest.NewRecorder()

	serveMux(t.TempDir(), 1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if resp.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", resp.Body.String())
	}
}

func TestServeMuxLivenessRejectsOtherMethods(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, livenessPath, nil)
	resp := httptest.NewRecorder()

	serveMux(t.TempDir(), 1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d: %s", http.StatusMethodNotAllowed, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("expected Allow GET, HEAD, got %q", got)
	}
}

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
	if got := resp.Header().Get(wasmhttp.FetchResultHeader); got != wasmhttp.FetchResultUpstream {
		t.Fatalf("expected fetch result %q, got %q", wasmhttp.FetchResultUpstream, got)
	}
}

func TestFetchHandlerPreservesUpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream says no"))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/fetch?url="+url.QueryEscape(upstream.URL), nil)
	resp := httptest.NewRecorder()

	fetchHandler(1024).ServeHTTP(resp, req)

	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.Code)
	}
	if got := resp.Body.String(); got != "upstream says no" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := resp.Header().Get(wasmhttp.FetchResultHeader); got != wasmhttp.FetchResultUpstream {
		t.Fatalf("expected fetch result %q, got %q", wasmhttp.FetchResultUpstream, got)
	}
}

func TestFetchHandlerMarksUpstreamTimeoutAsProxyError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/fetch?url="+url.QueryEscape(upstream.URL), nil)
	resp := httptest.NewRecorder()

	fetchHandlerWithTimeout(1024, 10*time.Millisecond).ServeHTTP(resp, req)

	if resp.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d: %s", http.StatusGatewayTimeout, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get(wasmhttp.FetchResultHeader); got != wasmhttp.FetchResultProxyError {
		t.Fatalf("expected fetch result %q, got %q", wasmhttp.FetchResultProxyError, got)
	}
	if !strings.Contains(resp.Body.String(), "upstream request timed out after 10ms") {
		t.Fatalf("unexpected response body: %q", resp.Body.String())
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
	if got := resp.Header().Get(wasmhttp.FetchResultHeader); got != wasmhttp.FetchResultProxyError {
		t.Fatalf("expected fetch result %q, got %q", wasmhttp.FetchResultProxyError, got)
	}
}

func TestCompressedFileServerServesBrotliWhenAvailable(t *testing.T) {
	dir := staticTestDir(t)
	writeStaticFile(t, dir, "app.wasm", "plain wasm")
	writeStaticFile(t, dir, "app.wasm.br", "brotli wasm")

	req := httptest.NewRequest(http.MethodGet, "/app.wasm", nil)
	req.Header.Set("Accept-Encoding", "br")
	resp := httptest.NewRecorder()

	compressedFileServer(http.Dir(dir)).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("expected br content encoding, got %q", got)
	}
	if got := resp.Body.String(); got != "brotli wasm" {
		t.Fatalf("unexpected body: %q", got)
	}
	assertVaryAcceptEncoding(t, resp.Header())
}

func TestCompressedFileServerServesGzipWhenAvailable(t *testing.T) {
	dir := staticTestDir(t)
	writeStaticFile(t, dir, "style.css", "plain css")
	writeStaticFile(t, dir, "style.css.gz", "gzip css")

	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp := httptest.NewRecorder()

	compressedFileServer(http.Dir(dir)).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}
	if got := resp.Body.String(); got != "gzip css" {
		t.Fatalf("unexpected body: %q", got)
	}
	assertVaryAcceptEncoding(t, resp.Header())
}

func TestCompressedFileServerPrefersBrotliOverGzip(t *testing.T) {
	dir := staticTestDir(t)
	writeStaticFile(t, dir, "wasm_exec.js", "plain js")
	writeStaticFile(t, dir, "wasm_exec.js.br", "brotli js")
	writeStaticFile(t, dir, "wasm_exec.js.gz", "gzip js")

	req := httptest.NewRequest(http.MethodGet, "/wasm_exec.js", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	resp := httptest.NewRecorder()

	compressedFileServer(http.Dir(dir)).ServeHTTP(resp, req)

	if got := resp.Header().Get("Content-Encoding"); got != "br" {
		t.Fatalf("expected br content encoding, got %q", got)
	}
	if got := resp.Body.String(); got != "brotli js" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestCompressedFileServerFallsBackToPlainFile(t *testing.T) {
	dir := staticTestDir(t)
	writeStaticFile(t, dir, "index.html", "<h1>plain</h1>")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	resp := httptest.NewRecorder()

	compressedFileServer(http.Dir(dir)).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("expected no content encoding, got %q", got)
	}
	if got := strings.TrimSpace(resp.Body.String()); got != "<h1>plain</h1>" {
		t.Fatalf("unexpected body: %q", got)
	}
	assertVaryAcceptEncoding(t, resp.Header())
}

func TestCompressedFileServerDoesNotServeOrphanedCompressedFile(t *testing.T) {
	dir := staticTestDir(t)
	writeStaticFile(t, dir, "missing.css.gz", "gzip css")

	req := httptest.NewRequest(http.MethodGet, "/missing.css", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp := httptest.NewRecorder()

	compressedFileServer(http.Dir(dir)).ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
	if got := resp.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("expected no content encoding, got %q", got)
	}
}

func staticTestDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func writeStaticFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func assertVaryAcceptEncoding(t *testing.T, header http.Header) {
	t.Helper()
	for _, vary := range header.Values("Vary") {
		for _, part := range strings.Split(vary, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "Accept-Encoding") {
				return
			}
		}
	}
	t.Fatalf("expected Vary header to include Accept-Encoding, got %q", header.Values("Vary"))
}
