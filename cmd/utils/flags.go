// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
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
func (t TimeValue) Set(s string) error {
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
func (t TimeValue) Type() string {
	return "time"
}

// String returns the string representation of the time.Time value.
func (t TimeValue) String() string {
	if t.Time == nil || t.Time.IsZero() {
		return ""
	}
	return t.Time.String()
}

func GetTime(f *pflag.FlagSet, name string) (time.Time, error) {
	t := time.Time{}
	flag := f.Lookup(name)
	if flag == nil {
		return t, fmt.Errorf("flag %s not found", name)
	}
	if flag.Value == nil || flag.Value.Type() != new(TimeValue).Type() {
		return t, fmt.Errorf("flag %s has wrong type or no value", name)
	}
	val := flag.Value.(*TimeValue)
	return *val.Time, nil
}
