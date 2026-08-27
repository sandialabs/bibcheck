// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package arxiv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientUsesArxivRatePolicy(t *testing.T) {
	if requestsPerSecond != 1.0/3.0 {
		t.Fatalf("requests per second = %v, want %v", requestsPerSecond, 1.0/3.0)
	}
	if burstSize != 1 {
		t.Fatalf("burst size = %d, want 1", burstSize)
	}
	if maxConcurrent != 1 {
		t.Fatalf("max concurrency = %d, want 1", maxConcurrent)
	}
}

func TestClientCancellationWhileWaitingForNextStartSkipsUpstream(t *testing.T) {
	var calls atomic.Int32
	var delayed atomic.Bool
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return arxivResponse(http.StatusNoContent), nil
		})}),
		WithDelayCallback(func(time.Duration) { delayed.Store(true) }),
	)

	req, _ := http.NewRequest(http.MethodGet, "http://export.arxiv.org/api/query", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, "http://export.arxiv.org/api/query", nil)
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport called %d times, want 1", got)
	}
	if delayed.Load() {
		t.Fatal("delay callback called for a canceled request")
	}
}

func TestClientLimitsConcurrentRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			close(started)
			<-release
			return arxivResponse(http.StatusNoContent), nil
		})}),
		WithDelayCallback(func(time.Duration) {}),
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		req, _ := http.NewRequest(http.MethodGet, "http://export.arxiv.org/api/query", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://export.arxiv.org/api/query", nil)
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	close(release)
	<-done
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport called %d times, want 1", got)
	}
}

func arxivResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
}
