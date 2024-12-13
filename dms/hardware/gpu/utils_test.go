// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gpu

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/types"
)

func TestAssignIndexToGPUs(t *testing.T) {
	gpus := []types.GPU{
		{
			Vendor: types.GPUVendorNvidia,
		},
		{
			Vendor: types.GPUVendorAMDATI,
		},
		{
			Vendor: types.GPUVendorIntel,
		},
	}

	assignIndexToGPUs(gpus)
	require.True(t, gpus[0].Index == 0)
	require.True(t, gpus[0].Vendor == types.GPUVendorNvidia)

	require.True(t, gpus[1].Index == 1)
	require.True(t, gpus[1].Vendor == types.GPUVendorAMDATI)

	require.True(t, gpus[2].Index == 2)
	require.True(t, gpus[2].Vendor == types.GPUVendorIntel)
}

func TestBdfIDToPCIAddress(t *testing.T) {
	tests := []struct {
		name     string
		bdfID    uint64
		expected string
	}{
		{
			name:     "All zeros",
			bdfID:    0x0000000000000000,
			expected: "0000:00:00.0",
		},
		{
			name:     "Maximum values",
			bdfID:    ((0xFFFFFFFF & 0xFFFFFFFF) << 32) | ((0xFF & 0xFF) << 8) | ((0x1F & 0x1F) << 3) | (0x7 & 0x7), //nolint
			expected: "FFFFFFFF:FF:1F.7",
		},
		{
			name:     "Typical value",
			bdfID:    ((0x0001 & 0xFFFFFFFF) << 32) | ((0x02 & 0xFF) << 8) | ((0x03 & 0x1F) << 3) | (0x4 & 0x7),
			expected: "0001:02:03.4",
		},
		{
			name:     "Random value",
			bdfID:    ((0xABCD & 0xFFFFFFFF) << 32) | ((0x12 & 0xFF) << 8) | ((0x1A & 0x1F) << 3) | (0x5 & 0x7),
			expected: "ABCD:12:1A.5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := bdfIDToPCIAddress(test.bdfID)
			if result != test.expected {
				t.Errorf("bdfIDToPCIAddress(%#x) = %s; expected %s", test.bdfID, result, test.expected)
			}
		})
	}
}

func TestConvertBusID(t *testing.T) {
	tests := []struct {
		name     string
		busID    [32]int8
		expected string
	}{
		{
			name:     "Standard format",
			busID:    stringToInt8Array("0000:01:00.0"),
			expected: "0000:01:00.0",
		},
		{
			name:     "With extra zeros",
			busID:    stringToInt8Array("00000000:02:00.1"),
			expected: "0000:02:00.1",
		},
		{
			name:     "Trailing null bytes",
			busID:    appendStringWithNulls("0000:03:00.0"),
			expected: "0000:03:00.0",
		},
		{
			name:     "Empty string",
			busID:    [32]int8{},
			expected: "",
		},
		{
			name:     "No correction needed",
			busID:    stringToInt8Array("0000:04:00.0"),
			expected: "0000:04:00.0",
		},
		{
			name:     "Long input with extra zeros",
			busID:    stringToInt8Array("00000000:05:1F.7"),
			expected: "0000:05:1F.7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := convertBusID(test.busID)
			if result != test.expected {
				t.Errorf("convertBusID(%v) = %s; expected %s", test.busID, result, test.expected)
			}
		})
	}
}

// stringToInt8Array converts a string to a [32]int8 array
func stringToInt8Array(s string) [32]int8 {
	var arr [32]int8
	for i := 0; i < len(s) && i < 32; i++ {
		arr[i] = int8(s[i])
	}
	return arr
}

// appendStringWithNulls creates a [32]int8 array with trailing null bytes
func appendStringWithNulls(s string) [32]int8 {
	arr := stringToInt8Array(s)
	for i := len(s); i < 32; i++ {
		arr[i] = 0
	}
	return arr
}
