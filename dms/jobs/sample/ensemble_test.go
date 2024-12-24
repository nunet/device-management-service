// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package sample

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

func TestEnsemble(t *testing.T) {
	// Read the sample YAML file
	samplePath := filepath.Join(".", "ensemble.yaml")
	data, err := os.ReadFile(samplePath)
	require.NoError(t, err, "Failed to read sample file")

	// Create a struct to hold the parsed config
	var config jobs.EnsembleConfig

	// Parse the YAML
	err = parser.Parse(parser.SpecTypeEnsembleV1, data, &config)
	require.NoError(t, err, "Failed to parse sample ensemble")

	// Verify the transformed and validated config
	require.NotNil(t, config.V1, "V1 config should be present")
	assert.NotEmpty(t, config.V1.Nodes, "Nodes should be present")
	assert.NotEmpty(t, config.V1.Edges, "Edges should be present")
	assert.NotEmpty(t, config.V1.Allocations, "Allocations should be present")
	assert.NotEmpty(t, config.V1.Keys, "Keys should be present")
	assert.NotEmpty(t, config.V1.Scripts, "Scripts should be present")
	assert.NotNil(t, config.V1.Supervisor, "Supervisor should be present")

	// Verify node configuration
	node1, ok := config.V1.Nodes["node1"]
	require.True(t, ok, "Node1 should exist")
	assert.Equal(t, "12D3KooWJwNf7KGNwGxgkGXuYvfS5ZKrHhXH2qwvVDiGFsGV5xqz", node1.Peer, "Node1 should have correct peer ID")
	assert.Contains(t, node1.Allocations, "worker1", "Node1 should have worker1 allocation")

	// Verify resource transformation
	worker1, ok := config.V1.Allocations["worker1"]
	require.True(t, ok, "Worker1 allocation should exist")

	expectedRAM, err := convert.ParseBytesWithDefaultUnit(2, "GiB")
	assert.NoError(t, err)

	assert.Equal(t, float32(2), worker1.Resources.CPU.Cores, "Worker1 should have correct CPU cores")
	assert.Equal(t, float64(expectedRAM), worker1.Resources.RAM.Size, "Worker1 should have correct RAM")
	assert.Equal(t, "docker", string(worker1.Executor), "Worker1 should have correct executor")
	assert.Equal(t, "docker", worker1.Execution.Type, "Worker1 should have correct execution type")
	assert.Equal(t, "ubuntu:22.04", worker1.Execution.Params["image"], "Worker1 should have correct image")
}
