package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

// ConnectionType represents the different ways to connect to the DMS
type ConnectionType string

const (
	ConnectionTCP        ConnectionType = "tcp"
	ConnectionUnixSocket ConnectionType = "unix"
	ConnectionNPipe      ConnectionType = "npipe"
)

// Config provides configuration for the DMS client
type Config struct {
	// Connection details
	Host      string
	Protocol  ConnectionType
	APIPrefix string
	Version   string

	// TLS configuration
	TLSConfig *tls.Config

	// Timeouts
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
}

// Client represents the main client for interacting with the DMS
type Client struct {
	// HTTP client for making requests
	httpClient *http.Client

	// Connection details
	host      string
	protocol  ConnectionType
	apiPrefix string
	version   string

	// Options for client behavior
	options Config

	// Actor options
	sctx      actor.SecurityContext
	dmsHandle actor.Handle
}

var _ DmsClient = (*Client)(nil)

func NewClientSecurityContext(priv io.Reader, capData io.Reader) (actor.SecurityContext, error) {
	// Generate ephemeral key pair for this session
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key pair: %w", err)
	}

	// Create trust context
	trustKeyData, err := io.ReadAll(priv)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	trustPriv, err := crypto.BytesToPrivateKey(trustKeyData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal private key: %w", err)
	}
	trustCtx, err := did.NewTrustContextWithPrivateKey(trustPriv)
	if err != nil {
		return nil, fmt.Errorf("create trust context: %w", err)
	}

	// Create capability context
	capCtx, err := ucan.LoadCapabilityContext(trustCtx, capData)
	if err != nil {
		return nil, fmt.Errorf("create capability context: %w", err)
	}

	return actor.NewBasicSecurityContext(pubk, privk, capCtx)
}

// NewClient creates a new DMS client with the given options
func NewClient(cfg Config, securityContext actor.SecurityContext) (*Client, error) {
	// Create transport based on connection type
	transport, err := createTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}
	return NewClientWithTransport(cfg, transport, securityContext)
}

func NewClientWithTransport(cfg Config, transport http.RoundTripper, securityContext actor.SecurityContext) (*Client, error) {
	// Set default values
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.Protocol == "" {
		cfg.Protocol = ConnectionTCP
	}

	// Create HTTP client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
	}

	client := &Client{
		httpClient: httpClient,
		host:       cfg.Host,
		protocol:   cfg.Protocol,
		apiPrefix:  cfg.APIPrefix,
		version:    cfg.Version,
		sctx:       securityContext,
		options:    cfg,
	}

	return client, nil
}

// createTransport creates a custom transport based on connection type
func createTransport(opts Config) (http.RoundTripper, error) {
	switch opts.Protocol {
	case ConnectionUnixSocket:
		return createUnixSocketTransport(opts)
	case ConnectionNPipe:
		return createNPipeTransport(opts)
	default:
		return createTCPTransport(opts)
	}
}

// createTCPTransport sets up a TCP transport with TLS support
func createTCPTransport(opts Config) (http.RoundTripper, error) {
	// Base dialer with timeout
	dialer := &net.Dialer{
		Timeout: opts.ConnectTimeout,
	}

	// Create transport with TLS and custom dialer
	transport := &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: opts.TLSConfig,
		// Additional TCP-specific configurations
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return transport, nil
}

// getAPIPath returns the versioned request path to call the API.
func (c *Client) getAPIPath(p string, query url.Values) string {
	apiPath := ""

	if c.version != "" {
		version := "/v" + strings.TrimPrefix(strings.ToLower(c.version), "v")
		apiPath = path.Join(c.apiPrefix, version, p)
	} else {
		apiPath = path.Join(c.apiPrefix, p)
	}
	return (&url.URL{Path: apiPath, RawQuery: query.Encode()}).String()
}

func (c *Client) encodeBody(obj interface{}, headers http.Header) (io.Reader, http.Header, error) {
	if obj == nil {
		return nil, headers, nil
	}
	// encoding/json encodes a nil pointer as the JSON document `null`,
	// irrespective of whether the type implements json.Marshaler or encoding.TextMarshaler.
	// That is almost certainly not what the caller intended as the request body.
	if reflect.TypeOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil() {
		return nil, headers, nil
	}

	data := bytes.NewBuffer(nil)
	if err := json.NewEncoder(data).Encode(obj); err != nil {
		return nil, headers, err
	}
	if headers == nil {
		headers = make(map[string][]string)
	}
	headers["Content-Type"] = []string{"application/json"}
	return data, headers, nil
}

func (c *Client) addHeaders(req *http.Request, headers http.Header) *http.Request {
	for k, v := range headers {
		req.Header[http.CanonicalHeaderKey(k)] = v
	}
	return req
}

// prepareRequest is a private helper method to construct a request
func (c *Client) buildRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	reqBody, headers, err := c.encodeBody(body, nil)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, c.getAPIPath(path, query), reqBody)
	if err != nil {
		return nil, err
	}

	// Add headers
	req = c.addHeaders(req, headers)

	if c.options.TLSConfig != nil {
		req.URL.Scheme = "https"
	} else {
		req.URL.Scheme = "http"
	}

	req.URL.Host = c.host

	if reqBody != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "text/plain")
	}

	return req, nil
}

// ParseResponse is a utility method to parse JSON response body
func parseResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Println(resp)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if body != nil && target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w", err)
		}
	}

	return nil
}

// do performs an HTTP request with retry logic
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	// Check for specific error types
	if err != nil {
		// Handle context errors
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("request canceled: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request deadline exceeded: %w", err)
		}

		// If error is EOF it is probably trying to connect to https server with http client
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("tried to connect to https server with http client: %w", err)
		}
		if errors.Is(err, http.ErrSchemeMismatch) {
			return nil, fmt.Errorf("tried to connect to http server with https client: %w", err)
		}

		// Handle TLS errors
		var tlsErr *tls.CertificateVerificationError
		if errors.As(err, &tlsErr) {
			return nil, fmt.Errorf("tls certificate verification failed: %w", err)
		}

		// Handle socket/permission-related errors
		if c.protocol == ConnectionUnixSocket {
			if os.IsPermission(err) {
				return nil, fmt.Errorf("permission denied for Unix socket: %w", err)
			}
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("unix socket does not exist: %w", err)
			}
		}

		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// sendRequest builds and sends the request, returning the response
func (c *Client) sendRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	req, err := c.buildRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// get performs a GET request
func (c *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	resp, err := c.sendRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}

	// Parse the response
	return parseResponse(resp, target)
}

// post performs a POST request
func (c *Client) post(ctx context.Context, path string, query url.Values, body any, target any) error {
	resp, err := c.sendRequest(ctx, http.MethodPost, path, query, body)
	if err != nil {
		return err
	}

	// Parse the response
	return parseResponse(resp, target)
}
