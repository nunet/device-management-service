package steps

import (
	"context"
	"fmt"
	"path/filepath"
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
}

func hasDeployedOn(ctx context.Context, spName, ensembleName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	sp := nodeMap[spName]
	cp := nodeMap[cpName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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

	err = cpDmsCtx.Onboard()
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

func hasServiceDeployedOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	return hasDeployedOn(ctx, spName, "nginx.yaml", cpName)
}

func deploymentIs(ctx context.Context, spName, status string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	sp := nodeMap[spName]
	cp := nodeMap[cpName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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
	assert.NotEmpty(t, ensembleID)

	_, err = sp.RunCMD([]string{"yq", "-i", "eval", fmt.Sprintf("del(.nodes.%s)", matchNode), ensemble})
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

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	sp := nodeMap[spName]
	cp := nodeMap[cpName]

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	ensemble, err := tc.EnsembleFile()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	_, err = sp.RunCMD([]string{"yq", "-i", ".allocations.\"nginx-node-2\" = .allocations.\"nginx-node-1\"", ensemble})
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", ".nodes.node2 = .nodes.node1", ensemble})
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", ".nodes.node2.allocations[0] = \"nginx-node-2\"", ensemble})
	assert.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", ".nodes.node2.ports[0].allocation = \"nginx-node-2\"", ensemble})
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

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	sp := nodeMap[spName]
	cp := nodeMap[cpName]

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	sp := nodeMap[spName]
	cp := nodeMap[cpName]

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	require.Eventually(t, func() bool {
		ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return strings.EqualFold(ensembleStatus, status)
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotEmpty(t, manifest)

	cpDmsCtx, ok := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

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
