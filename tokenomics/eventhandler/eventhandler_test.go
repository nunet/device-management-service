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

	handler := New(
		context.Background(),
		1,  // workers
		10, // queue size
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error {
			atomic.AddInt32(&processed, 1)
			return nil
		},
	)
	defer handler.Stop()

	handler.Push(Event{Payload: "ok", MaxRetries: 1})

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&processed) != 1 {
		t.Fatalf("expected event to be processed once, got %d", processed)
	}
}

func TestEventHandler_RetryUntilSuccess(t *testing.T) {
	var attempts int32

	handler := New(
		context.Background(),
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
	defer handler.Stop()

	handler.Push(Event{Payload: "retry", MaxRetries: 5})

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&attempts) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestEventHandler_MaxRetries(t *testing.T) {
	var attempts int32

	handler := New(
		context.Background(),
		1,
		10,
		5*time.Millisecond,
		50*time.Millisecond,
		func(_ Event) error {
			atomic.AddInt32(&attempts, 1)
			return errors.New("always fail")
		},
	)
	defer handler.Stop()

	handler.Push(Event{Payload: "fail", MaxRetries: 2})

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&attempts) != 3 { // 1 original + 2 retries
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestEventHandler_Backoff(t *testing.T) {
	handler := New(
		context.Background(),
		1,
		1,
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error { return nil },
	)
	defer handler.Stop()

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

func TestEventHandler_Stop(t *testing.T) {
	handler := New(
		context.Background(),
		2,
		10,
		10*time.Millisecond,
		100*time.Millisecond,
		func(_ Event) error { return nil },
	)

	done := make(chan struct{})
	go func() {
		handler.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop() did not complete")
	}
}

// Test that multiple workers process events concurrently.
func TestEventHandler_ParallelWorkers(t *testing.T) {
	var processed int32

	workerCount := 3
	eventCount := 6
	sleepPerEvent := 100 * time.Millisecond

	handler := New(
		context.Background(),
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
	defer handler.Stop()

	start := time.Now()

	for i := 0; i < eventCount; i++ {
		handler.Push(Event{Payload: i, MaxRetries: 1})
	}

	for {
		//nolint:staticcheck
		if atomic.LoadInt32(&processed) == int32(eventCount) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	elapsed := time.Since(start)

	// If sequential: eventCount * sleepPerEvent
	// If parallel: closer to (eventCount/workerCount) * sleepPerEvent
	expectedSequential := time.Duration(eventCount) * sleepPerEvent
	expectedParallel := time.Duration(eventCount/workerCount) * sleepPerEvent

	if elapsed >= expectedSequential {
		t.Errorf("expected parallel execution, elapsed=%v, expected < %v", elapsed, expectedSequential)
	}

	t.Logf("Processed %d events with %d workers in %v (sequential=%v, ideal-parallel≈%v)",
		eventCount, workerCount, elapsed, expectedSequential, expectedParallel)
}
