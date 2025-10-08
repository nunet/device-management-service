// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dockercompose

import (
	"sync"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/suite"
)

type WarningsTestSuite struct {
	suite.Suite
}

func TestWarningsTestSuite(t *testing.T) {
	suite.Run(t, new(WarningsTestSuite))
}

func (s *WarningsTestSuite) TestWarningCollectorAdd() {
	tests := []struct {
		name        string
		serviceName string
		feature     string
		reason      string
		expected    string
	}{
		{
			name:        "service-level warning",
			serviceName: "web",
			feature:     "build",
			reason:      "NuNet requires pre-built Docker images.",
			expected:    "Service 'web': Unsupported feature 'build' was ignored. Reason: NuNet requires pre-built Docker images.",
		},
		{
			name:        "top-level warning",
			serviceName: "",
			feature:     "configs",
			reason:      "top-level 'configs' are not supported.",
			expected:    "Top-level feature 'configs' was ignored. Reason: top-level 'configs' are not supported.",
		},
		{
			name:        "empty service name treated as top-level",
			serviceName: "",
			feature:     "networks",
			reason:      "multiple networks are not supported.",
			expected:    "Top-level feature 'networks' was ignored. Reason: multiple networks are not supported.",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			wc.Add(tt.serviceName, tt.feature, tt.reason)

			warnings := wc.Get()
			s.Len(warnings, 1)
			s.Equal(tt.expected, warnings[0])
		})
	}
}

func (s *WarningsTestSuite) TestWarningCollectorMultipleWarnings() {
	wc := &WarningCollector{}

	wc.Add("web", "build", "NuNet requires pre-built Docker images.")
	wc.Add("", "configs", "top-level 'configs' are not supported.")
	wc.Add("db", "privileged", "privileged mode is not supported for security reasons.")

	warnings := wc.Get()
	s.Len(warnings, 3)

	expectedWarnings := []string{
		"Service 'web': Unsupported feature 'build' was ignored. Reason: NuNet requires pre-built Docker images.",
		"Top-level feature 'configs' was ignored. Reason: top-level 'configs' are not supported.",
		"Service 'db': Unsupported feature 'privileged' was ignored. Reason: privileged mode is not supported for security reasons.",
	}

	s.ElementsMatch(expectedWarnings, warnings)
}

func (s *WarningsTestSuite) TestWarningCollectorConcurrency() {
	wc := &WarningCollector{}

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	warningsPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < warningsPerGoroutine; j++ {
				wc.Add("service", "feature", "reason")
			}
		}(i)
	}

	wg.Wait()

	warnings := wc.Get()
	s.Len(warnings, numGoroutines*warningsPerGoroutine)
}

func (s *WarningsTestSuite) TestWarningCollectorGetCopy() {
	wc := &WarningCollector{}
	wc.Add("web", "build", "test reason")

	warnings1 := wc.Get()
	warnings2 := wc.Get()

	// Verify we get copies, not the same slice
	s.Equal(warnings1, warnings2)
	s.NotSame(&warnings1[0], &warnings2[0]) // Different underlying arrays

	// Modifying one shouldn't affect the other
	warnings1[0] = "modified"
	warnings3 := wc.Get()
	s.NotEqual(warnings1[0], warnings3[0])
}

func (s *WarningsTestSuite) TestWarningCollectorEmpty() {
	wc := &WarningCollector{}
	warnings := wc.Get()
	s.Empty(warnings)
	s.NotNil(warnings) // Should return empty slice, not nil
}

