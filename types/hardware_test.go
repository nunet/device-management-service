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

func TestHardware_Comparable_Compare(t *testing.T) {
	t.Parallel()

	t.Run("CPU checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			c1   CPU
			c2   CPU
			want Comparison
		}{
			{
				name: "Equal Cores and ClockSpeed",
				c1: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				want: Equal,
			},
			{
				name: "Better Cores and ClockSpeed",
				c1: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				want: Better,
			},
			{
				name: "Worse Cores and ClockSpeed",
				c1: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.c1.Compare(tt.c2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("RAM checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			r1   RAM
			r2   RAM
			want Comparison
		}{
			{
				name: "Equal Size",
				r1: RAM{
					Size: 1024,
				},
				r2: RAM{
					Size: 1024,
				},
				want: Equal,
			},
			{
				name: "Better Size",
				r1: RAM{
					Size: 1024,
				},
				r2: RAM{
					Size: 512,
				},
				want: Better,
			},
			{
				name: "Worse Size",
				r1: RAM{
					Size: 512,
				},
				r2: RAM{
					Size: 1024,
				},
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.r1.Compare(tt.r2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Disk checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			d1   Disk
			d2   Disk
			want Comparison
		}{
			{
				name: "Equal Size",
				d1: Disk{
					Size: 1024,
				},
				d2: Disk{
					Size: 1024,
				},
				want: Equal,
			},
			{
				name: "Better Size",
				d1: Disk{
					Size: 1024,
				},
				d2: Disk{
					Size: 512,
				},
				want: Better,
			},
			{
				name: "Worse Size",
				d1: Disk{
					Size: 512,
				},
				d2: Disk{
					Size: 1024,
				},
				want: Worse,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.d1.Compare(tt.d2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GPUVendor checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			v1   GPUVendor
			v2   GPUVendor
			want Comparison
		}{
			{
				name: "Equal GPUVendor",
				v1:   GPUVendorNvidia,
				v2:   GPUVendorNvidia,
				want: Equal,
			},
			{
				name: "Unequal GPUVendor",
				v1:   GPUVendorNvidia,
				v2:   GPUVendorIntel,
				want: None,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.v1.Compare(tt.v2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GPU checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			g1   GPU
			g2   GPU
			want Comparison
		}{
			{
				name: "Equal VRAM",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      1,
					Vendor:     GPUVendorIntel,
					PCIAddress: "AAAA:BB:CC.D",
					Model:      "IntelA770",
					VRAM:       16384,
				},
				want: Equal,
			},
			{
				name: "Worse VRAM",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       8450,
				},
				g2: GPU{
					Index:      1,
					Vendor:     GPUVendorIntel,
					PCIAddress: "AAAA:BB:CC.D",
					Model:      "IntelA770",
					VRAM:       16384,
				},
				want: Worse,
			},
			{
				name: "Better VRAM",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      1,
					Vendor:     GPUVendorIntel,
					PCIAddress: "AAAA:BB:CC.D",
					Model:      "IntelA770",
					VRAM:       8192,
				},
				want: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.g1.Compare(tt.g2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GPUs checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			g1   GPUs
			g2   GPUs
			want Comparison
		}{
			{
				name: "Equal VRAM",
				g1: []GPU{
					{
						Index: 0,
						VRAM:  16384,
					},
				},
				g2: []GPU{
					{
						Index: 1,
						VRAM:  16384,
					},
				},
				want: Equal,
			},
			{
				name: "Worse VRAM",
				g1: []GPU{
					{
						Index: 0,
						VRAM:  8450,
					},
				},
				g2: []GPU{
					{
						Index: 1,
						VRAM:  16384,
					},
				},
				want: Worse,
			},
			{
				name: "Better VRAM",
				g1: []GPU{
					{
						Index: 0,
						VRAM:  16384,
					},
				},
				g2: []GPU{
					{
						Index: 1,
						VRAM:  8192,
					},
				},
				want: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.g1.Compare(tt.g2)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardware_Calculable_Add(t *testing.T) {
	t.Parallel()

	t.Run("CPU Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			c1   CPU
			c2   CPU
			want CPU
		}{
			{
				name: "Add two CPUs",
				c1: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				want: CPU{
					Cores:      3,
					ClockSpeed: 1000,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.c1.Add(tt.c2)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.c1)
			})
		}
	})

	t.Run("RAM Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			r1   RAM
			r2   RAM
			want RAM
		}{
			{
				name: "Add two RAMs",
				r1: RAM{
					Size: 1024,
				},
				r2: RAM{
					Size: 512,
				},
				want: RAM{
					Size: 1536,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.r1.Add(tt.r2)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.r1)
			})
		}
	})

	t.Run("Disk Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			d1   Disk
			d2   Disk
			want Disk
		}{
			{
				name: "Add two Disks",
				d1: Disk{
					Size: 1024,
				},
				d2: Disk{
					Size: 512,
				},
				want: Disk{
					Size: 1536,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.d1.Add(tt.d2)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.d1)
			})
		}
	})

	t.Run("GPU Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			g1   GPU
			g2   GPU
			want GPU
		}{
			{
				name: "Add two GPUs",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       8192,
				},
				want: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       24576,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.g1.Add(tt.g2)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.g1)
			})
		}
	})

	t.Run("GPUs Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			g1   GPUs
			g2   GPUs
			want GPUs
		}{
			{
				name: "Add two GPUs",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
				},
				g2: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				want: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       24576,
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.g1.Add(tt.g2)
				require.NoError(t, err)
				require.Equal(t, tt.want, tt.g1)
			})
		}
	})
}

