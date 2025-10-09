// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package eventhandler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventHandler_Success(t *testing.T) {
	var processed int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := New(
		ctx,
		1,  // workers
		10, // queue size
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error {
			atomic.AddInt32(&processed, 1)
			return nil
		},
	)

	handler.Push(Event{Payload: "ok", MaxRetries: 1})

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&processed) != 1 {
		t.Fatalf("expected event to be processed once, got %d", processed)
	}
}

func TestEventHandler_RetryUntilSuccess(t *testing.T) {
	var attempts int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := New(
		ctx,
		1,
		10,
		5*time.Millisecond,
		50*time.Millisecond,
		func(_ Event) error {
			n := atomic.AddInt32(&attempts, 1)
			if n < 3 {
				return errors.New("fail")
			}
			return nil
		},
	)

	handler.Push(Event{Payload: "retry", MaxRetries: 5})

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&attempts) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestEventHandler_MaxRetries(t *testing.T) {
	var attempts int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := New(
		ctx,
		1,
		10,
		5*time.Millisecond,
		50*time.Millisecond,
		func(_ Event) error {
			atomic.AddInt32(&attempts, 1)
			return errors.New("always fail")
		},
	)

	handler.Push(Event{Payload: "fail", MaxRetries: 2})

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&attempts) != 3 { // 1 original + 2 retries
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestEventHandler_Backoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := New(
		ctx,
		1,
		1,
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error { return nil },
	)

	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{1, 10 * time.Millisecond, 10 * time.Millisecond},
		{2, 20 * time.Millisecond, 20 * time.Millisecond},
		{3, 40 * time.Millisecond, 40 * time.Millisecond},
		{4, 80 * time.Millisecond, 80 * time.Millisecond},
		{5, 100 * time.Millisecond, 100 * time.Millisecond}, // capped
	}

	for _, tt := range tests {
		got := handler.backoff(tt.attempt)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("backoff(%d) = %v, want between %v and %v", tt.attempt, got, tt.wantMin, tt.wantMax)
		}
	}
}

// Test that multiple workers process events concurrently.
func TestEventHandler_ParallelWorkers(t *testing.T) {
	var processed int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerCount := 3
	eventCount := 6
	sleepPerEvent := 100 * time.Millisecond

	handler := New(
		ctx,
		workerCount,
		10,
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error {
			atomic.AddInt32(&processed, 1)
			time.Sleep(sleepPerEvent) // simulate sending event over network
			return nil
		},
	)

	start := time.Now()

	for i := 0; i < eventCount; i++ {
		handler.Push(Event{Payload: i, MaxRetries: 1})
	}

	for atomic.LoadInt32(&processed) == int32(eventCount) {
		time.Sleep(10 * time.Millisecond)
	}

	elapsed := time.Since(start)

	expectedSequential := time.Duration(eventCount) * sleepPerEvent
	expectedParallel := time.Duration(eventCount/workerCount) * sleepPerEvent

	if elapsed >= expectedSequential {
		t.Errorf("expected parallel execution, elapsed=%v, expected < %v", elapsed, expectedSequential)
	}

	t.Logf("Processed %d events with %d workers in %v (sequential=%v, ideal-parallel≈%v)",
		eventCount, workerCount, elapsed, expectedSequential, expectedParallel)
}
