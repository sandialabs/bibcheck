// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"log"
	"net/http"
	"time"
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
	semaphore  chan struct{}
	limiter    *tokenBucket
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
		semaphore: make(chan struct{}, maxConcurrent),
		limiter: &tokenBucket{
			tokens:     burstSize,
			lastRefill: time.Now(),
		},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// Do performs a rate-limited HTTP request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	waitStarted := time.Now()
	delayed := false

	select {
	case c.semaphore <- struct{}{}:
	default:
		delayed = true
		select {
		case c.semaphore <- struct{}{}:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	defer func() { <-c.semaphore }()

	rateDelayed, err := c.limiter.wait(req.Context())
	if err != nil {
		return nil, err
	}
	delayed = delayed || rateDelayed

	if delayed && c.delay != nil {
		c.delay(time.Since(waitStarted))
	}
	return c.httpClient.Do(req)
}
