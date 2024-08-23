package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/internal/config"

	"github.com/gin-gonic/gin"
)

const readHeaderTimeout = 10 * time.Second

func TestGetInternalBaseURL(t *testing.T) {
	// Test with a valid internal endpoint
	endpoint, err := InternalAPIURL("http", "/swagger/doc.json", "")
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	expected := "http://localhost:9999/swagger/doc.json"
	if endpoint != expected {
		t.Errorf("Expected %s, but got %s", expected, endpoint)
	}

	// Test with an empty internal endpoint
	_, err = InternalAPIURL("", "", "")
	assert.NotNil(t, err, "Expected an error, but got none")
}

func TestMakeInternalRequest(t *testing.T) { // *
	ctx := context.Background()
	// Test with a valid request
	type respType struct {
		Data string `json:"data"`
	}
	mockEndpoint := "/test_endpoint"
	mockResp := respType{Data: "the mock data"}
	router := gin.Default()
	router.GET(mockEndpoint, func(c *gin.Context) {
		c.JSON(http.StatusOK, mockResp)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.GetConfig().Rest.Port),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("error: ", err)
		}
	}()

	resp, err := MakeInternalRequest(nil, "GET", mockEndpoint, "", nil)
	assert.Nil(t, err, fmt.Sprintf("Expected no error, but got %v", err))

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err, fmt.Sprintf("Error reading response body: %v", err))

	assert.Equal(t, http.StatusOK, resp.StatusCode, "makeInternalRequest wasn't successful")

	expectedContentType := "application/json; charset=utf-8"
	assert.Equal(
		t,
		expectedContentType,
		resp.Header.Get("Content-Type"),
		fmt.Sprintf(
			"Expected Content-Type header %s, but got %s",
			expectedContentType,
			resp.Header.Get("Content-Type")))

	var dataStore respType
	err = json.Unmarshal(body, &dataStore)
	assert.Nil(t, err, fmt.Sprintf("Error unmarshaling response body: %v", err))

	// // Test with an invalid internal endpoint
	resp, err = MakeInternalRequest(nil, "GET", "", "", nil)
	assert.NotNil(t, err, "Expected an error, but got none")
	assert.Nil(t, resp, fmt.Sprintf("Expected resp to be nil but go %+v instead", resp))

	// shutdown server
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Println("error:", err)
	}
}
