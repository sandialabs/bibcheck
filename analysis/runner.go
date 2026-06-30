// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analysis

import (
	"context"
	"fmt"
	"sync"

	"github.com/sandialabs/bibcheck/lookup"
)

const DefaultWorkers = 4

type Status string

const (
	StatusWaiting   Status = "waiting"
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type Stage string

const (
	StageExtraction Stage = "extraction"
	StageLookup     Stage = "lookup"
	StageSummary    Stage = "summary"
)

type Summary struct {
	Mismatch bool
	Comment  string
}

type Entry struct {
	ID int

	ExtractionStatus Status
	Text             string
	ExtractionError  error

	LookupStatus Status
	Result       *lookup.Result
	LookupError  error

	SummaryStatus Status
	Summary       Summary
	SummaryError  error
}

func (e Entry) Terminal() bool {
	return e.ExtractionStatus == StatusError || e.LookupStatus == StatusError ||
		e.SummaryStatus == StatusCompleted || e.SummaryStatus == StatusError
}

type Snapshot struct {
	Entries   []Entry
	Completed int
	Done      bool
}

type Config struct {
	EntryIDs  []int
	Workers   int
	Extract   func(int) (string, error)
	Lookup    func(string) (*lookup.Result, error)
	Summarize func(*lookup.Result) (Summary, error)
	Progress  func(Snapshot)
}

type job struct {
	index int
	stage Stage
	entry Entry
}

type table struct {
	mu        sync.Mutex
	cond      *sync.Cond
	entries   []Entry
	completed int
	stopped   bool
}

func newTable(ids []int) *table {
	t := &table{entries: make([]Entry, len(ids))}
	t.cond = sync.NewCond(&t.mu)
	for i, id := range ids {
		t.entries[i] = Entry{
			ID:               id,
			ExtractionStatus: StatusPending,
			LookupStatus:     StatusWaiting,
			SummaryStatus:    StatusWaiting,
		}
	}
	return t
}

// claim returns the first available operation in bibliography order.
func (t *table) claim(ctx context.Context) (job, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for {
		if ctx.Err() != nil {
			t.stopped = true
		}
		if t.stopped || t.completed == len(t.entries) {
			return job{}, false
		}
		for i := range t.entries {
			e := &t.entries[i]
			var stage Stage
			switch {
			case e.ExtractionStatus == StatusPending:
				e.ExtractionStatus = StatusActive
				stage = StageExtraction
			case e.ExtractionStatus == StatusCompleted && e.LookupStatus == StatusPending:
				e.LookupStatus = StatusActive
				stage = StageLookup
			case e.LookupStatus == StatusCompleted && e.SummaryStatus == StatusPending:
				e.SummaryStatus = StatusActive
				stage = StageSummary
			default:
				continue
			}
			return job{index: i, stage: stage, entry: *e}, true
		}
		t.cond.Wait()
	}
}

func (t *table) complete(j job, value any, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	e := &t.entries[j.index]
	switch j.stage {
	case StageExtraction:
		if err != nil {
			e.ExtractionStatus = StatusError
			e.ExtractionError = err
			t.completed++
		} else {
			e.ExtractionStatus = StatusCompleted
			e.Text = value.(string)
			e.LookupStatus = StatusPending
		}
	case StageLookup:
		if err != nil {
			e.LookupStatus = StatusError
			e.LookupError = err
			t.completed++
		} else {
			e.LookupStatus = StatusCompleted
			e.Result = value.(*lookup.Result)
			e.SummaryStatus = StatusPending
		}
	case StageSummary:
		if err != nil {
			e.SummaryStatus = StatusError
			e.SummaryError = err
		} else {
			e.SummaryStatus = StatusCompleted
			e.Summary = value.(Summary)
		}
		t.completed++
	}
	t.cond.Broadcast()
}

func (t *table) stop() {
	t.mu.Lock()
	t.stopped = true
	t.cond.Broadcast()
	t.mu.Unlock()
}

func (t *table) snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	entries := make([]Entry, len(t.entries))
	copy(entries, t.entries)
	return Snapshot{
		Entries:   entries,
		Completed: t.completed,
		Done:      t.completed == len(t.entries),
	}
}

func Run(ctx context.Context, cfg Config) (Snapshot, error) {
	if len(cfg.EntryIDs) == 0 {
		return Snapshot{}, fmt.Errorf("no bibliography entries")
	}
	if cfg.Extract == nil || cfg.Lookup == nil || cfg.Summarize == nil {
		return Snapshot{}, fmt.Errorf("incomplete analysis configuration")
	}
	workers := cfg.Workers
	if workers < 1 {
		workers = DefaultWorkers
	}
	if workers > len(cfg.EntryIDs) {
		workers = len(cfg.EntryIDs)
	}

	t := newTable(cfg.EntryIDs)
	updates := make(chan struct{}, workers*2+1)
	var dispatch sync.WaitGroup
	dispatch.Add(1)
	go func() {
		defer dispatch.Done()
		for range updates {
			if cfg.Progress != nil {
				cfg.Progress(t.snapshot())
			}
		}
	}()
	notify := func() {
		select {
		case updates <- struct{}{}:
		default:
			// A queued notification will observe the latest table snapshot.
		}
	}
	notify()

	stopWatching := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			t.stop()
		case <-stopWatching:
		}
	}()

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for {
				j, ok := t.claim(ctx)
				if !ok {
					return
				}
				notify()
				var value any
				var err error
				switch j.stage {
				case StageExtraction:
					value, err = cfg.Extract(j.entry.ID)
				case StageLookup:
					value, err = cfg.Lookup(j.entry.Text)
				case StageSummary:
					value, err = cfg.Summarize(j.entry.Result)
				}
				t.complete(j, value, err)
				notify()
			}
		}()
	}
	wg.Wait()
	close(stopWatching)
	close(updates)
	dispatch.Wait()

	result := t.snapshot()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}
