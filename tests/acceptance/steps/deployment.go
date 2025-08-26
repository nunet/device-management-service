package steps

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
)

// Deployment registers all step definitions for deployment feature
func Deployment(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if err := hooks.SaveLogs(ctx); err != nil {
			return ctx, err
		}
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.Step(`^the following nodes$`, theFollowingNodes)
	ctx.Step(`^"([^"]*)" has deployed "([^"]*)" on "([^"]*)"$`, hasDeployedOn)
	ctx.Step(`^"([^"]*)" deployment is (\w+)$`, deploymentIs)
	ctx.Step(`^"([^"]*)" ensemble should return "([^"]*)"$`, ensembleShouldReturn)
	ctx.Step(`^"([^"]*)" has services deployed on "([^"]*)" and "([^"]*)"$`, hasServicesDeployedOn)
	ctx.Step(`^"([^"]*)" has service deployed on "([^"]*)"$`, hasServiceDeployedOn)
	ctx.Step(`^"([^"]*)" deployment should be (\w+) on "([^"]*)"$`, deploymentShouldBeOn)
	ctx.Step(`^"([^"]*)" deployment should not be (\w+) on "([^"]*)"$`, deploymentShouldNotBeOn)
	ctx.Step(`^"([^"]*)" updates deployment to remove "([^"]*)"$`, updatesDeploymentToRemove)
	ctx.Step(`^"([^"]*)" updates deployment to add "([^"]*)"$`, updatesDeploymentToAdd)
	ctx.Step(`^"([^"]*)" updates deployment to run on "([^"]*)"$`, updatesDeploymentToRunOn)
	ctx.Step(`^"([^"]*)" has deployed ensemble with (\d+) allocations? on "([^"]*)"$`, hasDeployedEnsembleWithAllocationsOn)
	ctx.Step(`^"([^"]*)" updates deployment to add (\d+) allocation$`, updatesDeploymentToAddAllocation)
	ctx.Step(`^"([^"]*)" deployment should have (\d+) allocations? running on "([^"]*)"$`, deploymentShouldHaveAllocationsRunningOn)
	ctx.Step(`^"([^"]*)" updates deployment to remove (\d+) allocation$`, updatesDeploymentToRemoveAllocation)
}

func hasDeployedOn(ctx context.Context, spName, ensembleName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	sp, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	cp, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cp)
	assert.NotNil(t, cpDmsCtx)

	spInfo, err := spDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, spInfo)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, cpInfo)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, cpAddr)

	err = spDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	ensemblePath := fmt.Sprintf("ensembles/%s", ensembleName)
	file := utils.FindTestdata(ensemblePath)

	ensemble, err := utils.UploadEnsemble(sp, file)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	tc = tc.WithEnsembleFile(ensemble)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.peer = \"%s\"", cpInfo.ID), ensemble})
	assert.NoError(t, err)

	ensembleID, err := spDmsCtx.Deploy(ensemble)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	tc = tc.WithEnsembleID(ensembleID)
	return tc.Unwrap(), nil
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

func updatesDeploymentToAddAllocation(ctx context.Context, spName string, count int) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	tc = tc.WithManifest(manifest)

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	for i := 1; i <= count; i++ {
		newAllocName := fmt.Sprintf("new-nginx-%d", i)

		// Add new allocation definition by dereferencing nginx_service
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".allocations.%s = .nginx_wrapper", newAllocName), ensemble})
		assert.NoError(t, err)

		// Add allocation to node1
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.allocations += [\"%s\"]", newAllocName), ensemble})
		assert.NoError(t, err)

		// Add port configuration for new allocation
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.ports += [{\"private\": 80, \"public\": %d, \"allocation\": \"%s\"}]", 17000+i, newAllocName), ensemble})
		assert.NoError(t, err)
	}

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func deploymentShouldHaveAllocationsRunningOn(ctx context.Context, spName string, count int, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)
	cp := nodeMap[cpName]
	sp := nodeMap[spName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(status, "Running")
	}, 60*time.Second, 1*time.Second)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotEmpty(t, cpInfo)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
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

func updatesDeploymentToRemoveAllocation(ctx context.Context, spName string, count int) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	allocs := slices.Collect(maps.Keys(manifest.Allocations))
	assert.NotEmpty(t, allocs)

	// since ranging through map keys is arbitrary
	// this can be considered a random pick
	alloc := allocs[0]

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	assert.Less(t, count, len(allocs))

	// remove top-level allocations definition
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.allocations.%s)", alloc), ensemble})
	assert.NoError(t, err)

	// remove allocation mapping inside node
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.node1.allocations[] | select(. == \"%s\"))", alloc), ensemble})
	assert.NoError(t, err)

	// remove port configuration
	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.node1.ports[] | select(.allocation == \"%s\"))", alloc), ensemble})
	assert.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func hasServiceDeployedOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	return hasDeployedOn(ctx, spName, "nginx.yaml", cpName)
}

func deploymentIs(ctx context.Context, spName, status string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	_, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, spDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	return tc.Unwrap(), nil
}

func ensembleShouldReturn(ctx context.Context, spName, expected string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	sp, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)

	path, err := spDmsCtx.LogsFromAllocation(ensembleID, "alloc1")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)

	// TODO: keep it consistent on DMS, rename log file as stdout.log instead
	out, err := sp.RunCMD([]string{"cat", filepath.Join(path, "stdout.logs")})
	assert.NoError(t, err)
	assert.Contains(t, out, expected)
	return nil
}

func updatesDeploymentToRemove(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	sp, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	var matchNode string
	for name, info := range manifest.Nodes {
		if info.Peer == cpInfo.ID {
			matchNode = name
		}
	}

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.%s)", matchNode), ensemble})
	assert.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func updatesDeploymentToRunOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	sp, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	nodes := slices.Sorted(maps.Keys(manifest.Nodes))
	assert.Len(t, nodes, 1)

	selected := nodes[0]

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.%s.peer = \"%s\"", selected, cpInfo.ID), ensemble})
	assert.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func updatesDeploymentToAdd(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	sp, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	alloc := "new-nginx"
	port := 17001

	// Add new allocation
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".allocations.%s = .nginx_wrapper", alloc), ensemble})
	assert.NoError(t, err)

	// Add allocation to node1
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.allocations = [\"%s\"]", alloc), ensemble})
	assert.NoError(t, err)

	// Add port configuration for new allocation
	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.ports += [{\"private\": 80, \"public\": %d, \"allocation\": \"%s\"}]", port, alloc), ensemble})
	assert.NoError(t, err)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node2.peer = \"%s\"", cpInfo.ID), ensemble})
	assert.NoError(t, err)

	err = spDmsCtx.UpdateEnsemble(ensembleID, ensemble)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func deploymentShouldBeOn(ctx context.Context, spName, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	_, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

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

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	_, spDmsCtx := utils.NodeWithDMS(nodeMap, spName)
	assert.NotNil(t, spDmsCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodeMap, cpName)
	assert.NotNil(t, cpDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)

	var found bool
	for _, node := range manifest.Nodes {
		if node.Peer == cpInfo.ID {
			found = true
		}
	}
	assert.False(t, found)

	return tc.Unwrap(), nil
}
