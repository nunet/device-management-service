// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package convert

import (
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

// ParsePaymentPeriod converts a payment period string to duration
func ParsePaymentPeriod(period string) (time.Duration, error) {
	switch period {
	case contracts.PaymentPeriodMinute:
		return time.Minute, nil
	case contracts.PaymentPeriodHour:
		return time.Hour, nil
	case contracts.PaymentPeriodDay:
		return 24 * time.Hour, nil
	case contracts.PaymentPeriodWeek:
		return 7 * 24 * time.Hour, nil
	case contracts.PaymentPeriodMonth:
		return 30 * 24 * time.Hour, nil // Approximate
	default:
		return 0, fmt.Errorf("unsupported payment period: %s", period)
	}
}

// CalculateElapsedPeriods calculates elapsed billing periods
func CalculateElapsedPeriods(
	lastInvoiceAt time.Time,
	now time.Time,
	periodDuration time.Duration,
	periodCount int,
) (billingCyclesElapsed int, periodStart, periodEnd time.Time) {
	if periodCount <= 0 {
		periodCount = 1
	}

	elapsed := now.Sub(lastInvoiceAt)
	periodsElapsed := int(elapsed / periodDuration)
	billingCyclesElapsed = periodsElapsed / periodCount

	if billingCyclesElapsed < 1 {
		return 0, time.Time{}, time.Time{}
	}

	periodStart = lastInvoiceAt.Truncate(periodDuration)
	if periodStart.Before(lastInvoiceAt) {
		periodStart = periodStart.Add(periodDuration)
	}

	periodsToInvoice := billingCyclesElapsed * periodCount
	periodEnd = periodStart.Add(periodDuration * time.Duration(periodsToInvoice))

	return billingCyclesElapsed, periodStart, periodEnd
}
