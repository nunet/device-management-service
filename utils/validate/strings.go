// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
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

// Just checks if a variable is a string
func IsLiteral(s interface{}) bool {
	switch s.(type) {
	case string:
		return true
	default:
		return false
	}
}
