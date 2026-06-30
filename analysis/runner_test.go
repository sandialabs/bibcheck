// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analysis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sandialabs/bibcheck/lookup"
)

func TestRunSkipsActiveEarlierEntry(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), Config{
			EntryIDs: []int{1, 2},
			Workers:  2,
			Extract: func(id int) (string, error) {
				if id == 1 {
					close(firstStarted)
					<-releaseFirst
				} else {
					close(secondStarted)
				}
				return fmt.Sprintf("entry %d", id), nil
			},
			Lookup: func(text string) (*lookup.Result, error) {
				return &lookup.Result{Text: text}, nil
			},
			Summarize: func(*lookup.Result) (Summary, error) { return Summary{}, nil },
		})
		done <- err
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("entry 1 extraction did not start")
	}
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("entry 2 was not claimed while entry 1 was active")
	}
	close(releaseFirst)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsStageFailureAsTerminal(t *testing.T) {
	result, err := Run(context.Background(), Config{
		EntryIDs: []int{1},
		Workers:  1,
		Extract: func(int) (string, error) {
			return "", fmt.Errorf("bad extraction")
		},
		Lookup:    func(string) (*lookup.Result, error) { t.Fatal("unexpected lookup"); return nil, nil },
		Summarize: func(*lookup.Result) (Summary, error) { t.Fatal("unexpected summary"); return Summary{}, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Done || result.Completed != 1 || result.Entries[0].ExtractionError == nil {
		t.Fatalf("unexpected final snapshot: %+v", result)
	}
}

func TestRunSerializesProgressCallbacks(t *testing.T) {
	var mu sync.Mutex
	inCallback := false
	concurrent := false
	_, err := Run(context.Background(), Config{
		EntryIDs: []int{1, 2, 3},
		Workers:  3,
		Extract:  func(id int) (string, error) { return fmt.Sprint(id), nil },
		Lookup:   func(text string) (*lookup.Result, error) { return &lookup.Result{Text: text}, nil },
		Summarize: func(*lookup.Result) (Summary, error) {
			return Summary{}, nil
		},
		Progress: func(Snapshot) {
			mu.Lock()
			if inCallback {
				concurrent = true
			}
			inCallback = true
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			inCallback = false
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if concurrent {
		t.Fatal("progress callback ran concurrently")
	}
}

func TestRunCancellationStopsNewClaims(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	resultCh := make(chan Snapshot, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := Run(ctx, Config{
			EntryIDs: []int{1, 2},
			Workers:  1,
			Extract: func(id int) (string, error) {
				close(started)
				<-release
				return fmt.Sprint(id), nil
			},
			Lookup:    func(string) (*lookup.Result, error) { t.Fatal("unexpected lookup"); return nil, nil },
			Summarize: func(*lookup.Result) (Summary, error) { t.Fatal("unexpected summary"); return Summary{}, nil },
		})
		resultCh <- result
		errCh <- err
	}()
	<-started
	cancel()
	close(release)
	result := <-resultCh
	if err := <-errCh; err != context.Canceled {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if result.Completed != 0 {
		t.Fatalf("completed = %d, want 0", result.Completed)
	}
}
