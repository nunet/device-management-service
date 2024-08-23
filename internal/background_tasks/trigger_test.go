package backgroundtasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodicTrigger(t *testing.T) {
	trigger := PeriodicTrigger{
		Interval: 1 * time.Second,
	}
	trigger.Reset()

	time.Sleep(2 * time.Second) // Wait for trigger interval to pass
	assert.True(t, trigger.IsReady(), "PeriodicTrigger should be ready after the interval")

	trigger.Reset()
	assert.False(t, trigger.IsReady(), "PeriodicTrigger should not be ready immediately after reset")

	trigger.CronExpr = "@every 1s"
	time.Sleep(2 * time.Second) // Wait for trigger interval to pass
	assert.True(t, trigger.IsReady(), "PeriodicTrigger with CronExpr should be ready after the interval")

	trigger.Reset()
	trigger.Interval = 20 * time.Minute
	trigger.CronExpr = "thewrongcron expression"
	time.Sleep(4 * time.Second) // Wait for trigger interval to pass
	assert.False(t, trigger.IsReady(), "PeriodicTrigger with wrong CronExpr should throw an error")
}

func TestEventTrigger(t *testing.T) {
	trigger := EventTrigger{Trigger: make(chan bool, 1)}
	assert.False(t, trigger.IsReady(), "EventTrigger should not be ready without an event")

	trigger.Trigger <- true
	assert.True(t, trigger.IsReady(), "EventTrigger should be ready after receiving an event")
}

func TestOneTimeTrigger(t *testing.T) {
	trigger := OneTimeTrigger{Delay: 1 * time.Second}
	trigger.Reset()

	time.Sleep(2 * time.Second) // Wait for delay to pass
	assert.True(t, trigger.IsReady(), "OneTimeTrigger should be ready after the delay")
}
