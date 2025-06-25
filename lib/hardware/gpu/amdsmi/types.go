// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package amdsmi

import "C"

import (
	"unsafe"
)

// SocketHandle is a Go type that encapsulates the C amdsmi_socket_handle (void*).
type SocketHandle struct {
	handle unsafe.Pointer
}

// ProcessorHandle is a Go type that encapsulates the C amdsmi_processor_handle (void*).
type ProcessorHandle struct {
	handle unsafe.Pointer
}

// BoardInfo is a Go representation of the C amdsmi_board_info_t struct.
type BoardInfo struct {
	ModelNumber      string
	ProductSerial    string
	FruID            string
	ProductName      string
	ManufacturerName string
}

// VRAM is a Go representation of the C amdsmi_vram_info_t struct.
type VRAM struct {
	Total    uint32
	Used     uint32
	Reserved [5]uint64
}
