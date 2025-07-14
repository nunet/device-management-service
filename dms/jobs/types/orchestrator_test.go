// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import "testing"

func TestDeploymentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    DeploymentStatus
		want string
	}{
		{
			name: "Preparing",
			d:    DeploymentStatusPreparing,
			want: "Preparing",
		},
		{
			name: "Generating",
			d:    DeploymentStatusGenerating,
			want: "Generating",
		},
		{
			name: "Committing",
			d:    DeploymentStatusCommitting,
			want: "Committing",
		},
		{
			name: "Provisioning",
			d:    DeploymentStatusProvisioning,
			want: "Provisioning",
		},
		{
			name: "Running",
			d:    DeploymentStatusRunning,
			want: "Running",
		},
		{
			name: "Updating",
			d:    DeploymentStatusUpdating,
			want: "Updating",
		},
		{
			name: "Failed",
			d:    DeploymentStatusFailed,
			want: "Failed",
		},
		{
			name: "ShuttingDown",
			d:    DeploymentStatusShuttingDown,
			want: "ShuttingDown",
		},
		{
			name: "Completed",
			d:    DeploymentStatusCompleted,
			want: "Completed",
		},
		{
			name: "Unknown",
			d:    100,
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.d.String()
			if got != tt.want {
				t.Errorf("DeploymentStatus String = %v, want %v", got, tt.want)
			}
		})
	}
}
