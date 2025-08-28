// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

const (
	allocName       = "a"
	origDNSName     = "orig"
	modifiedDNSName = "mutated"
)

func TestEnsembleCfgReaderDeepCopy(t *testing.T) {
	t.Parallel()
	t.Run("mutate original after copy", func(t *testing.T) {
		t.Parallel()
		orig := jobtypes.EnsembleConfig{V1: &jobtypes.EnsembleConfigV1{
			EscalationStrategy: jobtypes.EscalationStrategyRedeploy,
			Allocations:        map[string]jobtypes.AllocationConfig{allocName: {DNSName: origDNSName}},
			Nodes:              map[string]jobtypes.NodeConfig{},
		}}
		reader := jobtypes.NewEnsembleCfgReader(orig)
		copied := reader.Read()

		// mutate original
		orig.V1.Allocations[allocName] = jobtypes.AllocationConfig{DNSName: modifiedDNSName}

		assert.Equal(t, origDNSName, copied.V1.Allocations[allocName].DNSName,
			"copy should retain original value after original is mutated")
	})

	t.Run("mutate copy", func(t *testing.T) {
		t.Parallel()
		orig := jobtypes.EnsembleConfig{V1: &jobtypes.EnsembleConfigV1{
			EscalationStrategy: jobtypes.EscalationStrategyRedeploy,
			Allocations:        map[string]jobtypes.AllocationConfig{allocName: {DNSName: origDNSName}},
			Nodes:              map[string]jobtypes.NodeConfig{},
		}}
		reader := jobtypes.NewEnsembleCfgReader(orig)
		copied := reader.Read()

		// mutate copy
		copied.V1.Allocations[allocName] = jobtypes.AllocationConfig{DNSName: modifiedDNSName}

		assert.Equal(t, origDNSName, orig.V1.Allocations[allocName].DNSName,
			"original should remain unchanged after copy is mutated")
	})
}

func TestManifestReaderDeepCopy(t *testing.T) {
	t.Parallel()
	t.Run("mutate original after copy", func(t *testing.T) {
		t.Parallel()
		orig := jobtypes.EnsembleManifest{
			Allocations: map[string]jobtypes.AllocationManifest{allocName: {DNSName: origDNSName}},
			Nodes:       map[string]jobtypes.NodeManifest{},
		}
		reader := jobtypes.NewManifestReader(orig)
		copied := reader.Read()

		// mutate original
		orig.Allocations[allocName] = jobtypes.AllocationManifest{DNSName: modifiedDNSName}

		assert.Equal(t, origDNSName, copied.Allocations[allocName].DNSName,
			"copy should retain original value after original is mutated")
	})

	t.Run("mutate copy", func(t *testing.T) {
		t.Parallel()
		orig := jobtypes.EnsembleManifest{
			Allocations: map[string]jobtypes.AllocationManifest{allocName: {DNSName: origDNSName}},
			Nodes:       map[string]jobtypes.NodeManifest{},
		}
		reader := jobtypes.NewManifestReader(orig)
		copied := reader.Read()

		// mutate copy
		copied.Allocations[allocName] = jobtypes.AllocationManifest{DNSName: modifiedDNSName}

		assert.Equal(t, origDNSName, orig.Allocations[allocName].DNSName,
			"original should remain unchanged after copy is mutated")
	})
}
