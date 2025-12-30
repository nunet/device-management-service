package tokenomics

import (
	"time"
)

// BillingCycleTrigger calculates the next invoice time based on billing cycle
// and ensures we check frequently enough to catch period boundaries.
type BillingCycleTrigger struct {
	BillingCycle  time.Duration // Full billing cycle (e.g., 1 hour, 1 day)
	LastInvoiceAt time.Time     // Last time an invoice was generated
	CheckInterval time.Duration // Minimum check frequency (e.g., 30s)
	nextCheck     time.Time     // Next time to check for invoice
	startedAt     time.Time     // When trigger was initialized
}

// NewBillingCycleTrigger creates a new billing cycle trigger
func NewBillingCycleTrigger(
	billingCycle time.Duration,
	lastInvoiceAt time.Time,
	checkInterval time.Duration,
) *BillingCycleTrigger {
	now := time.Now().UTC()
	trigger := &BillingCycleTrigger{
		BillingCycle:  billingCycle,
		LastInvoiceAt: lastInvoiceAt,
		CheckInterval: checkInterval,
		startedAt:     now,
	}
	trigger.nextCheck = trigger.calculateNextCheck(now)
	return trigger
}

// IsReady checks if it's time to check for invoice generation
func (t *BillingCycleTrigger) IsReady(currentTime time.Time) bool {
	now := currentTime.UTC()
	return now.After(t.nextCheck) || now.Equal(t.nextCheck)
}

// MarkTriggered updates the trigger state after execution
func (t *BillingCycleTrigger) MarkTriggered(triggerTime time.Time) {
	t.startedAt = triggerTime.UTC()
	// Recalculate next check time
	t.nextCheck = t.calculateNextCheck(triggerTime.UTC())
}

// Reset resets the trigger to a new start time
func (t *BillingCycleTrigger) Reset(currentTime time.Time) {
	t.startedAt = currentTime.UTC()
	t.nextCheck = t.calculateNextCheck(currentTime.UTC())
}

// UpdateLastInvoiceAt updates the last invoice time (called after successful billing)
func (t *BillingCycleTrigger) UpdateLastInvoiceAt(invoiceTime time.Time) {
	t.LastInvoiceAt = invoiceTime.UTC()
	// Use invoiceTime (or current time if invoiceTime is in the future) to calculate next check
	now := time.Now().UTC()
	if invoiceTime.After(now) {
		// If invoice time is in the future (shouldn't happen), use current time
		t.nextCheck = t.calculateNextCheck(now)
	} else {
		// Use invoice time to calculate next check for accurate scheduling
		t.nextCheck = t.calculateNextCheck(invoiceTime.UTC())
	}
}

// calculateNextCheck determines when to check next for invoice generation
func (t *BillingCycleTrigger) calculateNextCheck(now time.Time) time.Time {
	if t.LastInvoiceAt.IsZero() {
		// First invoice - check after one billing cycle from start
		nextInvoice := t.startedAt.Add(t.BillingCycle)
		// But ensure we check at least every CheckInterval
		minCheck := now.Add(t.CheckInterval)
		if nextInvoice.Before(minCheck) {
			return minCheck
		}
		return nextInvoice
	}

	// Calculate next period boundary
	elapsed := now.Sub(t.LastInvoiceAt)
	periodsElapsed := elapsed / t.BillingCycle
	nextPeriodBoundary := t.LastInvoiceAt.Add((periodsElapsed + 1) * t.BillingCycle)

	// Ensure we check at least every CheckInterval
	minNextCheck := now.Add(t.CheckInterval)
	if nextPeriodBoundary.Before(minNextCheck) {
		return minNextCheck
	}

	return nextPeriodBoundary
}
