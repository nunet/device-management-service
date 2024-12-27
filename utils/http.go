// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
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

	c.Client.Timeout = 80 * time.Second

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
