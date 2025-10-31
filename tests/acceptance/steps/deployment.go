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
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
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

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	cp, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cp)
	assert.NotNil(t, cpDmsCtx)

	spInfo, err := spDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotNil(t, spInfo)

	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotNil(t, cpInfo)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, cpAddr)

	err = spDmsCtx.Connect(cpAddr)
	require.NoError(t, err)

	ensemblePath := fmt.Sprintf("ensembles/%s", ensembleName)
	file := utils.FindTestdata(ensemblePath)

	ensemble, err := utils.UploadFile(sp, file)
	require.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	tc = tc.WithEnsembleFile(ensemble)

	// Upload scripts listed in the ensemble file if needed
	err = utils.UploadScripts(sp, ensemble)
	require.NoError(t, err)

	_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.peer = \"%s\"", cpInfo.ID), ensemble})
	require.NoError(t, err)

	ensembleID, err := spDmsCtx.Deploy(ensemble)
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	tc = tc.WithEnsembleID(ensembleID)
	return tc.Unwrap(), nil
}

func deploymentIs(ctx context.Context, spName, status string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	_, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, spDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	var wantStatus string
	switch status {
	case "fail":
		// currently, invalid ensembles will be stuck at bidding phase,
		// thus always displaying preparing status
		// if we see preparing 3 times in a row we consider a failed deployment
		wantStatus = jobtypes.DeploymentStatusPreparing.String()
		wantSeen := 3
		seen := 0
		wait := 1 * time.Second

		for range wantSeen {
			ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
			require.NoError(t, err)
			if strings.EqualFold(ensembleStatus, wantStatus) {
				seen++
			} else {
				break
			}
			time.Sleep(wait)
		}
		assert.Equal(t, wantSeen, seen, "wanted %s status %d times, got %d", wantStatus, wantSeen, seen)
	default:
		wantStatus = status
		require.Eventually(t, func() bool {
			ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
			require.NoError(t, err)
			return strings.EqualFold(ensembleStatus, wantStatus)
		}, 60*time.Second, 1*time.Second)
	}

	return tc.Unwrap(), nil
}

func ensembleShouldReturn(ctx context.Context, spName, expected string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotNil(t, manifest)

	allocs := slices.Collect(maps.Keys(manifest.Allocations))
	assert.NotEmpty(t, allocs)

	alloc := allocs[0]

	path, err := spDmsCtx.LogsFromAllocation(ensembleID, alloc)
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	logFile := "stdout.log"

	out, err := sp.RunCMD([]string{"cat", filepath.Join(path, logFile)})
	require.NoError(t, err)
	assert.Contains(t, out, expected)
	return nil
}
