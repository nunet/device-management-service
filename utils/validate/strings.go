// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
	"regexp"
	"strings"
)

// IsBlank checks if a string is empty or contains only whitespace
func IsBlank(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// IsNotBlank checks if a string is not empty and does not contain only whitespace
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// IsLiteral just checks if a variable is a string
func IsLiteral(s interface{}) bool {
	_, ok := s.(string)
	return ok
}

var dnsNameRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// IsDNSNameValid performs DNS name validation using regex
func IsDNSNameValid(name string) bool {
	// DNSNameRegex validates DNS names with the following rules:
	// - Starts with a letter or number
	// - Contains only letters, numbers, dots, and hyphens
	// - Each label (part between dots) starts with a letter/number and ends with a letter/number
	// - Each label is between 1 and 63 characters
	// - Total length between 1 and 253 characters
	return dnsNameRegex.MatchString(name)
}

var dockerImageRegex = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)*(?::\d+)?)/)?(?:[a-z0-9]+(?:(?:[._-][a-z0-9]+)*)/)*[a-z0-9]+(?:(?:[._-][a-z0-9]+)*)?(?::[a-zA-Z0-9]+(?:(?:[._-][a-zA-Z0-9]+)*)?)?$`)

// IsDockerImageValid validates docker image format with a single regex
func IsDockerImageValid(image string) bool {
	// Docker image format: [registry/]name[:tag]
	// Registry: Optional domain name with optional port
	// Name: Required lowercase letters, numbers, separators (., _, -)
	// Tag: Optional letters, numbers, separators (., _, -)
	return dockerImageRegex.MatchString(image)
}

var envVarKeyRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

// IsEnvVarKeyValid validates environment variable key format
func IsEnvVarKeyValid(key string) bool {
	// Environment variable keys should start with a letter or underscore
	// and can contain only letters, numbers, and underscores
	return envVarKeyRegex.MatchString(key)
}
