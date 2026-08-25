// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientCancellationWhileWaitingForTokenSkipsUpstream(t *testing.T) {
	var calls atomic.Int32
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return response(http.StatusNoContent, ""), nil
		})}),
		WithDelayCallback(func(time.Duration) {}),
	)
	for i := 0; i < burstSize; i++ {
		req, err := http.NewRequest(http.MethodGet, "https://api.crossref.org/v1/works", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.crossref.org/v1/works", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if got := calls.Load(); got != burstSize {
		t.Fatalf("transport called %d times, want %d", got, burstSize)
	}
}

func TestClientLimitsConcurrentRequests(t *testing.T) {
	started := make(chan struct{}, 6)
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
		case <-req.Context().Done():
			active.Add(-1)
			return nil, req.Context().Err()
		}
		active.Add(-1)
		return response(http.StatusNoContent, ""), nil
	})
	client := NewClient(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithDelayCallback(func(time.Duration) {}),
	)

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, "https://api.crossref.org/v1/works", nil)
			if err != nil {
				t.Error(err)
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}

	for i := 0; i < maxConcurrent; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for initial requests")
		}
	}
	select {
	case <-started:
		t.Fatal("more than three requests reached the transport concurrently")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	wg.Wait()

	if got := maximum.Load(); got != maxConcurrent {
		t.Fatalf("maximum concurrency = %d, want %d", got, maxConcurrent)
	}
}

func TestClientCancellationWhileWaitingForConcurrency(t *testing.T) {
	var calls atomic.Int32
	client := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return response(http.StatusNoContent, ""), nil
	})}))
	for i := 0; i < maxConcurrent; i++ {
		client.semaphore <- struct{}{}
		defer func() { <-client.semaphore }()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.crossref.org/v1/works", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("transport called %d times", got)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
