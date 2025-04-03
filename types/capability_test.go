// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHardwareCapability_Comparable(t *testing.T) {
	t.Parallel()

	t.Run("JobType Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			l    JobType
			r    JobType
			want Comparison
		}{
			{
				name: "Error",
				l:    Batch,
				r:    SingleRun,
				want: None,
			},
			{
				name: "Equal",
				l:    LongRunning,
				r:    LongRunning,
				want: Equal,
			},
			{
				name: "Equal",
				l:    Batch,
				r:    Batch,
				want: Equal,
			},
			{
				name: "None",
				l:    Batch,
				r:    LongRunning,
				want: None,
			},
			{
				name: "None",
				l:    LongRunning,
				r:    Batch,
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("JobTypes checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			l    JobTypes
			r    JobTypes
			want Comparison
		}{
			{
				// currently the comparator is implemented in a way that it will return None
				// if compared job types contain slices that are not equal
				// we may want to change it to return Worse which is logically more correct
				name: "Different job type slices",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Batch, LongRunning},
				want: None,
			},
			{
				// currently the comparator is implemented in a way that it will return None
				// if compared job types contain slices that are not equal
				// we may want to change it to return Worse which is logically more correct
				name: "Different job types slices",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Recurring, SingleRun},
				want: None,
			},
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Batch},
				want: Better,
			},
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, LongRunning, SingleRun},
				r:    JobTypes{Batch, SingleRun},
				want: Better,
			},
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, LongRunning, SingleRun},
				r:    JobTypes{Batch, LongRunning},
				want: Better,
			},
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, LongRunning, SingleRun},
				r:    JobTypes{Recurring, SingleRun},
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Libraries checks", func(t *testing.T) {
		t.Parallel()
		library1 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library2 := Library{Name: "pytorch", Version: "1.8.1", Constraint: ">"}
		library3 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library4 := Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}
		library5 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library6 := Library{Name: "pytorch", Version: "1.7.1", Constraint: ">="}

		tests := []struct {
			name string
			l    Libraries
			r    Libraries
			want Comparison
		}{
			{
				name: "Better",
				l:    []Library{library1, library2},
				r:    []Library{library5, library6},
				want: Better,
			},
			{
				name: "Equal",
				l:    []Library{library1, library2},
				r:    []Library{library3, library4},
				want: Equal,
			},
			{
				name: "Equal",
				l:    []Library{library3, library4},
				r:    []Library{library1, library2},
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Locality checks", func(t *testing.T) {
		t.Parallel()

		locality1 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality2 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality3 := Locality{Kind: "GeoRegion", Name: "EU"}
		locality4 := Locality{Kind: "GeoRegion", Name: "US"}
		locality5 := Locality{Kind: "GeoRegion", Name: "EU"}

		tests := []struct {
			name string
			l    Locality
			r    Locality
			want Comparison
		}{
			{
				name: "Equal",
				l:    locality1,
				r:    locality2,
				want: Equal,
			},
			{
				name: "Worse",
				l:    locality3,
				r:    locality4,
				want: Worse,
			},
			{
				name: "Equal",
				l:    locality3,
				r:    locality5,
				want: Equal,
			},
			{
				name: "None",
				l:    locality1,
				r:    locality3,
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Localities checks", func(t *testing.T) {
		t.Parallel()
		var localities1 []Locality
		locality1 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality2 := Locality{Kind: "GeoRegion", Name: "EU"}
		localities1 = append(localities1, locality1)
		localities1 = append(localities1, locality2)

		var localities2 []Locality
		locality3 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality4 := Locality{Kind: "GeoRegion", Name: "US"}
		localities2 = append(localities2, locality3)
		localities2 = append(localities2, locality4)

		var localities3 []Locality
		locality5 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality6 := Locality{Kind: "GeoRegion", Name: "EU"}
		localities3 = append(localities3, locality5)
		localities3 = append(localities3, locality6)

		var localities4 []Locality
		locality7 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality8 := Locality{Kind: "GeoRegion", Name: "EU"}
		locality9 := Locality{Kind: "GeoCounty", Name: "Belgium"}

		localities4 = append(localities4, locality7)
		localities4 = append(localities4, locality8)
		localities4 = append(localities4, locality9)

		tests := []struct {
			name string
			l    Localities
			r    Localities
			want Comparison
		}{
			{
				name: "Equal",
				l:    localities4,
				r:    localities1,
				want: Equal,
			},
			{
				name: "Worse",
				l:    localities1,
				r:    localities2,
				want: Worse,
			},
			{
				name: "Equal",
				l:    localities1,
				r:    localities3,
				want: Equal,
			},
			{
				name: "Worse",
				l:    localities2,
				r:    localities3,
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Connectivity checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    Connectivity
			r    Connectivity
			want Comparison
		}{
			{
				name: "Better",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				r:    Connectivity{Ports: []int{3000}, VPN: false},
				want: Better,
			},
			{
				name: "Equal",
				l:    Connectivity{Ports: []int{3000}, VPN: false},
				r:    Connectivity{Ports: []int{3000}, VPN: false},
				want: Equal,
			},
			{
				name: "None",
				l:    Connectivity{Ports: []int{3000}, VPN: false},
				r:    Connectivity{Ports: []int{3000}, VPN: true},
				want: None,
			},
			{
				name: "None",
				l:    Connectivity{Ports: []int{3000, 2000}, VPN: false},
				r:    Connectivity{Ports: []int{3000}, VPN: true},
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.l.Compare(tt.r)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("PriceInformation checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host PriceInformation
			job  PriceInformation
			want Comparison
		}{
			{
				name: "Equal",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				want: Equal,
			},
			{
				name: "Better",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 80,
					TotalPerJob:     1000,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				want: Better,
			},
			{
				name: "Worse",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 80,
					TotalPerJob:     1000,
					Preference:      0,
				},
				want: Worse,
			},
			{
				name: "Worse",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 80,
					TotalPerJob:     800,
					Preference:      0,
				},
				want: Worse,
			},
			{
				name: "Worse",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     800,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 80,
					TotalPerJob:     1000,
					Preference:      0,
				},
				want: Worse,
			},
			{
				name: "Better",
				host: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 80,
					TotalPerJob:     800,
					Preference:      0,
				},
				job: PriceInformation{
					Currency:        "USD",
					CurrencyPerHour: 100,
					TotalPerJob:     1000,
					Preference:      0,
				},
				want: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("PricesInformation checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host PricesInformation
			job  PricesInformation
			want Comparison
		}{
			{
				name: "Equal",
				host: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				job: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: Equal,
			},
			{
				name: "Equal",
				host: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				job: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: Equal,
			},
			{
				name: "Better",
				host: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 80, TotalPerJob: 1000, Preference: 0},
				},
				job: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: Better,
			},
			{
				name: "Worse",
				host: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				job: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 80, TotalPerJob: 800, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: Worse,
			},
			{
				name: "None",
				host: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				job: []PriceInformation{
					{Currency: "MAD", CurrencyPerHour: 80, TotalPerJob: 800, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("TimeInformation checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host TimeInformation
			job  TimeInformation
			want Comparison
		}{
			{
				name: "Equal",
				host: TimeInformation{
					Units:      "seconds",
					MaxTime:    120,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "seconds",
					MaxTime:    120,
					Preference: 0,
				},
				want: Equal,
			},
			{
				name: "Equal",
				host: TimeInformation{
					Units:      "seconds",
					MaxTime:    120,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "minutes",
					MaxTime:    2,
					Preference: 0,
				},
				want: Equal,
			},
			{
				name: "Better",
				host: TimeInformation{
					Units:      "seconds",
					MaxTime:    180,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "minutes",
					MaxTime:    2,
					Preference: 0,
				},
				want: Better,
			},
			{
				name: "Better",
				host: TimeInformation{
					Units:      "minutes",
					MaxTime:    180,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "hours",
					MaxTime:    2,
					Preference: 0,
				},
				want: Better,
			},
			{
				name: "Better",
				host: TimeInformation{
					Units:      "minutes",
					MaxTime:    180,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "minutes",
					MaxTime:    2,
					Preference: 0,
				},
				want: Better,
			},
			{
				name: "Equal",
				host: TimeInformation{
					Units:      "minutes",
					MaxTime:    120,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "hours",
					MaxTime:    2,
					Preference: 0,
				},
				want: Equal,
			},
			{
				name: "Better",
				host: TimeInformation{
					Units:      "minutes",
					MaxTime:    180,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "hours",
					MaxTime:    2,
					Preference: 0,
				},
				want: Better,
			},
			{
				name: "Worse",
				host: TimeInformation{
					Units:      "minutes",
					MaxTime:    60,
					Preference: 0,
				},
				job: TimeInformation{
					Units:      "hours",
					MaxTime:    2,
					Preference: 0,
				},
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("KYC checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host KYC
			job  KYC
			want Comparison
		}{
			{
				name: "Equal",
				host: KYC{
					Type: "ke",
					Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233",
				},
				job: KYC{
					Type: "ke",
					Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233",
				},
				want: Equal,
			},
			{
				name: "None",
				host: KYC{
					Type: "ek",
					Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233",
				},
				job: KYC{
					Type: "ke",
					Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233",
				},
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("KYCs checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host KYCs
			job  KYCs
			want Comparison
		}{
			{
				name: "Equal",
				host: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				job: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: Equal,
			},
			{
				name: "None",
				host: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				job: []KYC{
					{Type: "ek", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: None,
			},
			{
				name: "Equal",
				host: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				job: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: Equal,
			},
			{
				name: "None",
				host: []KYC{
					{Type: "kde", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				job: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("HardwareCapability checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			host HardwareCapability
			job  HardwareCapability
			want Comparison
		}{
			{
				name: "JobTypes check",
				host: HardwareCapability{
					JobTypes: JobTypes{Batch, SingleRun},
				},
				job: HardwareCapability{
					JobTypes: JobTypes{Batch},
				},
				want: Better,
			},
			{
				name: "Library check",
				host: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.4.1", Constraint: "="},
						{Name: "pytorch", Version: "1.8.1", Constraint: ">"},
					},
				},
				job: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.4.1", Constraint: "="},
					},
				},
				want: Equal,
			},
			{
				name: "Locality check",
				host: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
					},
				},
				job: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
					},
				},
				want: Equal,
			},
			{
				name: "Connectivity check",
				host: HardwareCapability{
					Connectivity: Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				},
				job: HardwareCapability{
					Connectivity: Connectivity{Ports: []int{3000}, VPN: false},
				},
				want: Better,
			},
			{
				name: "PricesInformation check",
				host: HardwareCapability{
					Price: []PriceInformation{
						{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					},
				},
				job: HardwareCapability{
					Price: []PriceInformation{
						{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					},
				},
				want: Equal,
			},
			{
				name: "TimeInformation check",
				host: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    120,
						Preference: 0,
					},
				},
				job: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    120,
						Preference: 0,
					},
				},
				want: Equal,
			},
			{
				name: "KYCs check",
				host: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				job: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				want: Equal,
			},
			{
				name: "Executors checks",
				host: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker},
				},
				job: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker},
				},
				want: Equal,
			},
			{
				name: "Resources checks",
				host: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      2,
							ClockSpeed: 1000,
						},
						RAM:  RAM{Size: 1000},
						Disk: Disk{Size: 1000},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  1000,
							},
						},
					},
				},
				job: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM:  RAM{Size: 500},
						Disk: Disk{Size: 500},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  500,
							},
						},
					},
				},
				want: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.host.Compare(tt.job)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardwareCapability_Calculable_Add(t *testing.T) {
	t.Parallel()

	t.Run("JobTypes checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    JobTypes
			r    JobTypes
			want JobTypes
		}{
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Batch, LongRunning},
				want: JobTypes{Batch, SingleRun, LongRunning},
			},
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Recurring, SingleRun},
				want: JobTypes{Batch, SingleRun, Recurring},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("JobTypes.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("JobTypes.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Libraries checks", func(t *testing.T) {
		t.Parallel()

		library1 := Library{Name: "tensorflow", Version: "2.5.1", Constraint: "="}
		library2 := Library{Name: "pytorch", Version: "1.9.1", Constraint: ">"}
		library3 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library4 := Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}

		tests := []struct {
			name string
			l    Libraries
			r    Libraries
			want Libraries
		}{
			{
				name: "Different libraries",
				l:    []Library{library1, library2},
				r:    []Library{library3, library4},
				want: []Library{library1, library2, library3, library4},
			},
			{
				name: "Different libraries",
				l:    []Library{library1, library2},
				r:    []Library{library3},
				want: []Library{library1, library2, library3},
			},
			{
				name: "Same libraries",
				l:    []Library{library1, library2},
				r:    []Library{library1},
				want: []Library{library1, library2},
			},
			{
				name: "Same libraries",
				l:    []Library{library1, library2},
				r:    []Library{library2},
				want: []Library{library1, library2},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("Libraries.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Libraries.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Localities checks", func(t *testing.T) {
		t.Parallel()

		locality1 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality2 := Locality{Kind: "GeoRegion", Name: "EU"}
		locality3 := Locality{Kind: "NuNetRegion", Name: "us-east-1"}
		locality4 := Locality{Kind: "GeoRegion", Name: "US"}

		tests := []struct {
			name string
			l    Localities
			r    Localities
			want Localities
		}{
			{
				name: "Different localities",
				l:    Localities{locality1, locality2},
				r:    Localities{locality3, locality4},
				want: Localities{locality1, locality2, locality3, locality4},
			},
			{
				name: "Different localities",
				l:    Localities{locality1, locality2},
				r:    Localities{locality3},
				want: Localities{locality1, locality2, locality3},
			},
			{
				name: "Same localities",
				l:    Localities{locality1, locality2},
				r:    Localities{locality1},
				want: Localities{locality1, locality2},
			},
			{
				name: "Same localities",
				l:    Localities{locality1, locality2},
				r:    Localities{locality2},
				want: Localities{locality1, locality2},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("Localities.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Localities.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Connectivity checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    Connectivity
			r    Connectivity
			want Connectivity
		}{
			{
				name: "can add ports",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				r:    Connectivity{Ports: []int{1000}, VPN: false},
				want: Connectivity{Ports: []int{2000, 3000, 4000, 1000}, VPN: true},
			},
			{
				name: "can toggle VPN",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: false},
				r:    Connectivity{Ports: []int{1000}, VPN: true},
				want: Connectivity{Ports: []int{2000, 3000, 4000, 1000}, VPN: true},
			},
			{
				name: "does not duplicate ports",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				r:    Connectivity{Ports: []int{2000}, VPN: false},
				want: Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("Connectivity.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Connectivity.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Prices checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    PricesInformation
			r    PricesInformation
			want PricesInformation
		}{
			{
				name: "can add prices",
				l: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				r: []PriceInformation{
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
			},
			{
				name: "does not duplicate prices",
				l: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				r: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("PricesInformation.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("PricesInformation.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Time checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    TimeInformation
			r    TimeInformation
			want TimeInformation
		}{
			{
				name: "can add times",
				l: TimeInformation{
					Units:      "seconds",
					MaxTime:    120,
					Preference: 0,
				},
				r: TimeInformation{
					Units:      "seconds",
					MaxTime:    2,
					Preference: 0,
				},
				want: TimeInformation{
					Units:      "seconds",
					MaxTime:    122,
					Preference: 0,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("TimeInformation.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("TimeInformation.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("KYCs checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    KYCs
			r    KYCs
			want KYCs
		}{
			{
				name: "can add KYCs",
				l: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				r: []KYC{
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
			},
			{
				name: "does not duplicate KYCs",
				l: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				r: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("KYCs.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("KYCs.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("HardwareCapabilities checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    HardwareCapability
			r    HardwareCapability
			want HardwareCapability
		}{
			{
				name: "can add Executors",
				l: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker},
				},
				r: HardwareCapability{
					Executors: Executors{ExecutorTypeFirecracker},
				},
				want: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
				},
			},
			{
				name: "can add job types",
				l: HardwareCapability{
					JobTypes: JobTypes{Batch, SingleRun},
				},
				r: HardwareCapability{
					JobTypes: JobTypes{LongRunning},
				},
				want: HardwareCapability{
					JobTypes: JobTypes{Batch, SingleRun, LongRunning},
				},
			},
			{
				name: "can add resources",
				l: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 1000,
						},
						Disk: Disk{
							Size: 1000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  1000,
							},
						},
					},
				},
				r: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 1000,
						},
						Disk: Disk{
							Size: 1000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  1000,
							},
						},
					},
				},
				want: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      2,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 2000,
						},
						Disk: Disk{
							Size: 2000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  2000,
							},
						},
					},
				},
			},
			{
				name: "can add libraries",
				l: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.5.1", Constraint: "="},
					},
				},
				r: HardwareCapability{
					Libraries: []Library{
						{Name: "pytorch", Version: "1.9.1", Constraint: ">"},
					},
				},
				want: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.5.1", Constraint: "="},
						{Name: "pytorch", Version: "1.9.1", Constraint: ">"},
					},
				},
			},
			{
				name: "can add localities",
				l: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
					},
				},
				r: HardwareCapability{
					Localities: Localities{
						{Kind: "GeoRegion", Name: "EU"},
					},
				},
				want: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
						{Kind: "GeoRegion", Name: "EU"},
					},
				},
			},
			{
				name: "can add connectivity",
				l: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{2000, 3000, 4000},
						VPN:   false,
					},
				},
				r: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{1000},
						VPN:   true,
					},
				},
				want: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{2000, 3000, 4000, 1000},
						VPN:   true,
					},
				},
			},
			{
				name: "can add prices",
				l: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "USD",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
				r: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "EUR",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
				want: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "USD",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
						{
							Currency:        "EUR",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
			},
			{
				name: "can add time",
				l: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    120,
						Preference: 0,
					},
				},
				r: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    2,
						Preference: 0,
					},
				},
				want: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    122,
						Preference: 0,
					},
				},
			},
			{
				name: "can add KYCs",
				l: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				r: HardwareCapability{
					KYCs: []KYC{
						{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				want: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
						{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("HardwareCapability.Add() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("HardwareCapability.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})
}

func TestHardwareCapability_Calculable_Subtract(t *testing.T) {
	t.Parallel()

	t.Run("JobTypes checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    JobTypes
			r    JobTypes
			want JobTypes
		}{
			{
				name: "Different job type slices",
				l:    JobTypes{Batch, SingleRun, LongRunning},
				r:    JobTypes{Batch, LongRunning},
				want: JobTypes{SingleRun},
			},
			{
				name: "a job type is not present in l",
				l:    JobTypes{Batch, SingleRun, LongRunning},
				r:    JobTypes{Recurring, SingleRun},
				want: JobTypes{Batch, LongRunning},
			},
			{
				name: "can empty the slice",
				l:    JobTypes{Batch, SingleRun},
				r:    JobTypes{Batch, SingleRun},
				want: JobTypes{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("JobTypes.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("JobTypes.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Libraries checks", func(t *testing.T) {
		t.Parallel()

		library1 := Library{Name: "tensorflow", Version: "2.5.1", Constraint: "="}
		library2 := Library{Name: "pytorch", Version: "1.9.1", Constraint: ">"}
		library3 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library4 := Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}

		tests := []struct {
			name string
			l    Libraries
			r    Libraries
			want Libraries
		}{
			{
				name: "can remove libraries",
				l:    []Library{library1, library2, library3, library4},
				r:    []Library{library3, library4},
				want: []Library{library1, library2},
			},
			{
				name: "a library is not present in l",
				l:    []Library{library1, library2, library4},
				r:    []Library{library3},
				want: []Library{library1, library2, library4},
			},
			{
				name: "can empty the slice",
				l:    []Library{library1, library2},
				r:    []Library{library1, library2},
				want: []Library{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("Libraries.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Libraries.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Localities checks", func(t *testing.T) {
		t.Parallel()

		locality1 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality2 := Locality{Kind: "GeoRegion", Name: "EU"}
		locality3 := Locality{Kind: "NuNetRegion", Name: "us-east-1"}
		locality4 := Locality{Kind: "GeoRegion", Name: "US"}

		tests := []struct {
			name string
			l    Localities
			r    Localities
			want Localities
		}{
			{
				name: "can remove localities",
				l:    Localities{locality1, locality2, locality3, locality4},
				r:    Localities{locality3, locality4},
				want: Localities{locality1, locality2},
			},
			{
				name: "a locality is not present in l",
				l:    Localities{locality1, locality2, locality4},
				r:    Localities{locality3},
				want: Localities{locality1, locality2, locality4},
			},
			{
				name: "can empty the slice",
				l:    Localities{locality1, locality2},
				r:    Localities{locality1, locality2},
				want: Localities{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("Localities.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Localities.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Connectivity checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    Connectivity
			r    Connectivity
			want Connectivity
		}{
			{
				name: "can remove ports",
				l:    Connectivity{Ports: []int{2000, 3000, 4000, 1000}, VPN: true},
				r:    Connectivity{Ports: []int{1000}, VPN: false},
				want: Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
			},
			{
				name: "can toggle VPN",
				l:    Connectivity{Ports: []int{2000, 3000, 4000, 1000}, VPN: true},
				r:    Connectivity{Ports: []int{1000}, VPN: true},
				want: Connectivity{Ports: []int{2000, 3000, 4000}, VPN: false},
			},
			{
				name: "does not duplicate ports",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				r:    Connectivity{Ports: []int{1000}, VPN: false},
				want: Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
			},
			{
				name: "can empty the slice",
				l:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				r:    Connectivity{Ports: []int{2000, 3000, 4000}, VPN: true},
				want: Connectivity{Ports: nil, VPN: false},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("Connectivity.Subtract() error = %v", err)
				}

				// compare arrays irrespective of order
				sort.Ints(tt.l.Ports)
				sort.Ints(tt.want.Ports)
				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Connectivity.Subtract() = %v, want %v", tt.l, tt.want)
				}

				if tt.l.VPN != tt.want.VPN {
					t.Errorf("Connectivity.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Prices checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    PricesInformation
			r    PricesInformation
			want PricesInformation
		}{
			{
				name: "can remove prices",
				l: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				r: []PriceInformation{
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
			},
			{
				name: "a price is not present in l",
				l: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				r: []PriceInformation{
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				want: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("PricesInformation.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("PricesInformation.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Time checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    TimeInformation
			r    TimeInformation
			want TimeInformation
		}{
			{
				name: "can subtract max time",
				l: TimeInformation{
					Units:      "seconds",
					MaxTime:    120,
					Preference: 0,
				},
				r: TimeInformation{
					Units:      "seconds",
					MaxTime:    2,
					Preference: 0,
				},
				want: TimeInformation{
					Units:      "seconds",
					MaxTime:    118,
					Preference: 0,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("TimeInformation.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("TimeInformation.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("KYCs checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    KYCs
			r    KYCs
			want KYCs
		}{
			{
				name: "can remove KYCs",
				l: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				r: []KYC{
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
			},
			{
				name: "a KYC is not present in l",
				l: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				r: []KYC{
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
			},
			{
				name: "can empty the slice",
				l: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				r: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				want: []KYC{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("KYCs.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("KYCs.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("HardwareCapabilities checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    HardwareCapability
			r    HardwareCapability
			want HardwareCapability
		}{
			{
				name: "can subtract Executors",
				l: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
				},
				r: HardwareCapability{
					Executors: Executors{ExecutorTypeDocker},
				},
				want: HardwareCapability{
					Executors: Executors{ExecutorTypeFirecracker},
				},
			},
			{
				name: "can subtract job types",
				l: HardwareCapability{
					JobTypes: JobTypes{Batch, SingleRun, LongRunning},
				},
				r: HardwareCapability{
					JobTypes: JobTypes{LongRunning},
				},
				want: HardwareCapability{
					JobTypes: JobTypes{Batch, SingleRun},
				},
			},
			{
				name: "can subtract resources",
				l: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      2,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 2000,
						},
						Disk: Disk{
							Size: 2000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  2000,
							},
						},
					},
				},
				r: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 1000,
						},
						Disk: Disk{
							Size: 1000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  1000,
							},
						},
					},
				},
				want: HardwareCapability{
					Resources: Resources{
						CPU: CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: RAM{
							Size: 1000,
						},
						Disk: Disk{
							Size: 1000,
						},
						GPUs: GPUs{
							{
								Index: 0,
								VRAM:  1000,
							},
						},
					},
				},
			},
			{
				name: "can subtract libraries",
				l: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.5.1", Constraint: "="},
						{Name: "pytorch", Version: "1.9.1", Constraint: ">"},
					},
				},
				r: HardwareCapability{
					Libraries: []Library{
						{Name: "pytorch", Version: "1.9.1", Constraint: ">"},
					},
				},
				want: HardwareCapability{
					Libraries: []Library{
						{Name: "tensorflow", Version: "2.5.1", Constraint: "="},
					},
				},
			},
			{
				name: "can subtract localities",
				l: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
						{Kind: "GeoRegion", Name: "EU"},
					},
				},
				r: HardwareCapability{
					Localities: Localities{
						{Kind: "GeoRegion", Name: "EU"},
					},
				},
				want: HardwareCapability{
					Localities: Localities{
						{Kind: "NuNetRegion", Name: "us-west-1"},
					},
				},
			},
			{
				name: "can subtract connectivity",
				l: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{2000, 3000, 4000},
						VPN:   false,
					},
				},
				r: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{2000},
						VPN:   true,
					},
				},
				want: HardwareCapability{
					Connectivity: Connectivity{
						Ports: []int{3000, 4000},
						VPN:   false,
					},
				},
			},
			{
				name: "can subtract prices",
				l: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "USD",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
						{
							Currency:        "EUR",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
				r: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "EUR",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
				want: HardwareCapability{
					Price: PricesInformation{
						{
							Currency:        "USD",
							CurrencyPerHour: 100,
							TotalPerJob:     1000,
						},
					},
				},
			},
			{
				name: "can subtract time",
				l: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    122,
						Preference: 0,
					},
				},
				r: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    2,
						Preference: 0,
					},
				},
				want: HardwareCapability{
					Time: TimeInformation{
						Units:      "seconds",
						MaxTime:    120,
						Preference: 0,
					},
				},
			},
			{
				name: "can subtract KYCs",
				l: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
						{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				r: HardwareCapability{
					KYCs: []KYC{
						{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
				want: HardwareCapability{
					KYCs: []KYC{
						{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("HardwareCapability.Subtract() error = %v", err)
				}

				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("HardwareCapability.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})
}

func TestHardwareCapability_Contains(t *testing.T) {
	t.Parallel()

	t.Run("JobTypes checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			slice JobTypes
			elem  JobType
			want  bool
		}{
			{
				name:  "JobType is present",
				slice: JobTypes{Batch, SingleRun, LongRunning},
				elem:  Batch,
				want:  true,
			},
			{
				name:  "JobType is not present",
				slice: JobTypes{Batch, SingleRun, LongRunning},
				elem:  Recurring,
				want:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.slice.Contains(tt.elem); got != tt.want {
					t.Errorf("JobTypes.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Libraries checks", func(t *testing.T) {
		t.Parallel()

		library1 := Library{Name: "tensorflow", Version: "2.5.1", Constraint: "="}
		library2 := Library{Name: "pytorch", Version: "1.9.1", Constraint: ">"}
		library3 := Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
		library4 := Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}

		tests := []struct {
			name  string
			slice Libraries
			elem  Library
			want  bool
		}{
			{
				name:  "Library is present",
				slice: []Library{library1, library2, library3, library4},
				elem:  library1,
				want:  true,
			},
			{
				name:  "Library is not present",
				slice: []Library{library1, library2, library4},
				elem:  library3,
				want:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.slice.Contains(tt.elem); got != tt.want {
					t.Errorf("Libraries.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("Localities checks", func(t *testing.T) {
		t.Parallel()

		locality1 := Locality{Kind: "NuNetRegion", Name: "us-west-1"}
		locality2 := Locality{Kind: "GeoRegion", Name: "EU"}
		locality3 := Locality{Kind: "NuNetRegion", Name: "us-east-1"}
		locality4 := Locality{Kind: "GeoRegion", Name: "US"}

		tests := []struct {
			name  string
			slice Localities
			elem  Locality
			want  bool
		}{
			{
				name:  "Locality is present",
				slice: Localities{locality1, locality2, locality3, locality4},
				elem:  locality1,
				want:  true,
			},
			{
				name:  "Locality is not present",
				slice: Localities{locality1, locality2, locality4},
				elem:  locality3,
				want:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.slice.Contains(tt.elem); got != tt.want {
					t.Errorf("Localities.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("PricesInformation checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			slice PricesInformation
			elem  PriceInformation
			want  bool
		}{
			{
				name: "Price is present",
				slice: []PriceInformation{
					{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				elem: PriceInformation{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				want: true,
			},
			{
				name: "Price is not present",
				slice: []PriceInformation{
					{Currency: "EUR", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				},
				elem: PriceInformation{Currency: "USD", CurrencyPerHour: 100, TotalPerJob: 1000, Preference: 0},
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.slice.Contains(tt.elem); got != tt.want {
					t.Errorf("PricesInformation.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("KYCs checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			slice KYCs
			elem  KYC
			want  bool
		}{
			{
				name: "KYC is present",
				slice: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				elem: KYC{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				want: true,
			},
			{
				name: "KYC is not present",
				slice: []KYC{
					{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
					{Type: "ed", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				},
				elem: KYC{Type: "ke", Data: "as09di42ktmf90scdvms84tgmspd0idfm31;r509jvmx.ods-94m233"},
				want: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.slice.Contains(tt.elem); got != tt.want {
					t.Errorf("KYCs.Contains() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
