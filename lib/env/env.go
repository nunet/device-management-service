package env

import (
	"os"
	"sync"
)

// EnvironmentProvider provides access to environment variables
type EnvironmentProvider interface {
	Getenv(key string) string
	Setenv(key, value string) error
}

// OSEnvironment uses the actual OS environment
type OSEnvironment struct{}

var _ EnvironmentProvider = (*OSEnvironment)(nil)

func NewOSEnvironment() OSEnvironment {
	return OSEnvironment{}
}

func (e OSEnvironment) Getenv(key string) string {
	return os.Getenv(key)
}

func (e OSEnvironment) Setenv(key, value string) error {
	return os.Setenv(key, value)
}

// MockEnvironment provides an in-memory environment for testing
type MockEnvironment struct {
	vars map[string]string
	mu   sync.RWMutex
}

func NewMockEnvironment() *MockEnvironment {
	return &MockEnvironment{
		vars: make(map[string]string),
	}
}

func (e *MockEnvironment) Getenv(key string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	v, ok := e.vars[key]
	if !ok {
		return ""
	}
	return v
}

func (e *MockEnvironment) Setenv(key, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vars[key] = value
	return nil
}
