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
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
)

func Capabilities(ctx *godog.ScenarioContext) {
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
	ctx.Step(`^"([^"]*)" deployment should not succeed on "([^"]*)"$`, deploymentShouldNotSucceedOn)
	ctx.Step(`^"([^"]*)" revokes a token from "([^"]*)" via "([^"]*)"$`, revokesATokenFromVia)
}

// When "Bob" revokes a token from "Alice" via "nunet"
// cpName: Bob
// spName: Alice
// orgName: nunet
func revokesATokenFromVia(ctx context.Context, cpName, spName, orgName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	tokenMap, err := tc.TokenMap()
	require.NoError(t, err)
	assert.NotEmpty(t, tokenMap)

	orgMap, err := tc.OrganizationMap()
	require.NoError(t, err)
	assert.NotEmpty(t, orgMap)

	orgCtx := utils.GetOrganization(orgMap, orgName)
	assert.NotEmpty(t, orgCtx)

	_, cpCtx := utils.GetNodeAndContext(nodes, cpName)
	assert.NotEmpty(t, cpCtx)

	_, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotEmpty(t, cpDmsCtx)

	token, found := tokenMap[strings.ToLower(spName)]
	assert.True(t, found)

	err = utils.RevokeFromPrivateNetwork(token, cpCtx, cpDmsCtx, orgCtx)
	require.NoError(t, err)

	time.Sleep(30 * time.Second)
	return tc.Unwrap(), nil
}

func deploymentShouldNotSucceedOn(ctx context.Context, spName, cpName string) (context.Context, error) {
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

	timeout := time.After(120 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	deployed := false
	hasError := false
	done := false
	for !done {
		select {
		case <-timeout:
			done = true
		case <-ticker.C:
			ensembleStatus, err := spDmsCtx.EnsembleStatus(ensembleID)
			switch {
			case err != nil:
				hasError = true
				done = true
			case strings.EqualFold(ensembleStatus, "failed"):
				deployed = false
				done = true
			case strings.EqualFold(ensembleStatus, "running"):
				deployed = true
				done = true
			case strings.EqualFold(ensembleStatus, "completed"):
				deployed = true
				done = true
			}
		}
	}
	require.True(t, hasError || !deployed)

	return tc.Unwrap(), nil
}
