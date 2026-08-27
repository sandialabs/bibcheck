// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package ratelimit

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

func TestClientLimitsConcurrentRequests(t *testing.T) {
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
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
		return response(http.StatusNoContent), nil
	})}, Policy{RequestsPerSecond: 1000, Burst: 4, MaxConcurrent: 2}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for initial requests")
		}
	}
	select {
	case <-started:
		t.Fatal("too many requests reached the transport")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	wg.Wait()
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", got)
	}
}

func TestClientCancellationWhileWaitingForTokenSkipsUpstream(t *testing.T) {
	var calls atomic.Int32
	client := NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return response(http.StatusNoContent), nil
	})}, Policy{RequestsPerSecond: 1, Burst: 1, MaxConcurrent: 1}, nil)

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport called %d times, want 1", got)
	}
}

func TestClientCancellationWhileWaitingForConcurrencySkipsUpstream(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	client := NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		close(started)
		<-release
		return response(http.StatusNoContent), nil
	})}, Policy{RequestsPerSecond: 100, Burst: 2, MaxConcurrent: 1}, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if _, err := client.Do(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	close(release)
	<-done
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport called %d times, want 1", got)
	}
}

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
}
