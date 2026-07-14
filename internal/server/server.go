// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"context"
	"crypto/sha256"
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
)

const (
	DefaultAddr          = "localhost:8080"
	DefaultWebDir        = "web/static"
	DefaultFetchMaxBytes = 25 * 1024 * 1024
)

const fetchUpstreamTimeout = 15 * time.Second

const livenessPath string = "/livez"

type Options struct {
	Addr          string
	WebDir        string
	FetchMaxBytes int64
}

func Handler(options Options) (http.Handler, error) {
	options = options.withDefaults()
	if options.FetchMaxBytes < 1 {
		return nil, fmt.Errorf("fetch-max-bytes must be positive")
	}
	return serveMux(options.WebDir, options.FetchMaxBytes), nil
}

func Run(options Options) error {
	options = options.withDefaults()
	handler, err := Handler(options)
	if err != nil {
		return err
	}
	log.Printf("serving %s at http://%s", options.WebDir, options.Addr)
	return http.ListenAndServe(options.Addr, handler)
}

func (o Options) withDefaults() Options {
	if o.Addr == "" {
		o.Addr = DefaultAddr
	}
	if o.WebDir == "" {
		o.WebDir = DefaultWebDir
	}
	if o.FetchMaxBytes == 0 {
		o.FetchMaxBytes = DefaultFetchMaxBytes
	}
	return o
}

func serveMux(staticDir string, maxBytes int64) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(livenessPath, livenessHandler())
	mux.Handle("/api/fetch", fetchHandler(maxBytes))
	mux.Handle("/", wasmBundleLogHandler(versionedFileServer(http.Dir(staticDir))))
	return mux
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

func fingerprintedName(root http.Dir, name string) (string, error) {
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s.%x%s", base, hash.Sum(nil), ext), nil
}

func (h versionedStaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, ok := staticFileName(r.URL.Path)
	if !ok {
		h.fallback.ServeHTTP(w, r)
		return
	}
	if name == "index.html" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		h.serveIndex(w, r)
		return
	}
	if source, ok := h.assets[name]; ok {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		request := r.Clone(r.Context())
		request.URL.Path = "/" + source
		request.URL.RawPath = ""
		h.fallback.ServeHTTP(w, request)
		return
	}
	if name == "app.wasm" || name == "wasm_exec.js" {
		w.Header().Set("Cache-Control", "no-store")
	}
	h.fallback.ServeHTTP(w, r)
}

func (h versionedStaticHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	file, err := h.root.Open("index.html")
	if err != nil {
		h.fallback.ServeHTTP(w, r)
		return
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read index", http.StatusInternalServerError)
		return
	}
	contents := string(body)
	for original, versioned := range h.references {
		contents = strings.ReplaceAll(contents, original, versioned)
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(contents)))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.WriteString(w, contents)
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

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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

type versionedStaticHandler struct {
	root       http.Dir
	fallback   http.Handler
	assets     map[string]string
	references map[string]string
}

func versionedFileServer(root http.Dir) http.Handler {
	h := versionedStaticHandler{
		root:       root,
		fallback:   compressedFileServer(root),
		assets:     make(map[string]string),
		references: make(map[string]string),
	}
	index, err := root.Open("index.html")
	if err != nil {
		log.Printf("index.html not found error=%q", err)
	} else {
		_ = index.Close()
		log.Printf("index.html found")
	}
	for _, name := range []string{"app.wasm", "wasm_exec.js"} {
		versionedName, err := fingerprintedName(root, name)
		if err != nil {
			log.Printf("static asset unavailable name=%q versioned_name=%q error=%q", name, versionedName, err)
			continue
		}
		h.assets[versionedName] = name
		h.references["/"+name] = "/" + versionedName
		log.Printf("static asset registered name=%q versioned_name=%q",
			name,
			versionedName,
		)
	}
	return h
}
