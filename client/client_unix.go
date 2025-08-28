// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build !windows

package client

import (
	"context"
	"net"
	"net/http"
)

// createUnixTransport sets up a Unix socket transport
func createUnixSocketTransport(opts Config) (http.RoundTripper, error) {
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", opts.Host, opts.ConnectTimeout)
		},
	}
	return transport, nil
}

// createNPipeTransport sets up a Windows named pipe transport
func createNPipeTransport(_ Config) (http.RoundTripper, error) {
	return nil, ErrProtocolNotAvailable
}
