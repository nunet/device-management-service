package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestNewServer tests the creation of a new server
func TestNewServer(t *testing.T) {
	t.Parallel()
	config := &ServerConfig{
		Addr: "localhost",
		Port: 8080,
	}
	server := NewServer(config)

	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.Equal(t, config, server.config)
}

// TestSetupRoutes tests the setup of routes
func TestSetupRoutes(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	config := &ServerConfig{
		Addr: "localhost",
		Port: 8080,
	}
	server := NewServer(config)
	server.SetupRoutes()

	// Test that all expected routes exist
	routes := server.router.Routes()
	expectedRoutes := map[string]string{
		"/health":                 "GET",
		"/api/v1/actor/handle":    "GET",
		"/api/v1/actor/send":      "POST",
		"/api/v1/actor/invoke":    "POST",
		"/api/v1/actor/broadcast": "POST",
	}

	// Create a map of actual routes
	actualRoutes := make(map[string]bool)
	for _, route := range routes {
		key := route.Path + "-" + route.Method
		actualRoutes[key] = true
	}

	// Check that all expected routes exist
	for path, method := range expectedRoutes {
		key := path + "-" + method
		assert.True(t, actualRoutes[key], "Route %s with method %s does not exist", path, method)
	}

	// Test health endpoint works
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)
	server.router.ServeHTTP(w, req)

	// Assert that the health endpoint is working
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	config := &ServerConfig{
		Addr: "localhost",
		Port: 8080,
	}
	server := NewServer(config)

	server.HealthCheck(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"status\":\"ok\"")
}

// TestCorsConfig tests the CORS configuration
func TestCorsConfig(t *testing.T) {
	t.Parallel()
	corsConfig := getCorsConfig()
	assert.Contains(t, corsConfig.AllowMethods, "GET")
	assert.Contains(t, corsConfig.AllowMethods, "POST")
	assert.Contains(t, corsConfig.AllowMethods, "PUT")
	assert.Contains(t, corsConfig.AllowMethods, "PATCH")
	assert.Contains(t, corsConfig.AllowMethods, "DELETE")
	assert.Contains(t, corsConfig.AllowMethods, "HEAD")
	assert.Contains(t, corsConfig.AllowMethods, "OPTIONS")

	// Assert that the allowed headers are set correctly
	assert.Contains(t, corsConfig.AllowHeaders, "Access-Control-Allow-Origin")
	assert.Contains(t, corsConfig.AllowHeaders, "Origin")
	assert.Contains(t, corsConfig.AllowHeaders, "Content-Length")
	assert.Contains(t, corsConfig.AllowHeaders, "Content-Type")

	// Assert that the allowed origins are set correctly
	assert.Contains(t, corsConfig.AllowOrigins, "http://localhost:9991")
	assert.Contains(t, corsConfig.AllowOrigins, "http://localhost:9992")
	assert.False(t, corsConfig.AllowCredentials)
}

// TestSetupRouter tests the setup of the router
func TestSetupRouter(t *testing.T) {
	t.Parallel()
	middlewares := []gin.HandlerFunc{}
	router := setupRouter(middlewares)
	assert.NotNil(t, router)
}
