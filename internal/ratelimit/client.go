// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package ratelimit

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"
)

// Policy configures request starts and in-flight request concurrency.
type Policy struct {
	RequestsPerSecond float64
	Burst             int
	MaxConcurrent     int
}

// Client applies a Policy before forwarding requests to an HTTP client. It is
// safe for concurrent use.
type Client struct {
	httpClient *http.Client
	delay      func(time.Duration)
	semaphore  chan struct{}
	bucket     tokenBucket
}

// NewClient creates a rate-limited HTTP client. Invalid policies panic because
// policies are static application configuration rather than user input.
func NewClient(httpClient *http.Client, policy Policy, delay func(time.Duration)) *Client {
	if httpClient == nil {
		panic("ratelimit: nil HTTP client")
	}
	if policy.RequestsPerSecond <= 0 {
		panic("ratelimit: requests per second must be positive")
	}
	if policy.Burst < 1 {
		panic("ratelimit: burst must be positive")
	}
	if policy.MaxConcurrent < 1 {
		panic("ratelimit: max concurrency must be positive")
	}

	return &Client{
		httpClient: httpClient,
		delay:      delay,
		semaphore:  make(chan struct{}, policy.MaxConcurrent),
		bucket: tokenBucket{
			rate:       policy.RequestsPerSecond,
			capacity:   float64(policy.Burst),
			tokens:     float64(policy.Burst),
			lastRefill: time.Now(),
		},
	}
}

// Do waits for both concurrency capacity and a rate-limit token, then performs
// the request. Waiting honors the request context.
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

	rateDelayed, err := c.bucket.wait(req.Context())
	if err != nil {
		return nil, err
	}
	if (delayed || rateDelayed) && c.delay != nil {
		c.delay(time.Since(waitStarted))
	}
	return c.httpClient.Do(req)
}

type tokenBucket struct {
	mu         sync.Mutex
	rate       float64
	capacity   float64
	tokens     float64
	lastRefill time.Time
}

func (b *tokenBucket) wait(ctx context.Context) (bool, error) {
	delayed := false
	for {
		now := time.Now()
		b.mu.Lock()
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = math.Min(b.capacity, b.tokens+elapsed*b.rate)
		b.lastRefill = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return delayed, nil
		}
		wait := time.Duration((1 - b.tokens) / b.rate * float64(time.Second))
		b.mu.Unlock()

		delayed = true
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return false, ctx.Err()
		}
	}
}
