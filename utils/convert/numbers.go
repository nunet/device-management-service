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
	"strconv"
)

// ToPositiveFloat64 converts various numeric types to float64 and validates it's positive
func ToPositiveFloat64(v any, fieldName string) (float64, error) {
	var result float64
	switch num := v.(type) {
	case float64:
		result = num
	case float32:
		result = float64(num)
	case int:
		result = float64(num)
	case int32:
		result = float64(num)
	case int64:
		result = float64(num)
	case uint:
		result = float64(num)
	case uint32:
		result = float64(num)
	case uint64:
		result = float64(num)
	case string:
		if parsed, err := strconv.ParseFloat(num, 64); err == nil {
			result = parsed
		}
	default:
		return 0, fmt.Errorf("%s must be a valid number", fieldName)
	}

	if result <= 0 {
		return 0, fmt.Errorf("%s must be positive", fieldName)
	}
	return result, nil
}
