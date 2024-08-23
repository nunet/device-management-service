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
