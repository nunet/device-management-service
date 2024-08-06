package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestVMHandler() *VMHandler {
	return &VMHandler{}
}

func TestStartCustom(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*VMHandler)
		expectedStatus int
		expectedBody   string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestVMHandler()
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			router := setupTestRouter()
			router.POST("/vm/start-custom", handler.StartCustom)

			w := performRequest(router, "POST", "/vm/start-custom")

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func TestStartDefault(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*VMHandler)
		expectedStatus int
		expectedBody   string
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestVMHandler()
			if tt.setupMock != nil {
				tt.setupMock(handler)
			}

			router := setupTestRouter()
			router.POST("/vm/start-default", handler.StartCustom)

			w := performRequest(router, "POST", "/vm/start-default")

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

// TODO: test it with incorrect bind json
//func TestStartCustomHandler(t *testing.T) {
//	router := SetupMockRouter()
//
//	body := CustomVM{
//		KernelImagePath: "/foo/bar",
//		FilesystemPath:  "/baz/foo",
//		VCPUCount:       1,
//		MemSizeMib:      5,
//		TapDevice:       "baz",
//	}
//	bodyBytes, _ := json.Marshal(body)
//
//	w := httptest.NewRecorder()
//	req, _ := http.NewRequest("POST", "/api/v1/vm/start-custom", bytes.NewBuffer(bodyBytes))
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, 200, w.Code)
//	assert.Contains(t, w.Body.String(), "VM started successfully")
//}
//
//func TestStartDefaultHandler(t *testing.T) {
//	router := SetupMockRouter()
//
//	body := DefaultVM{
//		KernelImagePath: "/foo/bar",
//		FilesystemPath:  "/baz/foo",
//		PublicKey:       "foobar",
//		NodeID:          "foobaz",
//	}
//	bodyBytes, _ := json.Marshal(body)
//
//	w := httptest.NewRecorder()
//	req, _ := http.NewRequest("POST", "/api/v1/vm/start-default", bytes.NewBuffer(bodyBytes))
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, 200, w.Code)
//	assert.Contains(t, w.Body.String(), "VM started successfully")
//}
