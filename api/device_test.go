package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*RESTServer)
		expectedStatus int
		expectedBody   string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &RESTServer{}
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			router := setupTestRouter()
			router.GET("/device/status", handler.DeviceStatus)

			w := performRequest(router, "GET", "/device/status")

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func TestUpdateDeviceStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*RESTServer)
		expectedStatus int
		expectedBody   string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &RESTServer{}
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			router := setupTestRouter()
			router.POST("/device/status", handler.UpdateDeviceStatus)

			w := performRequest(router, "POST", "/device/status")

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}
