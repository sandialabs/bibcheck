// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	enabled  bool
	dir      string
	now      func() time.Time
	mu       sync.Mutex
	sequence uint64
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

type auditAttempt struct {
	auditPath string
	timestamp string
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

func (a *auditLogger) begin(method, url string, body []byte) *auditAttempt {
	if a == nil || !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.currentTime()
	dir := filepath.Join(a.dir, now.Format("2006-01-02"))

	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("openai audit: mkdir failed dir=%q err=%v", dir, err)
		return nil
	}

	var auditPath string
	for {
		a.sequence++
		stem := fmt.Sprintf("%s-%04d", now.Format("20060102T150405.000000000"), a.sequence)
		bodyPath := filepath.Join(dir, stem+".body.json")
		err := writeFileExclusive(bodyPath, body)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			log.Printf("openai audit: write failed path=%q err=%v", bodyPath, err)
			return nil
		}

		cfgPath := filepath.Join(dir, stem+".cfg")
		err = writeFileExclusive(cfgPath, curlConfig(filepath.Base(cfgPath), filepath.Base(bodyPath), method, url))
		if err != nil {
			log.Printf("openai audit: write failed path=%q err=%v", cfgPath, err)
			if removeErr := os.Remove(bodyPath); removeErr != nil {
				log.Printf("openai audit: cleanup failed path=%q err=%v", bodyPath, removeErr)
			}
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return nil
		}

		auditPath = filepath.Join(dir, stem+".audit.json")
		break
	}

	return &auditAttempt{
		auditPath: auditPath,
		timestamp: now.Format(time.RFC3339Nano),
	}
}

func (a *auditAttempt) finish(record auditRecord) {
	if a == nil {
		return
	}

	record.Timestamp = a.timestamp
	payload, err := json.Marshal(record)
	if err != nil {
		log.Printf("openai audit: marshal failed err=%v", err)
		return
	}
	payload = append(payload, '\n')

	if err := writeFileExclusive(a.auditPath, payload); err != nil {
		log.Printf("openai audit: write failed path=%q err=%v", a.auditPath, err)
	}
}

func writeFileExclusive(path string, payload []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func (a *auditLogger) currentTime() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}

func curlConfig(filename, bodyFilename, method, url string) []byte {
	var b strings.Builder
	b.WriteString("# Replay with:\n")
	b.WriteString("# curl -K ")
	b.WriteString(filename)
	b.WriteString(" \\\n")
	b.WriteString("#   -H \"Authorization: Bearer $TOKEN\"\n\n")
	writeCurlOption(&b, "request", method)
	writeCurlOption(&b, "url", url)
	writeCurlOption(&b, "header", "Content-Type: application/json")
	writeCurlOption(&b, "data-binary", "@"+bodyFilename)
	return []byte(b.String())
}

func writeCurlOption(b *strings.Builder, name, value string) {
	b.WriteString(name)
	b.WriteString(" = \"")
	b.WriteString(escapeCurlConfig(value))
	b.WriteString("\"\n")
}

func escapeCurlConfig(value string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\t", `\t`,
		"\n", `\n`,
		"\r", `\r`,
		"\v", `\v`,
	).Replace(value)
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
