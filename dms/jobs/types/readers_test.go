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
