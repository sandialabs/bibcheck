package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/sandialabs/bibcheck/internal/wasmhttp"
	"github.com/sandialabs/bibcheck/version"
)

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
		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = defaultUserAgent()
		}
		req.Header.Set("User-Agent", userAgent)

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

func defaultUserAgent() string {
	return "bibcheck / " + version.String() + " github.com/sandialabs/bibcheck"
}
