// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processors

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

// allocationWindow represents a time window for an allocation
type allocationWindow struct {
	allocationID string
	deploymentID string
	startTime    time.Time
	endTime      time.Time
	isComplete   bool
}

// deploymentWindow represents a time window for a deployment
type deploymentWindow struct {
	deploymentID string
	startTime    time.Time
	endTime      time.Time
	isComplete   bool
}

// processAllocationEndEvent processes end events (CompleteAllocationEvent, StopAllocationEvent)
// and returns the allocationID if found. Returns empty string and false if not found or invalid.
func processAllocationEndEvent(evt *usage.Usage) (allocationID string, ok bool) {
	if evt.Timestamp.IsZero() {
		return "", false
	}

	switch evt.EventType {
	case events.CompleteAllocationEvent:
		var data events.CompleteAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return "", false
		}
		return data.AllocationID, true
	case events.StopAllocationEvent:
		var data events.StopAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return "", false
		}
		return data.AllocationID, true
	default:
		return "", false
	}
}

// formatAmount formats a float64 amount as a string with 8 decimal places
func formatAmount(amount float64) string {
	return fmt.Sprintf("%.8f", amount)
}
