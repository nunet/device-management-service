// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gpu

import (
	"fmt"
	"strings"

	"gitlab.com/nunet/device-management-service/types"
)

// assignIndexToGPUs assigns an index to each GPU in the list starting from 0
func assignIndexToGPUs(gpus []types.GPU) []types.GPU {
	for i := range gpus {
		gpus[i].Index = i
	}
	return gpus
}

// bdfIDToPCIAddress converts a 64-bit BDFID to a standard PCI address string.
// The PCI address format is 'domain:bus:device.function'.
//
// Taken from the AMD SMI library:
// BDFID = ((DOMAIN & 0xffffffff) << 32) | ((BUS & 0xff) << 8) |
// ((DEVICE & 0x1f) <<3 ) | (FUNCTION & 0x7)
//
// | Name     | Field   |
// ---------- | ------- |
// | Domain   | [64:32] |
// | Reserved | [31:16] |
// | Bus      | [15: 8] |
// | Device   | [ 7: 3] |
// | Function | [ 2: 0] |
func bdfIDToPCIAddress(bdfID uint64) string {
	// Extract Domain: Bits [63:32]
	domain := (bdfID >> 32) & 0xFFFFFFFF

	// Extract Bus: Bits [15:8]
	bus := (bdfID >> 8) & 0xFF

	// Extract Device: Bits [7:3]
	device := (bdfID >> 3) & 0x1F

	// Extract Function: Bits [2:0]
	function := bdfID & 0x7

	// Format each component into hexadecimal with appropriate padding
	// Domain: 4 hex digits, Bus: 2 hex digits, Device: 2 hex digits, Function: 1 hex digit
	return fmt.Sprintf("%04X:%02X:%02X.%X", domain, bus, device, function)
}

// convertBusID converts the BusId array to a correctly formatted PCI address string.
func convertBusID(busID [32]int8) string {
	busIDBytes := make([]byte, len(busID))
	for i, b := range busID {
		busIDBytes[i] = byte(b)
	}

	busIDStr := strings.TrimRight(string(busIDBytes), "\x00")

	// Check if the string starts with extra zero groups and correct the format
	if strings.HasPrefix(busIDStr, "00000000:") {
		// Trim it to the correct format: "0000:XX:YY.Z"
		return "0000" + busIDStr[8:]
	}

	return busIDStr
}
