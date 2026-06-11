// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChatWritesReplayableAuditPair(t *testing.T) {
	const apiKey = "secret-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"Message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	now := time.Date(2026, time.June, 11, 12, 34, 56, 789, time.Local)
	client := newAuditTestClient(t, server.URL, apiKey, dir, func() time.Time { return now })
	req := &ChatRequest{
		Model: "test-model",
		Messages: []Message{
			MakeUserMessage("quote: \"hello\"\nnext"),
		},
	}

	if _, err := client.Chat(req); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	cfgPaths := auditFiles(t, dir, "*.cfg")
	jsonPaths := auditFiles(t, dir, "*.json")
	if len(cfgPaths) != 1 || len(jsonPaths) != 1 {
		t.Fatalf("audit files = %v, %v; want one pair", cfgPaths, jsonPaths)
	}
	if strings.TrimSuffix(cfgPaths[0], ".cfg") != strings.TrimSuffix(jsonPaths[0], ".json") {
		t.Fatalf("audit pair stems differ: %q and %q", cfgPaths[0], jsonPaths[0])
	}
	if filepath.Base(filepath.Dir(cfgPaths[0])) != "2026-06-11" {
		t.Fatalf("audit day directory = %q", filepath.Dir(cfgPaths[0]))
	}

	cfgBytes, err := os.ReadFile(cfgPaths[0])
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(cfgBytes)
	for _, want := range []string{
		"# curl -K " + filepath.Base(cfgPaths[0]) + " \\",
		`#   -H "Authorization: Bearer $TOKEN"`,
		`request = "POST"`,
		`url = "` + server.URL + `/chat/completions"`,
		`header = "Content-Type: application/json"`,
		`data-binary = "{\"model\":\"test-model\"`,
		`quote: \\\"hello\\\"\\nnext`,
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("CURL config missing %q:\n%s", want, cfg)
		}
	}
	if strings.Contains(cfg, apiKey) {
		t.Fatalf("CURL config contains API key:\n%s", cfg)
	}

	var record auditRecord
	readAuditRecord(t, jsonPaths[0], &record)
	if record.Outcome != "success" || record.Attempt != 1 {
		t.Fatalf("audit record = %+v", record)
	}
	if record.Timestamp != now.Format(time.RFC3339Nano) {
		t.Fatalf("timestamp = %q, want %q", record.Timestamp, now.Format(time.RFC3339Nano))
	}
}

func TestChatWritesAuditPairForEveryRetryAttempt(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "retry", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"Message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	now := time.Date(2026, time.June, 11, 12, 34, 56, 0, time.Local)
	client := newAuditTestClient(t, server.URL, "token", dir, func() time.Time { return now })

	if _, err := client.Chat(&ChatRequest{Model: "test-model"}); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	cfgPaths := auditFiles(t, dir, "*.cfg")
	jsonPaths := auditFiles(t, dir, "*.json")
	if len(cfgPaths) != 2 || len(jsonPaths) != 2 {
		t.Fatalf("audit files = %v, %v; want two pairs", cfgPaths, jsonPaths)
	}
	if filepath.Base(cfgPaths[0]) >= filepath.Base(cfgPaths[1]) {
		t.Fatalf("filenames are not ordered: %q, %q", cfgPaths[0], cfgPaths[1])
	}

	var first, second auditRecord
	readAuditRecord(t, jsonPaths[0], &first)
	readAuditRecord(t, jsonPaths[1], &second)
	if first.Attempt != 1 || first.Outcome != "rate_limited" {
		t.Fatalf("first audit record = %+v", first)
	}
	if second.Attempt != 2 || second.Outcome != "success" {
		t.Fatalf("second audit record = %+v", second)
	}
}

func TestAuditLoggersConcurrentAttemptsHaveUniqueOrderedStems(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.June, 11, 12, 34, 56, 0, time.Local)

	const count = 20
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger, err := newAuditLogger(auditLoggerConfig{
				enabled: true,
				dir:     dir,
				now:     func() time.Time { return now },
			})
			if err != nil {
				t.Error(err)
				return
			}
			attempt := logger.begin(http.MethodPost, "https://example.com", []byte(`{}`))
			attempt.finish(auditRecord{Outcome: "success"})
		}()
	}
	wg.Wait()

	cfgPaths := auditFiles(t, dir, "*.cfg")
	jsonPaths := auditFiles(t, dir, "*.json")
	if len(cfgPaths) != count || len(jsonPaths) != count {
		t.Fatalf("audit file counts = %d cfg, %d json; want %d each", len(cfgPaths), len(jsonPaths), count)
	}
	for i := range cfgPaths {
		if strings.TrimSuffix(cfgPaths[i], ".cfg") != strings.TrimSuffix(jsonPaths[i], ".json") {
			t.Fatalf("audit pair %d stems differ: %q and %q", i, cfgPaths[i], jsonPaths[i])
		}
	}
}

func TestChatDoesNotAuditDisabledOrPreflightFailures(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[]}`))
		}))
		defer server.Close()

		dir := t.TempDir()
		client := newAuditTestClient(t, server.URL, "token", dir, time.Now)
		client.audit.enabled = false
		if _, err := client.Chat(&ChatRequest{}); err != nil {
			t.Fatalf("Chat() error = %v", err)
		}
		if files := auditFiles(t, dir, "*"); len(files) != 0 {
			t.Fatalf("disabled audit wrote files: %v", files)
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		dir := t.TempDir()
		client := newAuditTestClient(t, "https://example.com", "token", dir, time.Now)
		_, err := client.Chat(&ChatRequest{ResponseFormat: NewResponseFormat(func() {})})
		if err == nil {
			t.Fatal("Chat() error = nil, want marshal error")
		}
		if files := auditFiles(t, dir, "*"); len(files) != 0 {
			t.Fatalf("preflight failure wrote files: %v", files)
		}
	})
}

func TestChatContinuesWhenAuditWriteFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	dir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(dir, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := newAuditTestClient(t, server.URL, "token", dir, time.Now)
	if _, err := client.Chat(&ChatRequest{}); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func newAuditTestClient(t *testing.T, baseURL, apiKey, dir string, now func() time.Time) *Client {
	t.Helper()
	logger, err := newAuditLogger(auditLoggerConfig{enabled: true, dir: dir, now: now})
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(apiKey, WithBaseUrl(baseURL))
	client.audit = logger
	return client
}

func auditFiles(t *testing.T, dir, pattern string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*", pattern))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return paths
}

func readAuditRecord(t *testing.T, path string, record *auditRecord) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, record); err != nil {
		t.Fatalf("unmarshal %q: %v", path, err)
	}
}
