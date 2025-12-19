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
	"time"
)

// DurationToUnit converts a duration to the specified time unit
func DurationToUnit(duration time.Duration, unit string) (float64, error) {
	switch unit {
	case "second":
		return duration.Seconds(), nil
	case "minute":
		return duration.Minutes(), nil
	case "hour":
		return duration.Hours(), nil
	default:
		return 0, fmt.Errorf("unsupported time unit: %s", unit)
	}
}

// SecondsToUnit converts seconds (float64) to the specified time unit
func SecondsToUnit(seconds float64, unit string) (float64, error) {
	duration := time.Duration(seconds * float64(time.Second))
	return DurationToUnit(duration, unit)
}
