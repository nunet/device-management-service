// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package behaviors

// ChunkedTransferRequest provides position-based chunked transfer fields that
// behaviors can embed in their request types
//
// Ack confirms receipt of all bytes before Offset. The client advances Offset
// to request the next chunk; setting Ack equal to the previous NextOffset
// acknowledges successful delivery and enables future behaviors to track
// transfer progress or resume from the last acked position
type ChunkedTransferRequest struct {
	Ack      int64 `json:"ack,omitempty"`
	Offset   int64 `json:"offset,omitempty"`
	MaxBytes int   `json:"max_bytes,omitempty"`
}

// ChunkedTransferResponse provides the corresponding chunk metadata and payload
// that behaviors can embed in their response types for chunked messaging
type ChunkedTransferResponse struct {
	Offset     int64  `json:"offset,omitempty"`
	NextOffset int64  `json:"next_offset,omitempty"`
	TotalSize  int64  `json:"total_size,omitempty"`
	EOF        bool   `json:"eof,omitempty"`
	Data       []byte `json:"data,omitempty"`
}

const (
	// DefaultChunkSize is the recommended max raw payload bytes per chunk = 512KB
	// Currently the max is set by network/libp2p at 1MB
	DefaultChunkSize = 512 * 1024

	// DefaultLogChunkSize is the default chunk size for log transfer behaviors
	DefaultLogChunkSize = DefaultChunkSize
)

// LogStream identifies which allocation log file to read in chunked mode.
type LogStream string

const (
	LogStreamStdout LogStream = "stdout"
	LogStreamStderr LogStream = "stderr"
)

// CapChunkSize returns size capped at maxSize, or DefaultChunkSize when size is zero.
func CapChunkSize(size, maxSize int) int {
	if size <= 0 {
		return maxSize
	}
	if size > maxSize {
		return maxSize
	}
	return size
}
