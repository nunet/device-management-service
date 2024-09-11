package utils

import (
	"fmt"
	"strings"
	"time"
)

// TimeValue adapts time.Time for use as a flag.
type TimeValue struct {
	Time    *time.Time
	Formats []string
}

// NewTimeValue creates a new TimeValue.
func NewTimeValue(t *time.Time, formats ...string) *TimeValue {
	if formats == nil {
		formats = []string{
			time.RFC822,
			time.RFC822Z,
			time.RFC3339,
			time.RFC3339Nano,
			time.DateTime,
			time.DateOnly,
		}
	}
	return &TimeValue{
		Time:    t,
		Formats: formats,
	}
}

// Set time.Time value from string based on accepted formats.
func (t *TimeValue) Set(s string) error {
	s = strings.TrimSpace(s)
	for _, format := range t.Formats {
		v, err := time.Parse(format, s)
		if err == nil {
			*t.Time = v
			return nil
		}
	}
	return fmt.Errorf("format must be one of: %v", strings.Join(t.Formats, ", "))
}

// Type name for time.Time flags.
func (t *TimeValue) Type() string {
	return "time"
}

// String returns the string representation of the time.Time value.
func (t *TimeValue) String() string {
	if t.Time == nil || t.Time.IsZero() {
		return ""
	}
	return t.Time.String()
}
