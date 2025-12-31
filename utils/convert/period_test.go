// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package convert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func TestParsePaymentPeriod(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "minute",
			period:  contracts.PaymentPeriodMinute,
			want:    time.Minute,
			wantErr: false,
		},
		{
			name:    "hour",
			period:  contracts.PaymentPeriodHour,
			want:    time.Hour,
			wantErr: false,
		},
		{
			name:    "day",
			period:  contracts.PaymentPeriodDay,
			want:    24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "week",
			period:  contracts.PaymentPeriodWeek,
			want:    7 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "month",
			period:  contracts.PaymentPeriodMonth,
			want:    30 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "invalid period",
			period:  "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePaymentPeriod(tt.period)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCalculateElapsedPeriods(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		lastInvoiceAt time.Time
		now           time.Time
		periodDur     time.Duration
		periodCount   int
		wantCycles    int
		wantStart     time.Time
		wantEnd       time.Time
	}{
		{
			name:          "one hour elapsed, period count 1",
			lastInvoiceAt: baseTime,
			now:           baseTime.Add(1 * time.Hour),
			periodDur:     time.Hour,
			periodCount:   1,
			wantCycles:    1,
			wantStart:     baseTime,
			wantEnd:       baseTime.Add(1 * time.Hour),
		},
		{
			name:          "two hours elapsed, period count 1",
			lastInvoiceAt: baseTime,
			now:           baseTime.Add(2 * time.Hour),
			periodDur:     time.Hour,
			periodCount:   1,
			wantCycles:    2,
			wantStart:     baseTime,
			wantEnd:       baseTime.Add(2 * time.Hour),
		},
		{
			name:          "two hours elapsed, period count 2",
			lastInvoiceAt: baseTime,
			now:           baseTime.Add(2 * time.Hour),
			periodDur:     time.Hour,
			periodCount:   2,
			wantCycles:    1,
			wantStart:     baseTime,
			wantEnd:       baseTime.Add(2 * time.Hour),
		},
		{
			name:          "not enough time elapsed",
			lastInvoiceAt: baseTime,
			now:           baseTime.Add(30 * time.Minute),
			periodDur:     time.Hour,
			periodCount:   1,
			wantCycles:    0,
			wantStart:     time.Time{},
			wantEnd:       time.Time{},
		},
		{
			name:          "zero period count defaults to 1",
			lastInvoiceAt: baseTime,
			now:           baseTime.Add(1 * time.Hour),
			periodDur:     time.Hour,
			periodCount:   0,
			wantCycles:    1,
			wantStart:     baseTime,
			wantEnd:       baseTime.Add(1 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycles, start, end := CalculateElapsedPeriods(
				tt.lastInvoiceAt,
				tt.now,
				tt.periodDur,
				tt.periodCount,
			)
			require.Equal(t, tt.wantCycles, cycles)
			if tt.wantCycles > 0 {
				require.Equal(t, tt.wantStart, start)
				require.Equal(t, tt.wantEnd, end)
			} else {
				require.True(t, start.IsZero())
				require.True(t, end.IsZero())
			}
		})
	}
}
