// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/spf13/cobra"
)

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
	mux.Handle("/api/fetch", fetchHandler(maxBytes))
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	return mux
}

func fetchHandler(maxBytes int64) http.Handler {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := validateFetchURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		log.Println("proxy GET", targetURL.String())
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL.String(), nil)
		if err != nil {
			http.Error(w, "create upstream request failed", http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", config.UserAgent())

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		body, tooLarge, err := readLimited(resp.Body, maxBytes)
		if err != nil {
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
		w.WriteHeader(resp.StatusCode)
		if _, err := w.Write(body); err != nil {
			log.Printf("write proxied response failed: %v", err)
		}
	})
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
