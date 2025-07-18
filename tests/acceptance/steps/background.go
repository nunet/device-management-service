package steps

import (
	"context"
	"fmt"
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
	assert.NoError(t, err)
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
	// amount of nodes and orgs
	total := len(nodes) + len(orgs)
	instances, err := hooks.SetupNodes(total)
	assert.NoError(t, err)
	assert.NotEmpty(t, instances)

	assert.Len(t, instances, total)

	// here we disambiguate the concept of node/org
	// to assign each one of them a unique instance
	// regardless of their role
	allNodes := []string{}
	allNodes = append(allNodes, orgs...)
	for _, node := range nodes {
		allNodes = append(allNodes, node.name)
	}
	assert.Len(t, allNodes, total)

	// map nodes to unique instances
	nodeToInstance := make(map[string]*utils.Node)
	for i, node := range allNodes {
		instance := instances[i]
		nodeToInstance[node] = instance
	}

	tc = tc.WithNodeMap(nodeToInstance)

	// all setup for orgs
	orgMap := make(map[string]*utils.Context)
	for _, org := range orgs {
		instance, ok := nodeToInstance[org]
		assert.True(t, ok)

		orgCtx, err := instance.CreateContext(org)
		assert.NoError(t, err)

		orgMap[org] = orgCtx
	}

	// all setup for nodes (sp/cp)
	for _, node := range nodes {
		instance, ok := nodeToInstance[node.name]
		assert.True(t, ok)

		// only compute providers need docker
		// launch goroutines to install dependencies
		g := new(errgroup.Group)
		if strings.EqualFold(node.role, "cp") {
			g.Go(func() error {
				if err := instance.InstallDocker(); err != nil {
					return err
				}
				if err := instance.PruneResolved(); err != nil {
					return err
				}
				return nil
			})
		}

		userCtx, dmsCtx, err := instance.InitialCaps(node.name)
		assert.NoError(t, err)
		assert.NotNil(t, userCtx)
		assert.NotNil(t, dmsCtx)

		orgCtx, ok := orgMap[node.org]
		assert.True(t, ok)

		err = utils.SetupPrivateNetwork(userCtx, dmsCtx, orgCtx)
		assert.NoError(t, err)

		// wait dependencies before running, otherwise DMS won't
		// recognize Docker
		assert.NoError(t, g.Wait())

		err = dmsCtx.Run()
		assert.NoError(t, err)

		require.Eventually(t, func() bool {
			return instance.IsDMSRunning(9999)
		}, 20*time.Second, 500*time.Millisecond)

		if node.onboarded {
			assert.NoError(t, dmsCtx.Onboard())
		}
	}

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
