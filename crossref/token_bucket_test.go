// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTokenBucketAllowsBurstThenLimits(t *testing.T) {
	bucket := &tokenBucket{tokens: burstSize, lastRefill: time.Now()}
	for i := 0; i < burstSize; i++ {
		delayed, err := bucket.wait(context.Background())
		if err != nil {
			t.Fatalf("burst request %d: %v", i, err)
		}
		if delayed {
			t.Fatalf("burst request %d was delayed", i)
		}
	}

	started := time.Now()
	delayed, err := bucket.wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !delayed {
		t.Fatal("request after burst was not delayed")
	}
	if elapsed := time.Since(started); elapsed < 70*time.Millisecond {
		t.Fatalf("request after burst waited only %s", elapsed)
	}
}

func TestTokenBucketWaitHonorsCancellation(t *testing.T) {
	bucket := &tokenBucket{tokens: 0, lastRefill: time.Now()}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, err := bucket.wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