func TestHardware_Calculable_Subtract(t *testing.T) {
	t.Parallel()

	t.Run("CPU Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			c1      CPU
			c2      CPU
			want    any
			wantErr bool
		}{
			{
				name: "Subtract two CPUs",
				c1: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				want: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
			},
			{
				name: "Subtract two CPUs with the other having more cores",
				c1: CPU{
					Cores:      1,
					ClockSpeed: 1000,
				},
				c2: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				wantErr: true,
				want:    "core: underflow",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.c1.Subtract(tt.c2)
				if tt.wantErr {
					errorMessage := tt.want.(string)
					require.Error(t, err)
					require.ErrorContains(t, err, errorMessage)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.want, tt.c1)
				}
			})
		}
	})

	t.Run("RAM Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			r1      RAM
			r2      RAM
			want    any
			wantErr bool
		}{
			{
				name: "Subtract two RAMs",
				r1: RAM{
					Size: 1024,
				},
				r2: RAM{
					Size: 512,
				},
				want: RAM{
					Size: 512,
				},
			},
			{
				name: "Subtract two RAMs with the other having more size",
				r1: RAM{
					Size: 512,
				},
				r2: RAM{
					Size: 1024,
				},
				wantErr: true,
				want:    "size: underflow",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.r1.Subtract(tt.r2)
				if tt.wantErr {
					errorMessage := tt.want.(string)
					require.Error(t, err)
					require.ErrorContains(t, err, errorMessage)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.want, tt.r1)
				}
			})
		}
	})

	t.Run("Disk Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			d1      Disk
			d2      Disk
			want    any
			wantErr bool
		}{
			{
				name: "Subtract two Disks",
				d1: Disk{
					Size: 1024,
				},
				d2: Disk{
					Size: 512,
				},
				want: Disk{
					Size: 512,
				},
			},
			{
				name: "Subtract two Disks with the other having more size",
				d1: Disk{
					Size: 512,
				},
				d2: Disk{
					Size: 1024,
				},
				wantErr: true,
				want:    "size: underflow",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.d1.Subtract(tt.d2)
				if tt.wantErr {
					errorMessage := tt.want.(string)
					require.Error(t, err)
					require.ErrorContains(t, err, errorMessage)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.want, tt.d1)
				}
			})
		}
	})

	t.Run("GPU Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			g1      GPU
			g2      GPU
			want    any
			wantErr bool
		}{
			{
				name: "Subtract two GPUs",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       8192,
				},
				want: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       8192,
				},
			},
			{
				name: "Subtract two GPUs with the other having more VRAM",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       8192,
				},
				g2: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				wantErr: true,
				want:    "total VRAM: underflow",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.g1.Subtract(tt.g2)
				if tt.wantErr {
					errorMessage := tt.want.(string)
					require.Error(t, err)
					require.ErrorContains(t, err, errorMessage)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.want, tt.g1)
				}
			})
		}
	})

	t.Run("GPUs Checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			g1      GPUs
			g2      GPUs
			want    any
			wantErr bool
		}{
			{
				name: "Subtract two GPUs",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
				},
				g2: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				want: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
			},
			{
				name: "Subtract two GPUs with the other having more VRAM",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				g2: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
				},
				wantErr: true,
				want:    "total VRAM: underflow",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.g1.Subtract(tt.g2)
				if tt.wantErr {
					errorMessage := tt.want.(string)
					require.Error(t, err)
					require.ErrorContains(t, err, errorMessage)
				} else {
					require.NoError(t, err)
					expected, ok := tt.want.([]GPU)
					require.True(t, ok)
					require.Equal(t, GPUs(expected), tt.g1)
				}
			})
		}
	})
}

