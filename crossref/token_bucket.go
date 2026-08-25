// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"context"
	"math"
	"sync"
	"time"
)

type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
}

func (b *tokenBucket) wait(ctx context.Context) (bool, error) {
	delayed := false
	for {
		now := time.Now()
		b.mu.Lock()
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = math.Min(burstSize, b.tokens+elapsed*requestsPerSecond)
		b.lastRefill = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return delayed, nil
		}
		wait := time.Duration((1 - b.tokens) / requestsPerSecond * float64(time.Second))
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
