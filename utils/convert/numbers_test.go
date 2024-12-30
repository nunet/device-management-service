// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPositiveFloat64(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		expectedValue float64
		errorMsg      string
	}{
		{"float64", float64(42.5), 42.5, ""},
		{"float32", float32(42.5), 42.5, ""},
		{"int", 42, 42, ""},
		{"int32", int32(42), 42, ""},
		{"int64", int64(42), 42, ""},
		{"uint", uint(42), 42, ""},
		{"uint32", uint32(42), 42, ""},
		{"uint64", uint64(42), 42, ""},
		{"string valid", "42.5", 42.5, ""},
		{"negative float64", float64(-42.5), 0, "test must be positive"},
		{"zero float64", float64(0), 0, "test must be positive"},
		{"invalid type", []int{1, 2, 3}, 0, "test must be a valid number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToPositiveFloat64(tt.input, "test")
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, got)
		})
	}
}

func TestExtractValueAndUnit(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		expectedValue float64
		expectedUnit  string
		errorMsg      string
	}{
		// Basic numeric values
		{"integer", 123, 123, "", ""},
		{"float", 123.45, 123.45, "", ""},
		{"negative", -123.45, -123.45, "", ""},
		{"string integer", "123", 123, "", ""},
		{"string float", "123.45", 123.45, "", ""},
		{"string negative", "-123.45", -123.45, "", ""},
		{"string with spaces", "  123.45  ", 123.45, "", ""},

		// Scientific notation
		{"scientific notation positive", "1.23e4", 12300, "", ""},
		{"scientific notation negative exp", "1.23e-4", 0.000123, "", ""},
		{"scientific notation with sign", "-1.23e4", -12300, "", ""},

		// Valid units with different lengths
		{"single char unit", "123B", 123, "B", ""},
		{"two char unit", "123Hz", 123, "Hz", ""},
		{"three char unit", "123GHz", 123, "GHz", ""},
		{"four char unit", "123GiB", 123, "GiB", ""},

		// Valid units with spaces
		{"spaced single unit", "123 B", 123, "B", ""},
		{"spaced four unit", "123 GiB", 123, "GiB", ""},
		{"multiple spaces before unit", "123   GiB", 123, "GiB", ""},

		// Invalid unit formats
		{"unit with number", "123GB2", 0, "", "invalid numeric value"},
		{"unit with underscore", "123G_B", 0, "", "invalid numeric value"},
		{"unit with hyphen", "123G-B", 0, "", "invalid numeric value"},
		{"unit with dot", "123G.B", 0, "", "invalid numeric value"},

		// Other invalid inputs
		{"invalid format", "abc", 0, "", "invalid numeric value"},
		{"empty string", "", 0, "", "invalid numeric value"},
		{"unit only", "GHz", 0, "", "invalid numeric value"},
		{"multiple dots", "1.2.3", 0, "", "invalid numeric value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, unit, err := extractValueAndUnit(tt.input)
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedUnit, unit)
		})
	}
}

func TestParseBytesWithDefaultUnit(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		defaultUnit   string
		expectedValue uint64
		errorMsg      string
	}{
		{"integer with default", 1024, "GiB", 1024 * 1024 * 1024 * 1024, ""},
		{"float with default", 1.5, "GiB", 1610612736, ""},
		{"string with default", "1024", "MiB", 1073741824, ""},
		{"explicit GiB", "1GiB", "MiB", 1073741824, ""},
		{"explicit MB", "100MB", "GiB", 100000000, ""},
		{"explicit bytes", "1024B", "GiB", 1024, ""},
		{"spaced unit", "1.5 GiB", "MB", 1610612736, ""},
		{"scientific with default", "1.5e3", "MiB", 1572864000, ""},
		{"scientific with unit", "1.5e3MB", "GiB", 1500000000, ""},
		{"invalid format", "abc", "GiB", 0, "invalid numeric value"},
		{"invalid unit", "100XB", "GiB", 0, "unhandled size name"},
		{"empty string", "", "GiB", 0, "invalid numeric value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBytesWithDefaultUnit(tt.input, tt.defaultUnit)
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, got)
		})
	}
}

func TestParseSIWithDefaultUnit(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		defaultUnit   string
		expectedValue float64
		errorMsg      string
	}{
		{"integer with default", 2, "GHz", 2e9, ""},
		{"float with default", 2.4, "GHz", 2.4e9, ""},
		{"string with default", "2.4", "GHz", 2.4e9, ""},
		{"explicit Hz", "2400Hz", "GHz", 2400, ""},
		{"explicit MHz", "2.4MHz", "GHz", 2.4e6, ""},
		{"explicit GHz", "2.4GHz", "MHz", 2.4e9, ""},
		{"spaced unit", "2.4 GHz", "MHz", 2.4e9, ""},
		{"scientific with default", "2.4e3", "MHz", 2.4e9, ""},
		{"scientific with unit", "2.4e-3GHz", "MHz", 2.4e6, ""},
		{"invalid format", "abc", "GHz", 0, "invalid numeric value"},
		{"empty string", "", "GHz", 0, "invalid numeric value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSIWithDefaultUnit(tt.input, tt.defaultUnit)
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.InDelta(t, tt.expectedValue, got, 1e-9)
		})
	}
}
