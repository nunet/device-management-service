package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/internal/config"
)

// InternalAPIURL is a helper method to compose API URLs
func InternalAPIURL(protocol, endpoint, query string) (string, error) {
	if protocol == "" || endpoint == "" {
		return "", fmt.Errorf("protocol and endpoint values must be specified")
	}

	if protocol != "http" && protocol != "ws" {
		return "", fmt.Errorf("invalid protocol: %s", protocol)
	}

	port := config.GetConfig().Rest.Port
	if port == 0 {
		return "", fmt.Errorf("port is not configured")
	}

	serverURL := url.URL{
		Scheme:   protocol,
		Host:     fmt.Sprintf("localhost:%d", port),
		Path:     endpoint,
		RawQuery: query,
	}

	return serverURL.String(), nil
}

// MakeInternalRequest is a helper method to make call to DMS's own API
func MakeInternalRequest(c *gin.Context, methodType, internalEndpoint, query string, body []byte) (*http.Response, error) {
	endpoint, err := InternalAPIURL("http", internalEndpoint, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		methodType,
		endpoint,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	client := http.Client{}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	if c != nil {
		req.Header = c.Request.Header.Clone()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
		// c.JSON(400, gin.H{
		// 	"message":   fmt.Sprintf("Error making %s request to %s", methodType, internalEndpoint),
		// 	"timestamp": time.Now(),
		// })
		// return
	}

	return resp, nil
}

func MakeRequest(_ *gin.Context, client *http.Client, uri string, body []byte, _ string) error {
	// set the HTTP method, url, and request body
	req, err := http.NewRequest(http.MethodPut, uri, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("unable to make new request: %v", err)
	}

	// set the request header Content-Type for json
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	_, err = client.Do(req)
	if err != nil {
		// c.JSON(400, gin.H{
		// 	"message":   errMsg,
		// 	"timestamp": time.Now(),
		// })
		// return
		return err
	}
	return nil
}

func ResponseBody(c *gin.Context, methodType, internalEndpoint, query string, body []byte) (responseBody []byte, errMsg error) {
	resp, err := MakeInternalRequest(c, methodType, internalEndpoint, query, body)
	if err != nil {
		return nil, fmt.Errorf("unable to make internal request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %v", err)
	}

	return respBody, nil
}