func (s *WarningsTestSuite) TestCheckUnsupportedTopLevelFeatures() {
	tests := []struct {
		name             string
		project          *types.Project
		expectedCount    int
		expectedFeatures []string
	}{
		{
			name: "project with configs and secrets",
			project: &types.Project{
				Configs: map[string]types.ConfigObjConfig{
					"config1": {},
				},
				Secrets: map[string]types.SecretConfig{
					"secret1": {},
				},
			},
			expectedCount:    2,
			expectedFeatures: []string{"configs", "secrets"},
		},
		{
			name: "project with only configs",
			project: &types.Project{
				Configs: map[string]types.ConfigObjConfig{
					"config1": {},
					"config2": {},
				},
			},
			expectedCount:    1,
			expectedFeatures: []string{"configs"},
		},
		{
			name: "project with only secrets",
			project: &types.Project{
				Secrets: map[string]types.SecretConfig{
					"secret1": {},
				},
			},
			expectedCount:    1,
			expectedFeatures: []string{"secrets"},
		},
		{
			name:             "project with no unsupported features",
			project:          &types.Project{},
			expectedCount:    0,
			expectedFeatures: []string{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			checkUnsupportedTopLevelFeatures(tt.project, wc)

			warnings := wc.Get()
			s.Len(warnings, tt.expectedCount)

			for _, feature := range tt.expectedFeatures {
				found := false
				for _, warning := range warnings {
					if containsFeature(warning, feature) {
						found = true
						break
					}
				}
				s.True(found, "Expected warning for feature '%s' not found", feature)
			}
		})
	}
}

func (s *WarningsTestSuite) TestCheckForUnsupportedServiceFeatures() {
	tests := []struct {
		name             string
		service          types.ServiceConfig
		expectedCount    int
		expectedFeatures []string
	}{
		{
			name: "service with build configuration",
			service: types.ServiceConfig{
				Name: "web",
				Build: &types.BuildConfig{
					Context: "./",
				},
			},
			expectedCount:    1,
			expectedFeatures: []string{"build"},
		},
		{
			name: "service with multiple unsupported features",
			service: types.ServiceConfig{
				Name: "web",
				Build: &types.BuildConfig{
					Context: "./",
				},
				CgroupParent:  "parent",
				ContainerName: "custom-name",
				Devices: []types.DeviceMapping{
					{
						Source: "/dev/sda",
					},
				},
			},
			expectedCount:    4,
			expectedFeatures: []string{"build", "cgroup_parent", "container_name", "devices"},
		},
		{
			name: "service with container name only",
			service: types.ServiceConfig{
				Name:          "web",
				ContainerName: "my-container",
			},
			expectedCount:    1,
			expectedFeatures: []string{"container_name"},
		},
		{
			name: "service with devices",
			service: types.ServiceConfig{
				Name: "web",
				Devices: []types.DeviceMapping{
					{
						Source: "/dev/sda",
						Target: "/dev/sda",
					},

					{
						Source: "/dev/sdb",
						Target: "/dev/sdb",
					},
				},
			},
			expectedCount:    1,
			expectedFeatures: []string{"devices"},
		},
		{
			name:             "service with no unsupported features",
			service:          types.ServiceConfig{Name: "web"},
			expectedCount:    0,
			expectedFeatures: []string{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			checkForUnsupportedServiceFeatures(tt.service, wc)

			warnings := wc.Get()
			s.Len(warnings, tt.expectedCount)

			for _, feature := range tt.expectedFeatures {
				found := false
				for _, warning := range warnings {
					if containsFeature(warning, feature) {
						found = true
						break
					}
				}
				s.True(found, "Expected warning for feature '%s' not found", feature)
			}
		})
	}
}

func (s *WarningsTestSuite) TestWarningMessageFormat() {
	wc := &WarningCollector{}

	// Test service-level warning format
	wc.Add("web", "build", "NuNet requires pre-built Docker images.")
	warnings := wc.Get()
	s.Contains(warnings[0], "Service 'web':")
	s.Contains(warnings[0], "Unsupported feature 'build'")
	s.Contains(warnings[0], "was ignored")
	s.Contains(warnings[0], "Reason: NuNet requires pre-built Docker images.")

	// Test top-level warning format
	wc = &WarningCollector{}
	wc.Add("", "configs", "top-level 'configs' are not supported.")
	warnings = wc.Get()
	s.Contains(warnings[0], "Top-level feature 'configs'")
	s.Contains(warnings[0], "was ignored")
	s.Contains(warnings[0], "Reason: top-level 'configs' are not supported.")
}

// Helper function to check if a warning message contains a specific feature
func containsFeature(warning, feature string) bool {
	if len(warning) == 0 || len(feature) == 0 {
		return false
	}

	// Check if the feature name appears in the warning message
	for i := 0; i <= len(warning)-len(feature); i++ {
		if warning[i:i+len(feature)] == feature {
			return true
		}
	}
	return false
}
