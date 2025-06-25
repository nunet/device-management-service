// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package backgroundtasks

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Trigger interface defines a method to check if a trigger condition is met.
type Trigger interface {
	IsReady(currentTime time.Time) bool  // Returns true if the trigger condition is met.
	MarkTriggered(triggerTime time.Time) // Marks the trigger as triggered.
	Reset(currentTime time.Time)         // Resets the trigger state.
}

// PeriodicTrigger triggers at regular intervals or based on a cron expression.
type PeriodicTrigger struct {
	Interval  time.Duration // Interval for periodic triggering.
	CronExpr  string        // Cron expression for triggering.
	Jitter    func() time.Duration
	startedAt time.Time
}

// IsReady checks if the trigger should activate based on time or cron expression.
func (t *PeriodicTrigger) IsReady(currentTime time.Time) bool {
	var jitter time.Duration
	if t.Jitter != nil {
		jitter = t.Jitter()
	}

	if t.startedAt.IsZero() {
		t.startedAt = currentTime
	}

	now := currentTime.UTC()

	// Trigger based on cron expression.
	if t.CronExpr != "" {
		cronExpr, err := cron.ParseStandard(t.CronExpr)
		if err != nil {
			return false
		}

		nextCronTriggerTime := cronExpr.Next(t.startedAt)
		return !nextCronTriggerTime.IsZero() && nextCronTriggerTime.Add(jitter).Before(now)
	}

	// Trigger based on interval.
	if t.Interval > 0 {
		if t.startedAt.Add(t.Interval + jitter).Before(now) {
			return true
		}
	}

	return false
}

func (t *PeriodicTrigger) MarkTriggered(triggerTime time.Time) {
	t.startedAt = triggerTime.UTC()
}

// Reset updates the last triggered time to the current time.
func (t *PeriodicTrigger) Reset(currentTime time.Time) {
	t.startedAt = currentTime.UTC()
}

// EventTrigger triggers based on an external event signaled through a channel.
type EventTrigger struct {
	Trigger chan bool // Channel to signal an event.
}

// IsReady checks if there is a signal in the trigger channel.
func (t *EventTrigger) IsReady(_ time.Time) bool {
	select {
	case <-t.Trigger:
		return true
	default:
		return false
	}
}

// MarkTriggered for EventTrigger does nothing as its state is managed externally.
func (t *EventTrigger) MarkTriggered(_ time.Time) {}

// Reset for EventTrigger does nothing as its state is managed externally.
func (t *EventTrigger) Reset(_ time.Time) {}

// OneTimeTrigger triggers once after a specified delay.
type OneTimeTrigger struct {
	After     time.Duration // The delay after which to trigger.
	startedAt time.Time     // Time when the trigger was set.
	triggered bool          // Flag indicating if the trigger has been triggered.
}

// IsReady checks if the current time has passed the delay period.
func (t *OneTimeTrigger) IsReady(currentTime time.Time) bool {
	if t.startedAt.IsZero() {
		t.startedAt = currentTime
	}
	return !t.triggered && t.startedAt.Add(t.After).Before(currentTime)
}

func (t *OneTimeTrigger) MarkTriggered(_ time.Time) {
	t.triggered = true
}

// Reset sets the trigger registration time to the current time.
func (t *OneTimeTrigger) Reset(currentTime time.Time) {
	t.startedAt = currentTime.UTC()
	t.triggered = false
}
