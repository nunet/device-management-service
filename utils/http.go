package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

type HTTPClient struct {
	BaseURL    string
	APIVersion string
	Client     *http.Client
}

func NewHTTPClient(baseURL, version string) *HTTPClient {
	return &HTTPClient{
		BaseURL:    baseURL,
		APIVersion: version,
		Client:     http.DefaultClient,
	}
}

// MakeRequest performs an HTTP request with the given method, path, and body
// It returns the response body, status code, and an error if any
func (c *HTTPClient) MakeRequest(method, relativePath string, body []byte) ([]byte, int, error) {
	url, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse base URL: %v", err)
	}

	url.Path = path.Join(c.APIVersion, relativePath)

	req, err := http.NewRequest(method, url.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %v", err)
	}

	return respBody, resp.StatusCode, nil
}
