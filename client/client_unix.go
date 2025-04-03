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
