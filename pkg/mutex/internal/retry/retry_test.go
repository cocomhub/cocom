// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"testing"
	"time"
)

func TestRetry_LimitRetryStrategy(t *testing.T) {
	strategy := LimitRetry(LinearBackoff(1*time.Millisecond), 3)
	// Give test a reasonable deadline
	var backoffs []time.Duration
	for range 5 {
		d := strategy.NextBackoff()
		backoffs = append(backoffs, d)
		if d == 0 {
			break
		}
	}
	if len(backoffs) != 4 {
		t.Errorf("expected 3 non-zero + 1 zero = 4 backoffs, got %d: %v", len(backoffs), backoffs)
	} else if backoffs[0] == 0 || backoffs[1] == 0 || backoffs[2] == 0 {
		t.Errorf("first 3 backoffs should be non-zero, got: %v", backoffs[:3])
	} else if backoffs[3] != 0 {
		t.Errorf("4th backoff should be 0 (limit reached), got: %v", backoffs[3])
	}
}

func TestRetry_LinearBackoff(t *testing.T) {
	d := 10 * time.Millisecond
	strategy := LinearBackoff(d)
	b1 := strategy.NextBackoff()
	b2 := strategy.NextBackoff()
	if b1 != d || b2 != d {
		t.Errorf("expected LinearBackoff to always return %v, got %v, %v", d, b1, b2)
	}
}

func TestRetry_NoRetry(t *testing.T) {
	strategy := NoRetry()
	if strategy.NextBackoff() != 0 {
		t.Error("NoRetry should return 0 backoff")
	}
}
