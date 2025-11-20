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
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	"golang.org/x/sync/errgroup"
)

// Step designed to be called as Background.
// It parses the data table (format/headers can be found in `parseNodesTable`)
// and provision nodes automatically, as well as their necessary setup based
// on their role (SP, CP)
// The setup includes creating keys, capability contexts, installing dependencies,
// setting up private network and, finally, running DMS
func theFollowingNodes(ctx context.Context, table *godog.Table) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := parseNodesTable(table)
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

	// get all unique organizations
	uniqueOrgs := make(map[string]struct{})
	for _, node := range nodes {
		uniqueOrgs[node.Org] = struct{}{}
	}
	orgNames := make([]string, 0, len(uniqueOrgs))
	for orgName := range uniqueOrgs {
		orgNames = append(orgNames, orgName)
	}
	slices.Sort(orgNames)

	// create instances for all nodes
	total := len(nodes)
	instances, err := hooks.SetupInstances(total)
	require.NoError(t, err)
	assert.NotEmpty(t, instances)
	assert.Len(t, instances, total)

	// assign instances to nodes
	for i, node := range nodes {
		node.Instance = instances[i]
	}

	nodeMap := make(map[string]*utils.Node)
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	tc = tc.WithNodes(nodeMap)

	// all setup for orgs
	orgMap := make(map[string]*utils.Context)
	allInstances := slices.Collect(maps.Keys(nodeMap))

	for _, orgName := range orgNames {
		i := rand.Intn(len(allInstances))
		instance := instances[i]

		orgCtx, err := utils.CreateContext(instance, orgName)
		require.NoError(t, err)

		orgMap[orgName] = orgCtx
	}

	tc = tc.WithOrganizationMap(orgMap)

	tokenMap := make(map[string]string)

	g := new(errgroup.Group)

	// all setup for nodes (sp/cp)
	for _, node := range nodes {
		g.Go(func() error {
			instance := node.Instance

			if err := instance.PruneResolved(); err != nil {
				return err
			}

			err = node.InitialCaps()
			if err != nil {
				return err
			}

			// set dms config to test env for fast observable ip fetch
			err = node.DMS().SetConfig("general.env", "test")
			if err != nil {
				return fmt.Errorf("failed to set dms env to test: %w", err)
			}

			orgCtx, ok := orgMap[node.Org]
			if !ok {
				return fmt.Errorf("org context for %s not found", node.Org)
			}

			token, err := utils.SetupPrivateNetwork(node.User(), node.DMS(), orgCtx)
			if err != nil {
				return err
			}
			tokenMap[node.Name] = token

			if err := node.DMS().Run(t); err != nil {
				return err
			}

			assert.Eventually(t, func() bool {
				return instance.IsDMSRunning(9999)
			}, 20*time.Second, 500*time.Millisecond)

			if node.Onboarded {
				if err := node.DMS().Onboard(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	err = g.Wait()

	require.NoError(t, err)

	tc = tc.WithTokenMap(tokenMap)

	return tc.Unwrap(), nil
}

func parseNodesTable(table *godog.Table) ([]*utils.Node, error) {
	if len(table.Rows) < 2 {
		return nil, fmt.Errorf("table must have header and at least one data row")
	}

	expectedHeaders := []string{"nodes", "role", "onboarded", "org"}
	header := table.Rows[0]

	if len(header.Cells) != len(expectedHeaders) {
		return nil, fmt.Errorf("expected %d columns, got %d", len(expectedHeaders), len(header.Cells))
	}

	for i, expected := range expectedHeaders {
		if header.Cells[i].Value != expected {
			return nil, fmt.Errorf("expected header '%s' at column %d, got '%s'", expected, i, header.Cells[i].Value)
		}
	}

	nodes := make([]*utils.Node, 0, len(table.Rows)-1)
	for i := 1; i < len(table.Rows); i++ {
		row := table.Rows[i]

		onboarded, err := strconv.ParseBool(row.Cells[2].Value)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid onboarded value '%s': %v", i, row.Cells[2].Value, err)
		}

		name := strings.ToLower(row.Cells[0].Value)
		role := strings.ToLower(row.Cells[1].Value)
		org := strings.ToLower(row.Cells[3].Value)

		node := utils.NewNode(name, role, org, onboarded, nil)
		nodes = append(nodes, node)
	}

	return nodes, nil
}
