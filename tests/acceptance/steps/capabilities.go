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
	"slices"
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
	ctx.Step(`^"([^"]*)" says hello to "([^"]*)"$`, saysHelloto)
	ctx.Step(`^"([^"]*)" deployment is (\w+)$`, deploymentIs)
	ctx.Step(`^"([^"]*)" should respond with his <DID>$`, respondsToHelloWithDID)
	ctx.Step(`^"([^"]*)" should not respond with his <DID>$`, noResponseToHelloWithDID)
	ctx.Step(`^"([^"]*)" revokes permission from "([^"]*)" via "([^"]*)"$`, revokesATokenFromVia)
}

func saysHelloto(ctx context.Context, greeterName, receiverName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	greeter, greeterDmsCtx := utils.NodeWithDMS(nodes, greeterName)
	assert.NotNil(t, greeter)
	assert.NotNil(t, greeterDmsCtx)

	receiver, receiverDmsCtx := utils.NodeWithDMS(nodes, receiverName)
	assert.NotNil(t, receiver)
	assert.NotNil(t, receiverDmsCtx)

	greeterInfo, err := greeterDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotNil(t, greeterInfo)

	receiverInfo, err := receiverDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotNil(t, receiverInfo)

	receiverAddr, err := utils.MultiaddrFromCLI(receiverInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, receiverAddr)

	err = greeterDmsCtx.Connect(receiverAddr)
	require.NoError(t, err)

	// calling hello with nil receiver so that it's a broadcast.
	// direct invocation can be ambigious between error due to caps or other issues
	// broadcast works reliably since peers are already connected
	response, err := greeterDmsCtx.Hello(nil)
	require.NoError(t, err)

	tc = tc.WithHelloResponse(response)
	return tc.Unwrap(), nil
}

// When "Bob" revokes a token from "Alice" via "nunet"
// receiverName: Bob
// greeterName: Alice
// orgName: nunet
func revokesATokenFromVia(ctx context.Context, receiverName, greeterName, orgName string) (context.Context, error) {
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

	_, receiverCtx := utils.GetNodeAndContext(nodes, receiverName)
	assert.NotEmpty(t, receiverCtx)

	_, receiverDmsCtx := utils.NodeWithDMS(nodes, receiverName)
	assert.NotEmpty(t, receiverDmsCtx)

	token, found := tokenMap[strings.ToLower(greeterName)]
	assert.True(t, found)

	err = utils.RevokeFromPrivateNetwork(token, receiverCtx, receiverDmsCtx, orgCtx)
	require.NoError(t, err)

	time.Sleep(10 * time.Second)
	return tc.Unwrap(), nil
}

func respondsToHelloWithDID(ctx context.Context, expectedResponder string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	responder, responderDmsCtx := utils.NodeWithDMS(nodes, expectedResponder)
	assert.NotNil(t, responder)
	assert.NotNil(t, responderDmsCtx)

	responseDIDs, err := tc.HelloResponse()
	assert.NoError(t, err)
	assert.NotNil(t, responseDIDs, "no response DIDs")

	if slices.Contains(responseDIDs, responderDmsCtx.DID) {
		return nil
	}

	return fmt.Errorf("expected DID not found in hello response")
}

func noResponseToHelloWithDID(ctx context.Context, expectedResponder string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	responder, responderDmsCtx := utils.NodeWithDMS(nodes, expectedResponder)
	assert.NotNil(t, responder)
	assert.NotNil(t, responderDmsCtx)

	responseDIDs, err := tc.HelloResponse()
	assert.NoError(t, err)
	assert.NotNil(t, responseDIDs, "no response DIDs")

	if !slices.Contains(responseDIDs, responderDmsCtx.DID) {
		return nil
	}

	return fmt.Errorf("DID found in hello response")
}
