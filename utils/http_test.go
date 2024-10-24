// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTTPClient(t *testing.T) {
	baseURL := "http://localhost"
	version := "/api/v1"

	httpClient := NewHTTPClient(baseURL, version)

	assert.Equal(t, baseURL, httpClient.BaseURL)
	assert.Equal(t, version, httpClient.APIVersion)
	assert.Equal(t, http.DefaultClient, httpClient.Client)
}

func TestMakeRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "success"}`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	client := NewHTTPClient(mockServer.URL, "api/v1")

	respBody, statusCode, err := client.MakeRequest("GET", "/test/path", nil)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, statusCode)

	bodyString := strings.TrimSpace(string(respBody))
	expectedBody := `{"message": "success"}`
	assert.Equal(t, expectedBody, bodyString)
}
