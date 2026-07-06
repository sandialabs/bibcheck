// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const (
	defaultTimeout       = 30 * time.Second
	defaultStartInterval = 334 * time.Millisecond
	maxConcurrent        = 3
)

// Client is a rate-limited client for the Crossref API. A Client is safe for
// concurrent use and should be shared by all work in one process.
type Client struct {
	httpClient    *http.Client
	startInterval time.Duration
	delay         func(time.Duration)
	semaphore     chan struct{}
	startMu       sync.Mutex
	lastStart     time.Time
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

// NewClient returns a Crossref client limited to one request start every 334ms
// and at most three concurrent upstream requests.
func NewClient(options ...Option) *Client {
	c := &Client{
		httpClient:    &http.Client{Timeout: defaultTimeout},
		startInterval: defaultStartInterval,
		semaphore:     make(chan struct{}, maxConcurrent),
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

	rateDelayed, err := c.waitForStart(req.Context())
	if err != nil {
		return nil, err
	}
	if (delayed || rateDelayed) && c.delay != nil {
		c.delay(time.Since(waitStarted))
	}
	return c.httpClient.Do(req)
}

func (c *Client) waitForStart(ctx context.Context) (bool, error) {
	c.startMu.Lock()
	defer c.startMu.Unlock()

	delay := time.Until(c.lastStart.Add(c.startInterval))
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	c.lastStart = time.Now()
	return delay > 0, nil
}
