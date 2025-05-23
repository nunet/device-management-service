// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicTrigger_Interval(t *testing.T) {
	t.Parallel()

	t0 := time.Now()
	interval := 100 * time.Millisecond
	tr := &PeriodicTrigger{Interval: interval}
	tr.Reset(t0)

	// Should not be ready immediately
	require.False(t, tr.IsReady(t0), "Expected not ready immediately after creation")

	// Should be ready after interval
	t1 := t0.Add(interval + 1*time.Millisecond)
	assert.True(t, tr.IsReady(t1), "Expected ready after interval elapsed")
	// Should still be ready until marked triggered
	assert.True(t, tr.IsReady(t1.Add(interval/2)), "Expected ready until marked triggered")

	// MarkTriggered should reset startedAt
	tr.MarkTriggered(t1)
	require.Equalf(t, t1.UTC(), tr.startedAt, "MarkTriggered did not set startedAt correctly; got %v want %v", tr.startedAt, t1.UTC())

	// After MarkTriggered, should not be ready until interval passes again
	require.False(t, tr.IsReady(t1.Add(interval/2)), "Expected not ready before next interval")
	assert.True(t, tr.IsReady(t1.Add(interval+1*time.Millisecond)), "Expected ready after next interval")

	// Reset should set startedAt
	tr.Reset(t0)
	require.Equal(t, t0.UTC(), tr.startedAt, "Reset did not set startedAt")
}

func TestPeriodicTrigger_Cron(t *testing.T) {
	t.Parallel()

	cronExpr := "@every 1s"
	t0 := time.Now()
	tr := &PeriodicTrigger{CronExpr: cronExpr}
	tr.Reset(t0)

	// Should not be ready immediately
	require.False(t, tr.IsReady(t0), "Expected not ready immediately after creation")

	// Should be ready after 1s
	t1 := t0.Add(1100 * time.Millisecond)
	assert.True(t, tr.IsReady(t1), "Expected ready after cron interval elapsed")

	// MarkTriggered should update startedAt
	tr.MarkTriggered(t1)
	assert.Equal(t, t1.UTC(), tr.startedAt, "MarkTriggered should update startedAt")
}

func TestPeriodicTriggerWithJitter(t *testing.T) {
	t.Parallel()

	t0 := time.Now()
	interval := 100 * time.Millisecond
	jitter := 50 * time.Millisecond
	tr := &PeriodicTrigger{
		Interval: interval,
		Jitter:   func() time.Duration { return jitter },
	}
	tr.Reset(t0)

	// Should not be ready before interval + jitter
	assert.False(t, tr.IsReady(t0.Add(interval+jitter-1*time.Millisecond)), "Expected not ready before interval+jitter elapsed")
	// Should be ready after interval + jitter
	assert.True(t, tr.IsReady(t0.Add(interval+jitter+1*time.Millisecond)), "Expected ready after interval+jitter elapsed")
}

func TestEventTrigger(t *testing.T) {
	t.Parallel()

	ch := make(chan bool, 1)
	tr := &EventTrigger{Trigger: ch}

	// Should not be ready when channel is empty
	assert.False(t, tr.IsReady(time.Now()), "Expected not ready when channel is empty")

	// Send event
	ch <- true
	assert.True(t, tr.IsReady(time.Now()), "Expected ready when channel has event")

	// After reading, should not be ready again
	if tr.IsReady(time.Now()) {
		t.Error("Expected not ready after event consumed")
	}

	// Reset and MarkTriggered should be no-ops
	tr.Reset(time.Time{})
	tr.MarkTriggered(time.Now())
}

func TestOneTimeTrigger(t *testing.T) {
	t.Parallel()

	delay := 50 * time.Millisecond
	t0 := time.Now()
	tr := &OneTimeTrigger{After: delay}
	tr.Reset(t0)

	// Should not be ready immediately
	assert.False(t, tr.IsReady(t0), "Expected not ready immediately after creation")

	// Should be ready after delay
	t1 := t0.Add(delay + 1*time.Millisecond)
	assert.True(t, tr.IsReady(t1), "Expected ready after delay elapsed")

	// Should not be ready after MarkTriggered
	tr.MarkTriggered(t1)
	assert.False(t, tr.IsReady(t1.Add(1*time.Millisecond)), "Expected not ready after MarkTriggered")

	// Reset should allow trigger to be ready again after delay
	tr.Reset(t1)
	t2 := t1.Add(delay + 1*time.Millisecond)
	assert.False(t, tr.IsReady(t1), "Expected not ready immediately after Reset")
	tr.startedAt = t2.Add(-delay - 1*time.Millisecond) // simulate time passage
	assert.True(t, tr.IsReady(t2), "Expected ready after delay elapsed post-Reset")
}
