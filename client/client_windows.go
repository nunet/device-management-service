package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	winio "github.com/Microsoft/go-winio"
)

// createUnixTransport sets up a Unix socket transport
func createUnixSocketTransport(_ Config) (http.RoundTripper, error) {
	return nil, ErrProtocolNotAvailable
}

// createNPipeTransport sets up a Windows named pipe transport
func createNPipeTransport(opts Config) (http.RoundTripper, error) {
	// Construct full pipe path
	pipePath := fmt.Sprintf("\\\\.\\pipe\\%s", strings.TrimPrefix(opts.Host, `\\.\pipe\`))

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return winio.DialPipe(pipePath, &opts.ConnectTimeout)
		},
	}
	return transport, nil
}
