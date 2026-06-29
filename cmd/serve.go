// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/internal/wasmhttp"
	"github.com/spf13/cobra"
)

const fetchUpstreamTimeout = 15 * time.Second

const livenessPath = "/livez"

var (
	serveAddr     string
	serveDir      string
	fetchMaxBytes int64
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static web UI",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if fetchMaxBytes < 1 {
			return fmt.Errorf("fetch-max-bytes must be positive")
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "serving %s at http://%s\n", serveDir, serveAddr)
		return http.ListenAndServe(serveAddr, serveMux(serveDir, fetchMaxBytes))
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "localhost:8080", "Address for the static web UI")
	serveCmd.Flags().StringVar(&serveDir, "web-dir", "web/static", "Directory containing the static web UI")
	serveCmd.Flags().Int64Var(&fetchMaxBytes, "fetch-max-bytes", 25*1024*1024, "Maximum bytes to read from /api/fetch upstream responses")
}

func serveMux(staticDir string, maxBytes int64) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(livenessPath, livenessHandler())
	mux.Handle("/api/fetch", fetchHandler(maxBytes))
	mux.Handle("/", wasmBundleLogHandler(compressedFileServer(http.Dir(staticDir))))
	return mux
}

func livenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", "3")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.WriteString(w, "ok\n")
	})
}

type responseLogRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *responseLogRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseLogRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(body)
	r.bytes += int64(n)
	return n, err
}

func (r *responseLogRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *responseLogRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func wasmBundleLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseLogRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		name, ok := staticFileName(r.URL.Path)
		if !ok || name != "app.wasm" || recorder.Status() < 200 || recorder.Status() >= 300 {
			return
		}
		log.Printf("wasm bundle served method=%s path=%q status=%d bytes=%d %s",
			r.Method,
			r.URL.Path,
			recorder.Status(),
			recorder.bytes,
			clientAddressLogFields(r),
		)
	})
}

type compressedStaticHandler struct {
	root     http.Dir
	fallback http.Handler
}

func compressedFileServer(root http.Dir) http.Handler {
	return compressedStaticHandler{
		root:     root,
		fallback: http.FileServer(root),
	}
}

func (h compressedStaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addVary(w.Header(), "Accept-Encoding")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.fallback.ServeHTTP(w, r)
		return
	}

	name, ok := staticFileName(r.URL.Path)
	if !ok {
		h.fallback.ServeHTTP(w, r)
		return
	}
	original, err := h.root.Open(name)
	if err != nil {
		h.fallback.ServeHTTP(w, r)
		return
	}
	originalInfo, err := original.Stat()
	original.Close()
	if err != nil || originalInfo.IsDir() {
		h.fallback.ServeHTTP(w, r)
		return
	}

	encoding, suffix, ok := preferredEncoding(r.Header.Get("Accept-Encoding"))
	if !ok {
		h.fallback.ServeHTTP(w, r)
		return
	}
	compressed, err := h.root.Open(name + suffix)
	if err != nil {
		h.fallback.ServeHTTP(w, r)
		return
	}
	defer compressed.Close()
	compressedInfo, err := compressed.Stat()
	if err != nil || compressedInfo.IsDir() {
		h.fallback.ServeHTTP(w, r)
		return
	}

	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Encoding", encoding)
	http.ServeContent(w, r, path.Base(name), originalInfo.ModTime(), compressed)
}

func staticFileName(urlPath string) (string, bool) {
	if strings.Contains(urlPath, "\x00") {
		return "", false
	}
	clean := path.Clean("/" + urlPath)
	name := strings.TrimPrefix(clean, "/")
	if strings.HasSuffix(urlPath, "/") {
		name = path.Join(name, "index.html")
	}
	if name == "." || name == "" {
		name = "index.html"
	}
	return name, true
}

func preferredEncoding(acceptEncoding string) (string, string, bool) {
	if acceptsEncoding(acceptEncoding, "br") {
		return "br", ".br", true
	}
	if acceptsEncoding(acceptEncoding, "gzip") {
		return "gzip", ".gz", true
	}
	return "", "", false
}

func acceptsEncoding(acceptEncoding, encoding string) bool {
	for _, part := range strings.Split(acceptEncoding, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		if len(fields) == 0 || !strings.EqualFold(strings.TrimSpace(fields[0]), encoding) {
			continue
		}
		accepted := true
		for _, field := range fields[1:] {
			if strings.EqualFold(strings.TrimSpace(field), "q=0") || strings.EqualFold(strings.TrimSpace(field), "q=0.0") {
				accepted = false
				break
			}
		}
		if accepted {
			return true
		}
	}
	return false
}

func addVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func fetchHandler(maxBytes int64) http.Handler {
	return fetchHandlerWithTimeout(maxBytes, fetchUpstreamTimeout)
}

func fetchHandlerWithTimeout(maxBytes int64, timeout time.Duration) http.Handler {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := validateFetchURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(wasmhttp.FetchResultHeader, wasmhttp.FetchResultProxyError)

		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "missing url parameter", http.StatusBadRequest)
			return
		}

		targetURL, err := url.Parse(rawURL)
		if err != nil {
			http.Error(w, "invalid url parameter", http.StatusBadRequest)
			return
		}
		if err := validateFetchURL(targetURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("proxy GET url=%q %s", targetURL.String(), clientAddressLogFields(r))
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL.String(), nil)
		if err != nil {
			http.Error(w, "create upstream request failed", http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", config.UserAgent())

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("proxy GET failed url=%q: %v", targetURL.String(), err)
			if isTimeoutError(err) {
				http.Error(w, fmt.Sprintf("upstream request timed out after %s", timeout), http.StatusGatewayTimeout)
				return
			}
			http.Error(w, "upstream request failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		body, tooLarge, err := readLimited(resp.Body, maxBytes)
		if err != nil {
			log.Printf("proxy GET read failed url=%q: %v", targetURL.String(), err)
			if isTimeoutError(err) {
				http.Error(w, fmt.Sprintf("upstream response timed out after %s", timeout), http.StatusGatewayTimeout)
				return
			}
			http.Error(w, "read upstream response failed", http.StatusBadGateway)
			return
		}
		if tooLarge {
			http.Error(w, fmt.Sprintf("upstream response exceeds %d bytes", maxBytes), http.StatusRequestEntityTooLarge)
			return
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		} else if len(body) > 0 {
			w.Header().Set("Content-Type", http.DetectContentType(body))
		}
		w.Header().Set(wasmhttp.FetchResultHeader, wasmhttp.FetchResultUpstream)
		w.WriteHeader(resp.StatusCode)
		if _, err := w.Write(body); err != nil {
			log.Printf("write proxied response failed: %v", err)
		}
	})
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func clientAddressLogFields(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}

	return fmt.Sprintf("remote_addr=%q remote_ip=%q x_forwarded_for=%q x_real_ip=%q forwarded=%q",
		r.RemoteAddr,
		remoteIP,
		r.Header.Get("X-Forwarded-For"),
		r.Header.Get("X-Real-IP"),
		r.Header.Get("Forwarded"),
	)
}

func validateFetchURL(u *url.URL) error {
	if u == nil || u.Host == "" {
		return fmt.Errorf("url must be absolute")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if u.User != nil {
		return fmt.Errorf("url must not include user info")
	}
	return nil
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, bool, error) {
	if maxBytes < 1 {
		return nil, false, fmt.Errorf("max bytes must be positive")
	}

	body, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > maxBytes {
		return nil, true, nil
	}
	return body, false, nil
}
