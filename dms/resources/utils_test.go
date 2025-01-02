// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources

import (
	"context"
	"testing"

	"github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/db/repositories"
	cloverRepo "gitlab.com/nunet/device-management-service/db/repositories/clover"
	"gitlab.com/nunet/device-management-service/types"
)

// setupManagerRepos prepares a full structure of ManagerRepos to be
// used by tests.
func setupManagerRepos(db *clover.DB) ManagerRepos {
	return ManagerRepos{
		OnboardedResources: cloverRepo.NewOnboardedResources(db),
		ResourceAllocation: cloverRepo.NewResourceAllocation(db),
	}
}

// getOnboardedResourcesFromDB gets the onboarded resources using the repository
func getOnboardedResourcesFromDB(repo repositories.OnboardedResources, t *testing.T) types.OnboardedResources {
	t.Helper()
	onboardedResources, err := repo.Get(context.Background())
	require.NoError(t, err)
	return onboardedResources
}

func assertResources(t *testing.T, expected, actual types.Resources) {
	t.Helper()

	require.Equal(t, expected.CPU.Cores, actual.CPU.Cores)
	require.Equal(t, expected.CPU.ClockSpeed, actual.CPU.ClockSpeed)
	require.Equal(t, expected.RAM, actual.RAM)
	require.Equal(t, expected.Disk, actual.Disk)
	// TODO: GPU
}

// newMockManagerRepos creates a new mock ManagerRepos
func newMockManagerRepos(t *testing.T,
	onboardedResources repositories.OnboardedResources,
	resourceAllocation repositories.ResourceAllocation,
) ManagerRepos {
	t.Helper()

	return ManagerRepos{
		OnboardedResources: onboardedResources,
		ResourceAllocation: resourceAllocation,
	}
}
