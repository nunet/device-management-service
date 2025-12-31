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
)

func TestDurationToUnit(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		unit     string
		want     float64
		wantErr  bool
	}{
		{
			name:     "seconds",
			duration: 120 * time.Second,
			unit:     "second",
			want:     120.0,
			wantErr:  false,
		},
		{
			name:     "minutes",
			duration: 2 * time.Minute,
			unit:     "minute",
			want:     2.0,
			wantErr:  false,
		},
		{
			name:     "hours",
			duration: 3 * time.Hour,
			unit:     "hour",
			want:     3.0,
			wantErr:  false,
		},
		{
			name:     "invalid unit",
			duration: time.Hour,
			unit:     "day",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DurationToUnit(tt.duration, tt.unit)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}

func TestSecondsToUnit(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		unit    string
		want    float64
		wantErr bool
	}{
		{
			name:    "seconds to seconds",
			seconds: 120.0,
			unit:    "second",
			want:    120.0,
			wantErr: false,
		},
		{
			name:    "seconds to minutes",
			seconds: 120.0,
			unit:    "minute",
			want:    2.0,
			wantErr: false,
		},
		{
			name:    "seconds to hours",
			seconds: 3600.0,
			unit:    "hour",
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "invalid unit",
			seconds: 3600.0,
			unit:    "day",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SecondsToUnit(tt.seconds, tt.unit)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}
