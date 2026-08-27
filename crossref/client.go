// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"log"
	"net/http"
	"time"

	"github.com/sandialabs/bibcheck/internal/ratelimit"
	"github.com/sandialabs/bibcheck/internal/wasmhttp"
)

const (
	defaultTimeout    = 30 * time.Second
	requestsPerSecond = 10
	burstSize         = 3
	maxConcurrent     = 3
)

// Client is a rate-limited client for the Crossref API. A Client is safe for
// concurrent use and should be shared by all work in one process.
type Client struct {
	httpClient *http.Client
	delay      func(time.Duration)
	limiter    *ratelimit.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the HTTP client used for upstream requests.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

// WithDelayCallback registers a callback for requests delayed by rate limiting.
func WithDelayCallback(callback func(time.Duration)) Option {
	return func(c *Client) { c.delay = callback }
}

// NewClient returns a Crossref client limited to 10 request starts per second,
// with a burst of three and at most three concurrent upstream requests.
func NewClient(options ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		delay: func(delay time.Duration) {
			log.Printf("Crossref request delayed by rate limit: %s", delay)
		},
	}
	for _, option := range options {
		option(c)
	}
	c.limiter = ratelimit.NewClient(c.httpClient, ratelimit.Policy{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burstSize,
		MaxConcurrent:     maxConcurrent,
	}, c.delay)
	return c
}

// Do performs a rate-limited HTTP request. WASM requests rely on the shared
// fetch proxy's limiter instead of applying a second browser-local limit.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if wasmhttp.UsesFetchProxy() {
		return c.httpClient.Do(req)
	}
	return c.limiter.Do(req)
}
