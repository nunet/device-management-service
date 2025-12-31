// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package steps

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	"gitlab.com/nunet/device-management-service/types"
)

// DeploymentUpdate registers all step definitions for deployment update feature
func DeploymentUpdate(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, scenario *godog.Scenario, _ error) (context.Context, error) {
		scenarioName := strings.ReplaceAll(scenario.Name, " ", "_")
		if err := hooks.SaveLogs(ctx, scenarioName); err != nil {
			return ctx, err
		}
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.Step(`^the following nodes$`, theFollowingNodes)
	ctx.Step(`^"([^"]*)" has services deployed on "([^"]*)" and "([^"]*)"$`, hasServicesDeployedOn)
	ctx.Step(`^"([^"]*)" has service deployed on "([^"]*)"$`, hasServiceDeployedOn)
	ctx.Step(`^"([^"]*)" has (\d+) allocations? deployed on "([^"]*)"$`, hasDeployedEnsembleWithAllocationsOn)
	ctx.Step(`^"([^"]*)" deployment is (\w+)$`, deploymentIs)
	ctx.Step(`^"([^"]*)" removes "([^"]*)" from the deployment$`, updatesDeploymentToRemove)
	ctx.Step(`^"([^"]*)" adds "([^"]*)" to the deployment$`, updatesDeploymentToAdd)
	ctx.Step(`^"([^"]*)" updates deployment to run on "([^"]*)"$`, updatesDeploymentToRunOn)
	ctx.Step(`^"([^"]*)" adds (\d+) allocation? to "([^"]*)"$`, updatesDeploymentToAddAllocation)
	ctx.Step(`^"([^"]*)" removes (\d+) allocation from "([^"]*)"$`, updatesDeploymentToRemoveAllocation)
	ctx.Step(`^"([^"]*)" deployment should be (\w+) on "([^"]*)"$`, deploymentShouldBeOn)
	ctx.Step(`^"([^"]*)" deployment should not be (\w+) on "([^"]*)"$`, deploymentShouldNotBeOn)
	ctx.Step(`^"([^"]*)" deployment should have (\d+) allocations? running on "([^"]*)"$`, deploymentShouldHaveAllocationsRunningOn)
}

func hasDeployedEnsembleWithAllocationsOn(ctx context.Context, spName string, count int, cpName string) (context.Context, error) {
	switch count {
	case 1:
		return hasDeployedOn(ctx, spName, "nginx.yaml", cpName)
	case 2:
		return hasDeployedOn(ctx, spName, "two_allocs_one_node.yaml", cpName)
	default:
		return ctx, fmt.Errorf("invalid number of allocations: %d", count)
	}
}

func hasServiceDeployedOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	return hasDeployedOn(ctx, spName, "nginx.yaml", cpName)
}

func updatesDeploymentToAddAllocation(ctx context.Context, spName string, count int, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotEmpty(t, sp)
	assert.NotEmpty(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotEmpty(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	tc = tc.WithManifest(manifest)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotEmpty(t, cpInfo)

	var nodeName string
	for name, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			nodeName = name
			break
		}
	}
	assert.NotEmpty(t, nodeName)

	ensemble, err := tc.EnsembleFile()
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	for i := 1; i <= count; i++ {
		newAllocName := fmt.Sprintf("new-nginx-%d", i)

		// Add new allocation definition by dereferencing nginx_service
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".allocations.%s = .nginx_wrapper", newAllocName), ensemble})
		require.NoError(t, err)

		// Add allocation to node1
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.%s.allocations += [\"%s\"]", nodeName, newAllocName), ensemble})
		require.NoError(t, err)

		// Add port configuration for new allocation
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.%s.ports += [{\"private\": 80, \"public\": %d, \"allocation\": \"%s\"}]", nodeName, 17000+i, newAllocName), ensemble})
		require.NoError(t, err)
	}

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func deploymentShouldHaveAllocationsRunningOn(ctx context.Context, spName string, count int, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotEmpty(t, sp)
	assert.NotEmpty(t, spDmsCtx)

	cp, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotEmpty(t, cp)
	assert.NotEmpty(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		require.NoError(t, err)
		return strings.EqualFold(status, "Running")
	}, 60*time.Second, 1*time.Second)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotEmpty(t, cpInfo)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	var found bool
	for _, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			found = true
			assert.Len(t, node.Allocations, count)
		}
	}
	assert.True(t, found)

	return tc.Unwrap(), nil
}

func updatesDeploymentToRemoveAllocation(ctx context.Context, spName string, count int, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotEmpty(t, sp)
	assert.NotEmpty(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotEmpty(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	allocs := slices.Collect(maps.Keys(manifest.Allocations))
	assert.NotEmpty(t, allocs)

	// since ranging through map keys is arbitrary
	// this can be considered a random pick
	selected := allocs[0]

	alloc, err := types.ParseManifestKey(selected, ensembleID)
	require.NoError(t, err)

	ensemble, err := tc.EnsembleFile()
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	assert.Less(t, count, len(allocs))

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotEmpty(t, cpInfo)

	var nodeName string
	for name, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			nodeName = name
			break
		}
	}
	assert.NotEmpty(t, nodeName)

	// remove top-level allocations definition
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.allocations.%s)", alloc.AllocationName), ensemble})
	require.NoError(t, err)

	// remove allocation mapping inside node
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.%s.allocations[] | select(. == \"%s\"))", nodeName, alloc.AllocationName), ensemble})
	require.NoError(t, err)

	// remove port configuration
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.%s.ports[] | select(.allocation == \"%s\"))", nodeName, alloc.AllocationName), ensemble})
	require.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func updatesDeploymentToRemove(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)

	var matchNode string
	for name, info := range manifest.Nodes {
		if info.Peer == cpInfo.ID {
			matchNode = name
		}
	}

	ensemble, err := tc.EnsembleFile()
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.%s)", matchNode), ensemble})
	require.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func updatesDeploymentToRunOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	ensemble, err := tc.EnsembleFile()
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	manifestNodes := slices.Sorted(maps.Keys(manifest.Nodes))
	assert.Len(t, manifestNodes, 1)

	selected := manifestNodes[0]

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.%s.peer = \"%s\"", selected, cpInfo.ID), ensemble})
	require.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func updatesDeploymentToAdd(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	ensemble, err := tc.EnsembleFile()
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	alloc := "new-nginx"
	port := 17001

	// Add new allocation
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".allocations.%s = .nginx_wrapper", alloc), ensemble})
	require.NoError(t, err)

	// Add allocation to node1
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.allocations = [\"%s\"]", alloc), ensemble})
	require.NoError(t, err)

	// Add port configuration for new allocation
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.ports += [{\"private\": 80, \"public\": %d, \"allocation\": \"%s\"}]", port, alloc), ensemble})
	require.NoError(t, err)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.peer = \"%s\"", cpInfo.ID), ensemble})
	require.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func deploymentShouldBeOn(ctx context.Context, spName, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		require.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)

	var found bool
	for _, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			found = true
		}
	}
	assert.True(t, found)

	return tc.Unwrap(), nil
}

func deploymentShouldNotBeOn(ctx context.Context, spName, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		require.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)

	var found bool
	for _, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			found = true
		}
	}
	assert.False(t, found)

	return tc.Unwrap(), nil
}
