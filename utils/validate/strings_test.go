// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLiteral(t *testing.T) {
	intValue := int(2)
	float32Value := float32(3.456)
	float64Value := float64(45.59736)

	stringValue := "some string"

	// negative assertions
	assert.False(t, IsLiteral(intValue))
	assert.False(t, IsLiteral(float32Value))
	assert.False(t, IsLiteral(float64Value))

	// negative assertions
	assert.True(t, IsLiteral(stringValue))
}

func TestIsBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsBlank("   "))
	assert.True(t, IsBlank(""))
	assert.True(t, IsBlank(" "))

	// negative assertions
	assert.False(t, IsBlank("  a  "))
	assert.False(t, IsBlank("a"))
}

func TestIsNotBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsNotBlank("  a  "))
	assert.True(t, IsNotBlank("a"))

	// negative assertions
	assert.False(t, IsNotBlank("   "))
	assert.False(t, IsNotBlank(""))
	assert.False(t, IsNotBlank(" "))
}

func TestIsDNSNameValid(t *testing.T) {
	tests := []struct {
		name     string
		dnsName  string
		expected bool
	}{
		{
			name:     "valid hostname",
			dnsName:  "host-1",
			expected: true,
		},
		{
			name:     "valid fqdn",
			dnsName:  "host-1.example.com",
			expected: true,
		},
		{
			name:     "valid with numbers",
			dnsName:  "host123.example.com",
			expected: true,
		},
		{
			name:     "empty string",
			dnsName:  "",
			expected: false,
		},
		{
			name:     "too long",
			dnsName:  "a" + string(make([]byte, 254)),
			expected: false,
		},
		{
			name:     "starts with dot",
			dnsName:  ".example.com",
			expected: false,
		},
		{
			name:     "ends with dot",
			dnsName:  "example.com.",
			expected: false,
		},
		{
			name:     "starts with hyphen",
			dnsName:  "-example.com",
			expected: false,
		},
		{
			name:     "ends with hyphen",
			dnsName:  "example.com-",
			expected: false,
		},
		{
			name:     "invalid characters",
			dnsName:  "host_1@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDNSNameValid(tt.dnsName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDockerImageValid(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{
			name:  "simple image name",
			image: "ubuntu",
			want:  true,
		},
		{
			name:  "image with tag",
			image: "ubuntu:latest",
			want:  true,
		},
		{
			name:  "image with registry",
			image: "registry.example.com/ubuntu",
			want:  true,
		},
		{
			name:  "image with registry and port",
			image: "registry.example.com:5000/ubuntu",
			want:  true,
		},
		{
			name:  "image with registry, port, and tag",
			image: "registry.example.com:5000/ubuntu:latest",
			want:  true,
		},
		{
			name:  "image with organization",
			image: "organization/repo-name",
			want:  true,
		},
		{
			name:  "image with multiple path segments",
			image: "organization/suborg/repo-name",
			want:  true,
		},
		{
			name:  "image with version tag",
			image: "ubuntu:20.04",
			want:  true,
		},
		{
			name:  "image with complex tag",
			image: "myapp:v1.2.3-alpha",
			want:  true,
		},
		{
			name:  "empty string",
			image: "",
			want:  false,
		},
		{
			name:  "invalid characters",
			image: "my@app:latest",
			want:  false,
		},
		{
			name:  "starts with number",
			image: "123image",
			want:  true,
		},
		{
			name:  "invalid registry format",
			image: "registry..com/ubuntu",
			want:  false,
		},
		{
			name:  "invalid port",
			image: "registry.com:port/ubuntu",
			want:  false,
		},
		{
			name:  "spaces in name",
			image: "my app:latest",
			want:  false,
		},
		{
			name:  "uppercase in name",
			image: "MyApp:latest",
			want:  false,
		},
		{
			name:  "invalid tag format",
			image: "ubuntu:latest!",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDockerImageValid(tt.image)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsEnvVarKeyValid(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "simple key",
			key:  "PATH",
			want: true,
		},
		{
			name: "key with underscore",
			key:  "HOME_DIR",
			want: true,
		},
		{
			name: "key with numbers",
			key:  "API2_KEY",
			want: true,
		},
		{
			name: "starts with underscore",
			key:  "_SECRET",
			want: true,
		},
		{
			name: "single letter",
			key:  "A",
			want: true,
		},
		{
			name: "empty string",
			key:  "",
			want: false,
		},
		{
			name: "starts with number",
			key:  "1PATH",
			want: false,
		},
		{
			name: "contains hyphen",
			key:  "MY-VAR",
			want: false,
		},
		{
			name: "contains space",
			key:  "MY VAR",
			want: false,
		},
		{
			name: "contains special characters",
			key:  "MY@VAR",
			want: false,
		},
		{
			name: "contains dot",
			key:  "MY.VAR",
			want: false,
		},
		{
			name: "only numbers",
			key:  "123",
			want: false,
		},
		{
			name: "only special characters",
			key:  "@#$",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEnvVarKeyValid(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}
