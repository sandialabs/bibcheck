// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"net/http"
	"time"

	"github.com/sandialabs/bibcheck/config"
)

type Client struct {
	apiKey     string
	baseUrl    string
	httpClient *http.Client
	audit      *auditLogger
}

type ClientOpt func(*Client)

func NewClient(apiKey string, options ...ClientOpt) *Client {
	settings := config.Runtime()
	audit, err := newAuditLogger(auditLoggerConfig{
		enabled: settings.OpenAIAuditEnable,
		dir:     settings.OpenAIAuditDir,
		now:     time.Now,
	})
	if err != nil {
		audit = nil
	}

	c := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		audit: audit,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

func WithBaseUrl(baseUrl string) ClientOpt {
	return func(c *Client) {
		c.baseUrl = baseUrl
	}
}

func WithTimeout(t time.Duration) ClientOpt {
	return func(c *Client) {
		c.httpClient.Timeout = t
	}
}

func WithAuditEnabled(enabled bool) ClientOpt {
	return func(c *Client) {
		if c.audit == nil {
			audit, err := newAuditLogger(auditLoggerConfig{
				enabled: enabled,
				now:     time.Now,
			})
			if err == nil {
				c.audit = audit
			}
			return
		}
		c.audit.enabled = enabled
	}
}

func WithAuditDir(dir string) ClientOpt {
	return func(c *Client) {
		audit, err := newAuditLogger(auditLoggerConfig{
			enabled: c.auditEnabledOrDefault(),
			dir:     dir,
			now:     time.Now,
		})
		if err == nil {
			c.audit = audit
		}
	}
}

func (c *Client) auditEnabledOrDefault() bool {
	if c.audit != nil {
		return c.audit.enabled
	}
	return true
}