func TestHardware_Equal(t *testing.T) {
	t.Parallel()

	// gpu.Equal() works as a deep equal
	t.Run("GPU Checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			g1   GPU
			g2   GPU
			want bool
		}{
			{
				name: "Equal GPUs",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				want: true,
			},
			{
				name: "Different GPUs",
				g1: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				g2: GPU{
					Index:      1,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.D",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.g1.Equal(tt.g2)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("GPUs Checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			g1   GPUs
			g2   GPUs
			want bool
		}{
			{
				name: "Equal GPUs",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				g2: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				want: true,
			},
			{
				name: "Equal GPUs with different order",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				g2: []GPU{
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
				},
				want: true,
			},
			{
				name: "Different GPUs",
				g1: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				g2: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
				},
				want: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.g1.Equal(tt.g2)
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardware_CPU(t *testing.T) {
	t.Parallel()
	t.Run("ClockSpeedInGHz", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			cpu  CPU
			want float64
		}{
			{
				name: "lower value",
				cpu: CPU{
					ClockSpeed: 1000000000,
				},
				want: 1,
			},
			{
				name: "higher value",
				cpu: CPU{
					ClockSpeed: 200000000000000000000,
				},
				want: 200000000000,
			},
			{
				name: "zero value",
				cpu: CPU{
					ClockSpeed: 0,
				},
				want: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.cpu.ClockSpeedInGHz()
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("ComputeInGHz", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			cpu  CPU
			want float64
		}{
			{
				name: "lower value",
				cpu: CPU{
					Cores:      1,
					ClockSpeed: 1000000000,
				},
				want: 1,
			},
			{
				name: "higher value",
				cpu: CPU{
					Cores:      2,
					ClockSpeed: 200000000000000000000,
				},
				want: 400000000000,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.cpu.ComputeInGHz()
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("Compute", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			c    CPU
			want float64
		}{
			{
				name: "Compute CPU",
				c: CPU{
					Cores:      2,
					ClockSpeed: 1000,
				},
				want: 2000,
			},
			{
				name: "Compute CPU with 0 Cores",
				c: CPU{
					Cores:      0,
					ClockSpeed: 1000,
				},
				want: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.c.Compute()
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardware_RAM(t *testing.T) {
	t.Run("SizeInGB", func(t *testing.T) {
		tests := []struct {
			name string
			ram  RAM
			want uint64
		}{
			{
				name: "lower value",
				ram: RAM{
					Size: 1e9,
				},
				want: 1,
			},
			{
				name: "higher value",
				ram: RAM{
					Size: 100 * 1e9,
				},
				want: 100,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.ram.SizeInGB()
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardware_Disk(t *testing.T) {
	t.Parallel()
	t.Run("SizeInGB", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			disk Disk
			want uint64
		}{
			{
				name: "lower value",
				disk: Disk{
					Size: 1e9,
				},
				want: 1,
			},
			{
				name: "higher value",
				disk: Disk{
					Size: 100 * 1e9,
				},
				want: 100,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := tt.disk.SizeInGB()
				require.Equal(t, tt.want, got)
			})
		}
	})
}

func TestHardware_GPU(t *testing.T) {
	t.Parallel()

	t.Run("parseGPUVendor", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name   string
			vendor string
			want   GPUVendor
		}{
			{
				name:   "Parse NVIDIA GPU vendor",
				vendor: "some NVIDIA Corporation gpu",
				want:   GPUVendorNvidia,
			},
			{
				name:   "Parse AMD GPU vendor",
				vendor: "a amd gpu with some other stuff",
				want:   GPUVendorAMDATI,
			},
			{
				name:   "Parse Intel GPU vendor",
				vendor: "intel gpu",
				want:   GPUVendorIntel,
			},
			{
				name:   "Parse AMD GPU vendor",
				vendor: "a gpu with ati in the name",
				want:   GPUVendorAMDATI,
			},
			{
				name:   "Parse unknown GPU vendor",
				vendor: "some other gpu",
				want:   GPUVendorUnknown,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := ParseGPUVendor(tt.vendor)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("MaxFreeVRAMGPU", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			gpus GPUs
			want GPU
		}{
			{
				name: "MaxFreeVRAMGPU",
				gpus: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				want: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.gpus.MaxFreeVRAMGPU()
				require.NoError(t, err)
				require.True(t, tt.want.Equal(got))
			})
		}
	})

	t.Run("GetWithIndex", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			gpus GPUs
			idx  int
			want GPU
		}{
			{
				name: "Get GPU with index 0",
				gpus: []GPU{
					{
						Index:      0,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.C",
						Model:      "Tesla T4A100",
						VRAM:       16384,
					},
					{
						Index:      1,
						Vendor:     GPUVendorNvidia,
						PCIAddress: "AAAA:BB:CC.D",
						Model:      "Tesla T4A100",
						VRAM:       8192,
					},
				},
				idx: 0,
				want: GPU{
					Index:      0,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "Tesla T4A100",
					VRAM:       16384,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := tt.gpus.GetWithIndex(tt.idx)
				require.NoError(t, err)
				require.True(t, tt.want.Equal(got))
			})
		}
	})

	t.Run("Copy", func(t *testing.T) {
		t.Parallel()

		gpus := GPUs{
			{
				Index:      0,
				Vendor:     GPUVendorNvidia,
				PCIAddress: "AAAA:BB:CC.C",
				Model:      "Tesla T4A100",
				VRAM:       16384,
			},
			{
				Index:      1,
				Vendor:     GPUVendorNvidia,
				PCIAddress: "AAAA:BB:CC.D",
				Model:      "Tesla T4A100",
				VRAM:       8192,
			},
		}

		// Ensure that the copy is a deep copy
		copiedGPUs := gpus.Copy()
		require.Equal(t, gpus, copiedGPUs)

		// Ensure that the copy is returned safely
		copiedGPUs[0].VRAM = 0
		copiedGPUs[1].VRAM = 0
		require.NotEqual(t, gpus, copiedGPUs)
	})
}
