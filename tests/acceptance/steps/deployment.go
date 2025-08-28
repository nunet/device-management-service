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
