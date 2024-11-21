// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResources_Calculable_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		r       *Resources
		other   Resources
		wantErr bool
		want    Resources
	}{
		{
			name: "Add",
			r: &Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  8192,
					},
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  8192,
					},
				},
			},
			wantErr: false,
			want: Resources{
				CPU: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 2048,
				},
				Disk: Disk{
					Size: 2048,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  16384,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.r.Add(tt.other); (err != nil) != tt.wantErr {
				t.Errorf("Resources.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertResources(t, *tt.r, tt.want)
		})
	}
}

func TestResources_Calculable_Subtract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		r       Resources
		other   Resources
		wantErr bool
		want    Resources
		err     string
	}{
		{
			name: "Subtract",
			r: Resources{
				CPU: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 2048,
				},
				Disk: Disk{
					Size: 2048,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  16384,
					},
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  8192,
					},
				},
			},
			wantErr: false,
			want: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  8192,
					},
				},
			},
		},
		{
			name: "cpu underflow scenario",
			r: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 2048,
				},
				Disk: Disk{
					Size: 2048,
				},
			},
			wantErr: true,
			err:     "error subtracting CPU",
		},
		{
			name: "ram underflow scenario",
			r: Resources{
				RAM: RAM{
					Size: 1024,
				},
			},
			other: Resources{
				RAM: RAM{
					Size: 2048,
				},
			},
			wantErr: true,
			err:     "error subtracting RAM",
		},
		{
			name: "disk underflow scenario",
			r: Resources{
				Disk: Disk{
					Size: 1024,
				},
			},
			other: Resources{
				Disk: Disk{
					Size: 2048,
				},
			},
			wantErr: true,
			err:     "error subtracting Disk",
		},
		{
			name: "gpu underflow scenario",
			r: Resources{
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  8192,
					},
				},
			},
			other: Resources{
				GPUs: GPUs{
					{
						Model: "GTX 1080",
						VRAM:  16384,
					},
				},
			},
			wantErr: true,
			err:     "error subtracting GPUs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.r.Subtract(tt.other)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resources.Subtract() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				require.ErrorContains(t, err, tt.err)
			}

			if !tt.wantErr {
				assertResources(t, tt.r, tt.want)
			}
		})
	}
}

func TestResources_Comparable_Compare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		r     *Resources
		other Resources
		want  Comparison
	}{
		{
			name:  "Default state",
			r:     &Resources{},
			other: Resources{},
			want:  Equal,
		},
		{
			name: "Equal resources",
			r: &Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       8192,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       8192,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			want: Equal,
		},
		{
			name: "Better resources",
			r: &Resources{
				CPU: CPU{
					Cores:      2,
					ClockSpeed: 2000,
				},
				RAM: RAM{
					Size: 2048,
				},
				Disk: Disk{
					Size: 2048,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       16384,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       8192,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			want: Better,
		},
		{
			name: "Worse resources",
			r: &Resources{
				CPU: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				RAM: RAM{
					Size: 1024,
				},
				Disk: Disk{
					Size: 1024,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       8192,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			other: Resources{
				CPU: CPU{
					Cores:      2,
					ClockSpeed: 2000,
				},
				RAM: RAM{
					Size: 2048,
				},
				Disk: Disk{
					Size: 2048,
				},
				GPUs: GPUs{
					{
						Model:      "GTX 1080",
						VRAM:       16384,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "0000:00:00.0",
						Index:      0,
					},
				},
			},
			want: Worse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.r.Compare(tt.other)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestResources_Equal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		r     Resources
		other Resources
		want  bool
	}{
		{
			name:  "Equal CPUs",
			r:     Resources{CPU: CPU{Cores: 1, ClockSpeed: 1000}},
			other: Resources{CPU: CPU{Cores: 1, ClockSpeed: 1000}},
			want:  true,
		},
		{
			name:  "Different CPUs",
			r:     Resources{CPU: CPU{Cores: 1, ClockSpeed: 1000}},
			other: Resources{CPU: CPU{Cores: 2, ClockSpeed: 1000}},
			want:  false,
		},
		{
			name:  "Equal RAM",
			r:     Resources{RAM: RAM{Size: 1024}},
			other: Resources{RAM: RAM{Size: 1024}},
			want:  true,
		},
		{
			name:  "Different RAM",
			r:     Resources{RAM: RAM{Size: 1024}},
			other: Resources{RAM: RAM{Size: 2048}},
			want:  false,
		},
		{
			name:  "Equal Disk",
			r:     Resources{Disk: Disk{Size: 1024}},
			other: Resources{Disk: Disk{Size: 1024}},
			want:  true,
		},
		{
			name:  "Different Disk",
			r:     Resources{Disk: Disk{Size: 1024}},
			other: Resources{Disk: Disk{Size: 2048}},
			want:  false,
		},
		{
			name:  "Equal GPUs",
			r:     Resources{GPUs: GPUs{{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"}}},
			other: Resources{GPUs: GPUs{{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"}}},
			want:  true,
		},
		{
			name:  "Different GPUs",
			r:     Resources{GPUs: GPUs{{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"}}},
			other: Resources{GPUs: GPUs{{Model: "An AMD GPU", VRAM: 8192, Index: 1, Vendor: GPUVendorAMDATI, PCIAddress: "0000:00:00.1"}}},
			want:  false,
		},
		{
			name: "Different number of GPUs",
			r: Resources{
				GPUs: GPUs{
					{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"},
					{Model: "GTX 1080", VRAM: 8192, Index: 1, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.1"},
				},
			},
			other: Resources{
				GPUs: GPUs{
					{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"},
				},
			},
		},
		{
			name: "Equal resources",
			r: Resources{
				CPU:  CPU{Cores: 1, ClockSpeed: 1000},
				RAM:  RAM{Size: 1024},
				Disk: Disk{Size: 1024},
				GPUs: GPUs{
					{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"},
				},
			},
			other: Resources{
				CPU:  CPU{Cores: 1, ClockSpeed: 1000},
				RAM:  RAM{Size: 1024},
				Disk: Disk{Size: 1024},
				GPUs: GPUs{
					{Model: "GTX 1080", VRAM: 8192, Index: 0, Vendor: GPUVendorNvidia, PCIAddress: "0000:00:00.0"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.r.Equal(tt.other))
		})
	}
}

func assertResources(t *testing.T, r1, r2 Resources) {
	t.Helper()

	// CPU
	require.Equal(t, r1.CPU.Cores, r2.CPU.Cores)
	require.Equal(t, r1.CPU.ClockSpeed, r2.CPU.ClockSpeed)

	// RAM
	require.Equal(t, r1.RAM.Size, r2.RAM.Size)

	// Disk
	require.Equal(t, r1.Disk.Size, r2.Disk.Size)

	// GPUs
	require.Len(t, r1.GPUs, len(r2.GPUs))
	for i := range r1.GPUs {
		require.Equal(t, r1.GPUs[i].Model, r2.GPUs[i].Model)
		require.Equal(t, r1.GPUs[i].VRAM, r2.GPUs[i].VRAM)
	}
}
