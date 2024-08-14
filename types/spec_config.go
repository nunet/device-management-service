package types

import (
	"errors"
	"strings"

	"gitlab.com/nunet/device-management-service/utils/validate"
)

// SpecConfig represents a configuration for a spec
// A SpecConfig can be used to define an engine spec, a storage volume, etc.
type SpecConfig struct {
	// Type of the spec (e.g. docker, firecracker, storage, etc.)
	Type string `json:"type"`
	// Params of the spec
	Params map[string]interface{} `json:"params,omitempty"`
}

type Config interface {
	GetNetworkConfig() *SpecConfig
}

// NewSpecConfig creates a new SpecConfig with the given type
func NewSpecConfig(t string) *SpecConfig {
	return &SpecConfig{
		Type:   t,
		Params: make(map[string]interface{}),
	}
}

// WithParam adds a new key-value pair to the spec params
func (s *SpecConfig) WithParam(key string, value interface{}) *SpecConfig {
	if s.Params == nil {
		s.Params = make(map[string]interface{})
	}
	s.Params[key] = value
	return s
}

// Normalize ensures that the spec config is in a valid state
func (s *SpecConfig) Normalize() {
	if s == nil {
		return
	}

	s.Type = strings.TrimSpace(s.Type)

	// Ensure that an empty and nil map are treated the same
	if len(s.Params) == 0 {
		s.Params = make(map[string]interface{})
	}
}

// Validate checks if the spec config is valid
func (s *SpecConfig) Validate() error {
	if s == nil {
		return errors.New("nil spec config")
	}
	if validate.IsBlank(s.Type) {
		return errors.New("missing spec type")
	}
	return nil
}

// IsType returns true if the current SpecConfig is of the given type
func (s *SpecConfig) IsType(t string) bool {
	if s == nil {
		return false
	}
	t = strings.TrimSpace(t)
	return strings.EqualFold(s.Type, t)
}

// IsEmpty returns true if the spec config is empty
func (s *SpecConfig) IsEmpty() bool {
	return s == nil || (validate.IsBlank(s.Type) && len(s.Params) == 0)
}
