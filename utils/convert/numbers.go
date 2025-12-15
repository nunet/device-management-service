// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package convert

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

const (
	// BytesPerGB is the number of bytes in 1 GB (SI standard: 1 GB = 10^9 bytes)
	BytesPerGB = 1e9
)

// ToPositiveFloat64 converts various numeric types to float64 and validates it's positive
func ToPositiveFloat64(value any, fieldName string) (float64, error) {
	var result float64
	switch v := value.(type) {
	case float64:
		result = v
	case float32:
		result = float64(v)
	case int, int8, int16, int32, int64:
		result = float64(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		result = float64(reflect.ValueOf(v).Uint())
	case string:
		v = strings.TrimSpace(v)
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a valid number", fieldName)
		}
		result = parsed
	default:
		return 0, fmt.Errorf("%s must be a valid number", fieldName)
	}

	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", fieldName)
	}

	return result, nil
}

// Matches: optional sign, digits, optional decimal point and digits, optional e/E notation, optional space, optional unit
var numberWithUnitRegex = regexp.MustCompile(`^([+-]?\d*\.?\d+(?:[eE][+-]?\d+)?)\s*([a-zA-Z]+)?$`)

// extractValueAndUnit takes an input value of any type and returns the numeric value and unit if present.
// The numeric value is parsed and normalized to avoid scientific notation.
func extractValueAndUnit(value any) (float64, string, error) {
	// Convert value to string
	strValue := fmt.Sprintf("%v", value)

	// Remove leading and trailing spaces
	strValue = strings.TrimSpace(strValue)

	// Match the input string
	matches := numberWithUnitRegex.FindStringSubmatch(strValue)

	// If the number part is not captured, return an error
	if len(matches) == 0 {
		return 0, "", fmt.Errorf("invalid numeric value: %v", value)
	}

	// Parse the numeric part to float
	numValue, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid numeric value: %v", value)
	}

	// Return the numeric value and unit (if present)
	return numValue, matches[2], nil
}

// ParseBytesWithDefaultUnit converts a value to bytes using humanize.ParseBytes.
// If the value has no unit suffix, appends the default unit before parsing.
// Example: ParseBytesWithDefaultUnit("100", "GiB") -> 107374182400 (100 GiB in bytes)
func ParseBytesWithDefaultUnit(value any, defaultUnit string) (uint64, error) {
	num, unit, err := extractValueAndUnit(value)
	if err != nil {
		return 0, err
	}

	// Use the extracted unit or default
	if unit == "" {
		unit = defaultUnit
	}

	// Format the value with the unit for humanize.ParseBytes
	strVal := fmt.Sprintf("%.10f%s", num, unit)
	return humanize.ParseBytes(strVal)
}

// ParseSIWithDefaultUnit converts a value to SI units using humanize.ParseSI.
// If the value has no unit suffix, appends the default unit before parsing.
// Example: ParseSIWithDefaultUnit("2.4", "GHz") -> 2400000000 (2.4 GHz in Hz)
func ParseSIWithDefaultUnit(value any, defaultUnit string) (float64, error) {
	num, unit, err := extractValueAndUnit(value)
	if err != nil {
		return 0, err
	}

	// Use the extracted unit or default
	if unit == "" {
		unit = defaultUnit
	} else if defaultUnit != "" {
		// Handle both cases:
		// 1. SI prefixes (e.g., Hz vs GHz, B vs MB)
		// 2. Same length units with different prefixes (e.g., MB vs GB)
		if !strings.HasSuffix(strings.ToLower(unit), strings.ToLower(defaultUnit)) &&
			!strings.HasSuffix(strings.ToLower(defaultUnit), strings.ToLower(unit)) &&
			(len(unit) != len(defaultUnit) || !strings.EqualFold(unit[1:], defaultUnit[1:])) {
			return 0, fmt.Errorf("invalid unit %q", unit)
		}
	}

	// Format the value with the unit for humanize.ParseSI
	strVal := fmt.Sprintf("%.10f%s", num, unit)
	parsed, _, err := humanize.ParseSI(strVal)
	return parsed, err
}

func ToBytesFormat(value any) (string, error) {
	v, err := strconv.ParseUint(fmt.Sprintf("%v", value), 10, 64)
	if err != nil {
		return "", err
	}
	return humanize.BigBytes(humanize.BigByte.SetUint64(v)), nil
}

func ToSIFormatWithUnit(value any, unit string) (string, error) {
	v, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
	if err != nil {
		return "", err
	}
	return humanize.SI(v, unit), nil
}

// BytesToGB converts bytes to gigabytes (GB) as float64 for precision.
// Uses SI standard: 1 GB = 10^9 bytes = 1,000,000,000 bytes
// Supports various numeric types: uint64, int64, float64, string, etc.
// Example: BytesToGB(5000000000) -> 5.0
func BytesToGB(bytes any) (float64, error) {
	var bytesValue uint64
	switch v := bytes.(type) {
	case uint64:
		bytesValue = v
	case uint32:
		bytesValue = uint64(v)
	case uint16:
		bytesValue = uint64(v)
	case uint8:
		bytesValue = uint64(v)
	case uint:
		bytesValue = uint64(v)
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case int:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case float32:
		if v < 0 {
			return 0, fmt.Errorf("bytes cannot be negative")
		}
		bytesValue = uint64(v)
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid bytes value: %w", err)
		}
		bytesValue = parsed
	default:
		return 0, fmt.Errorf("unsupported type for bytes conversion: %T", bytes)
	}

	return float64(bytesValue) / BytesPerGB, nil
}
