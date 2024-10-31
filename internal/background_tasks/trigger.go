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
	IsReady() bool // Returns true if the trigger condition is met.
	Reset()        // Resets the trigger state.
}

// PeriodicTrigger triggers at regular intervals or based on a cron expression.
type PeriodicTrigger struct {
	Interval      time.Duration // Interval for periodic triggering.
	CronExpr      string        // Cron expression for triggering.
	lastTriggered time.Time     // Last time the trigger was activated.
}

// IsReady checks if the trigger should activate based on time or cron expression.
func (t *PeriodicTrigger) IsReady() bool {
	// Trigger based on interval.
	if t.lastTriggered.Add(t.Interval).Before(time.Now()) {
		return true
	}

	// Trigger based on cron expression.
	if t.CronExpr != "" {
		cronExpr, err := cron.ParseStandard(t.CronExpr)
		if err != nil {
			return false
		}

		nextCronTriggerTime := cronExpr.Next(t.lastTriggered)
		return nextCronTriggerTime.Before(time.Now())
	}
	return false
}

// Reset updates the last triggered time to the current time.
func (t *PeriodicTrigger) Reset() {
	t.lastTriggered = time.Now()
}

// PeriodicTrigger triggers at regular intervals or based on a cron expression.
type PeriodicTriggerWithJitter struct {
	Interval      time.Duration // Interval for periodic triggering.
	CronExpr      string        // Cron expression for triggering.
	lastTriggered time.Time     // Last time the trigger was activated.
	Jitter        func() time.Duration
}

// IsReady checks if the trigger should activate based on time or cron expression.
func (t *PeriodicTriggerWithJitter) IsReady() bool {
	// Trigger based on interval.
	if t.lastTriggered.Add(t.Interval + t.Jitter()).Before(time.Now()) {
		return true
	}

	// Trigger based on cron expression.
	if t.CronExpr != "" {
		cronExpr, err := cron.ParseStandard(t.CronExpr)
		if err != nil {
			return false
		}

		nextCronTriggerTime := cronExpr.Next(t.lastTriggered)
		return nextCronTriggerTime.Before(time.Now())
	}
	return false
}

// Reset updates the last triggered time to the current time.
func (t *PeriodicTriggerWithJitter) Reset() {
	t.lastTriggered = time.Now()
}

// EventTrigger triggers based on an external event signaled through a channel.
type EventTrigger struct {
	Trigger chan bool // Channel to signal an event.
}

// IsReady checks if there is a signal in the trigger channel.
func (t *EventTrigger) IsReady() bool {
	select {
	case <-t.Trigger:
		return true
	default:
		return false
	}
}

// Reset for EventTrigger does nothing as its state is managed externally.
func (t *EventTrigger) Reset() {}

// OneTimeTrigger triggers once after a specified delay.
type OneTimeTrigger struct {
	Delay        time.Duration // The delay after which to trigger.
	registeredAt time.Time     // Time when the trigger was set.
}

// Reset sets the trigger registration time to the current time.
func (t *OneTimeTrigger) Reset() {
	t.registeredAt = time.Now()
}

// IsReady checks if the current time has passed the delay period.
func (t *OneTimeTrigger) IsReady() bool {
	return t.registeredAt.Add(t.Delay).Before(time.Now())
}
