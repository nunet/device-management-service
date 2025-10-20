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

type tableNode struct {
	name      string
	role      string
	onboarded bool
	org       string
}

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

	// get all unique organizations, since all nodes
	// on the table may be assigned to the same org
	orgs := []string{}
	for _, node := range nodes {
		if !slices.Contains(orgs, node.org) {
			orgs = append(orgs, node.org)
		}
	}

	// now create instances based on the total
	// amount of nodes
	total := len(nodes)
	instances, err := hooks.SetupNodes(total)
	require.NoError(t, err)
	assert.NotEmpty(t, instances)

	assert.Len(t, instances, total)

	// map nodes to unique instances
	nodeToInstance := make(map[string]*utils.Node)
	for i, node := range nodes {
		instance := instances[i]
		nodeToInstance[node.name] = instance
	}

	orgMap := make(map[string]*utils.Context)
	allInstances := slices.Collect(maps.Keys(nodeToInstance))

	for _, org := range orgs {
		i := rand.Intn(len(allInstances))
		instance := instances[i]

		orgCtx, err := instance.CreateContext(org)
		require.NoError(t, err)

		orgMap[org] = orgCtx
	}

	tc = tc.WithNodeMap(nodeToInstance)

	g := new(errgroup.Group)

	// all setup for nodes (sp/cp)
	for _, node := range nodes {
		g.Go(func() error {
			instance, ok := nodeToInstance[node.name]
			if !ok {
				return fmt.Errorf("instance for node %s not found", node.name)
			}

			if err := instance.PruneResolved(); err != nil {
				return err
			}

			userCtx, dmsCtx, err := instance.InitialCaps(node.name)
			if err != nil {
				return err
			}
			assert.NotNil(t, userCtx)
			assert.NotNil(t, dmsCtx)

			orgCtx, ok := orgMap[node.org]
			if !ok {
				return fmt.Errorf("org context for %s not found", node.org)
			}

			if err := utils.SetupPrivateNetwork(userCtx, dmsCtx, orgCtx); err != nil {
				return err
			}

			if err := dmsCtx.Run(t); err != nil {
				return err
			}

			assert.Eventually(t, func() bool {
				return instance.IsDMSRunning(9999)
			}, 20*time.Second, 500*time.Millisecond)

			if node.onboarded {
				if err := dmsCtx.Onboard(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	err = g.Wait()
	require.NoError(t, err)

	return tc.Unwrap(), nil
}

func parseNodesTable(table *godog.Table) ([]tableNode, error) {
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

	var nodes []tableNode
	for i := 1; i < len(table.Rows); i++ {
		row := table.Rows[i]

		onboarded, err := strconv.ParseBool(row.Cells[2].Value)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid onboarded value '%s': %v", i, row.Cells[2].Value, err)
		}

		u := tableNode{
			name:      strings.ToLower(row.Cells[0].Value),
			role:      strings.ToLower(row.Cells[1].Value),
			onboarded: onboarded,
			org:       strings.ToLower(row.Cells[3].Value),
		}

		nodes = append(nodes, u)
	}

	return nodes, nil
}
