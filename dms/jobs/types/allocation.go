// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

// AllocationStatus is a representation of the execution status
type AllocationStatus string

const (
	AllocationPending    AllocationStatus = "pending"
	AllocationRunning    AllocationStatus = "running"
	AllocationStopped    AllocationStatus = "stopped"
	AllocationFailed     AllocationStatus = "failed"
	AllocationCompleted  AllocationStatus = "completed"
	AllocationTerminated AllocationStatus = "terminated"
	AllocationStandby    AllocationStatus = "standby" // Allocation is in standby mode (redundancy)
)
