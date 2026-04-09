// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sandialabs/bibcheck/config"
)

type auditLoggerConfig struct {
	enabled bool
	dir     string
	now     func() time.Time
}

type auditLogger struct {
	enabled bool
	dir     string
	now     func() time.Time
	mu      sync.Mutex
}

type auditRecord struct {
	Timestamp      string   `json:"ts"`
	Method         string   `json:"method"`
	URL            string   `json:"url"`
	Model          string   `json:"model,omitempty"`
	Attempt        int      `json:"attempt"`
	MaxAttempts    int      `json:"max_attempts"`
	DurationMS     int64    `json:"duration_ms"`
	RequestBytes   int      `json:"request_bytes"`
	ResponseBytes  int      `json:"response_bytes"`
	StatusCode     int      `json:"status_code,omitempty"`
	Outcome        string   `json:"outcome"`
	CorrelationIDs []string `json:"correlation_ids,omitempty"`
	Error          string   `json:"error,omitempty"`
}

func newAuditLogger(cfg auditLoggerConfig) (*auditLogger, error) {
	now := cfg.now
	if now == nil {
		now = time.Now
	}

	dir := cfg.dir
	if dir == "" {
		resolved, err := config.OpenAIAuditDir(config.Settings{})
		if err != nil {
			return nil, err
		}
		dir = resolved
	}

	return &auditLogger{
		enabled: cfg.enabled,
		dir:     dir,
		now:     now,
	}, nil
}

func (a *auditLogger) write(record auditRecord) {
	if a == nil || !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now
	if now == nil {
		now = time.Now
	}

	record.Timestamp = now().Format(time.RFC3339Nano)

	if err := os.MkdirAll(a.dir, 0o700); err != nil {
		log.Printf("openai audit: mkdir failed dir=%q err=%v", a.dir, err)
		return
	}

	path := filepath.Join(a.dir, now().Format("2006-01-02")+".ndjson")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("openai audit: open failed path=%q err=%v", path, err)
		return
	}
	defer f.Close()

	payload, err := json.Marshal(record)
	if err != nil {
		log.Printf("openai audit: marshal failed err=%v", err)
		return
	}
	payload = append(payload, '\n')

	if _, err := f.Write(payload); err != nil {
		log.Printf("openai audit: write failed path=%q err=%v", path, err)
		return
	}
}

func newAuditRecord(method, url string, req *ChatRequest, requestBytes int, attempt int) auditRecord {
	record := auditRecord{
		Method:       method,
		URL:          url,
		Attempt:      attempt,
		MaxAttempts:  maxRetries + 1,
		RequestBytes: requestBytes,
	}
	if req != nil {
		record.Model = req.Model
	}
	return record
}

func outcomeForStatus(statusCode int, hasRetryAfter bool) string {
	if statusCode == httpStatusTooManyRequests || hasRetryAfter {
		return "rate_limited"
	}
	return "http_error"
}

const httpStatusTooManyRequests = 429

func formatAuditError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
