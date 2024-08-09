package nunet

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

type ValidateTestSuite struct {
	suite.Suite
}

func TestValidateTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateTestSuite))
}

func (s *ValidateTestSuite) TestValidateSpec() {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name: "Valid spec with jobs",
			data: map[string]any{
				"jobs": []any{
					map[string]any{"name": "job1"},
				},
			},
			wantErr: false,
		},
		{
			name:    "Invalid spec: missing jobs",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name: "Invalid spec: empty jobs list",
			data: map[string]any{
				"jobs": []any{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateSpec(nil, tt.data, tree.Path(""))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
		})
	}
}

func (s *ValidateTestSuite) TestValidateJob() {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name: "Valid job with children",
			data: map[string]any{
				"children": []any{
					map[string]any{"name": "child1"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid job with execution",
			data: map[string]any{
				"execution": map[string]any{"command": "echo hello"},
			},
			wantErr: false,
		},
		{
			name: "Invalid job: no children or execution",
			data: map[string]any{
				"name": "invalid job",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := ValidateJob(nil, tt.data, tree.Path("jobs.0"))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
		})
	}
}

func (s *ValidateTestSuite) TestNuNetValidator() {
	validator := NewNuNetValidator()

	validConfig := map[string]any{
		"jobs": []any{
			map[string]any{
				"name": "job1",
				"children": []any{
					map[string]any{
						"name":      "child1",
						"execution": map[string]any{"command": "echo hello"},
					},
				},
			},
			map[string]any{
				"name":      "job2",
				"execution": map[string]any{"command": "echo world"},
			},
		},
	}

	invalidConfig := map[string]any{
		"jobs": []any{
			map[string]any{
				"name": "invalid job",
			},
		},
	}

	s.Run("Valid configuration", func() {
		err := validator.Validate(&validConfig)
		s.NoError(err)
	})

	s.Run("Invalid configuration", func() {
		err := validator.Validate(&invalidConfig)
		s.Error(err)
	})
}
