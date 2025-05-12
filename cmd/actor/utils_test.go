package actor

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// TestProcessEnsembleYaml tests the ProcessEnsembleYaml function using
// one of our ensemble examples: multiple_allocation.yaml.
func TestProcessEnsembleYaml(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	// Read YAML content from the actual example file
	yamlPath := "../../examples/multiple_allocation.yaml"
	yamlContent, err := os.ReadFile(yamlPath)
	require.NoError(t, err, "Failed to read example YAML file")

	err = fs.WriteFile("/etc/multiple_allocation.yaml", yamlContent, 0o644)
	require.NoError(t, err)

	result, err := ProcessEnsembleYaml(fs, "/etc/multiple_allocation.yaml")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Ensemble.V1)

	// Assert the Ensemble output fields
	assert.Equal(t, jobtypes.EscalationStrategyRedeploy, result.Ensemble.V1.EscalationStrategy)

	// Check allocations
	assert.Len(t, result.Ensemble.V1.Allocations, 2)

	// Check alloc1
	alloc1, ok := result.Ensemble.V1.Allocations["alloc1"]
	assert.True(t, ok)
	assert.Equal(t, jobtypes.ExecutorDocker, alloc1.Executor)
	assert.Equal(t, jobtypes.AllocationTypeService, alloc1.Type)
	assert.Equal(t, jobtypes.AllocationFailureRecoveryRestForOne, alloc1.FailureRecovery)
	assert.Equal(t, 1, int(alloc1.Resources.CPU.Cores))
	assert.Empty(t, alloc1.Resources.GPUs)
	oneGiBInBytes, err := convert.ParseBytesWithDefaultUnit(1, "GiB")
	require.NoError(t, err)
	assert.EqualValues(t, oneGiBInBytes, alloc1.Resources.RAM.Size)
	assert.EqualValues(t, oneGiBInBytes, alloc1.Resources.Disk.Size)
	assert.Equal(t, string(jobtypes.ExecutorDocker), alloc1.Execution.Type)
	assert.Equal(t, "kennethreitz/httpbin", alloc1.Execution.Params["image"])

	// Check alloc2
	alloc2, ok := result.Ensemble.V1.Allocations["alloc2"]
	assert.True(t, ok)
	assert.Equal(t, jobtypes.ExecutorDocker, alloc2.Executor)
	assert.Equal(t, jobtypes.AllocationTypeService, alloc2.Type)
	assert.Equal(t, string(jobtypes.ExecutorDocker), alloc2.Execution.Type)
	assert.Equal(t, "buildpack-deps", alloc2.Execution.Params["image"])
	assert.Equal(t, []interface{}{"tail", "-f", "/dev/null"}, alloc2.Execution.Params["cmd"])

	// Check nodes
	assert.Len(t, result.Ensemble.V1.Nodes, 2)

	// Check node1
	node1, ok := result.Ensemble.V1.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, []string{"alloc1"}, node1.Allocations)
	assert.Equal(t, 2, node1.Redundancy)
	assert.Equal(t, jobtypes.NodeFailureRecoveryStayDown, node1.FailureRecovery)
	assert.Len(t, node1.Ports, 1)
	assert.Equal(t, 17000, node1.Ports[0].Public)
	assert.Equal(t, 80, node1.Ports[0].Private)
	assert.Equal(t, "alloc1", node1.Ports[0].Allocation)

	// Check node2
	node2, ok := result.Ensemble.V1.Nodes["node2"]
	assert.True(t, ok)
	assert.Equal(t, []string{"alloc2"}, node2.Allocations)
	assert.Equal(t, jobtypes.NodeFailureRecoveryRestart, node2.FailureRecovery)
	assert.Len(t, node2.Ports, 1)
	assert.Equal(t, 17001, node2.Ports[0].Public)
	assert.Equal(t, 80, node2.Ports[0].Private)
	assert.Equal(t, "alloc2", node2.Ports[0].Allocation)
}
