package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	"gitlab.com/nunet/device-management-service/types"
	"golang.org/x/sync/errgroup"
)

// Subnet registers all step definitions for subnet feature
func Subnet(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return hooks.SetupNodes(ctx, 4)
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		err := hooks.SaveLogs(ctx)
		if err != nil {
			return ctx, err
		}
		return hooks.TeardownNodes(ctx)
	})

	ctx.Step(`^"([^"]*)" has services deployed on "([^"]*)" and "([^"]*)"$`, hasServicesDeployedOn)
	ctx.Step(`^"([^"]*)" service tries to communicate with "([^"]*)"$`, serviceTriesToCommunicateWith)
	ctx.Step(`they should get a OK response$`, shouldGetAOKResponse)
}

func hasServicesDeployedOn(ctx context.Context, spName, cpName, otherCPName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)
	otherCPName = strings.ToLower(otherCPName)

	nodeMap := map[string]*utils.Node{
		spName:      nodes[0],
		cpName:      nodes[1],
		otherCPName: nodes[2],
		orgName:     nodes[3],
	}

	sp := nodeMap[spName]
	cp := nodeMap[cpName]
	otherCP := nodeMap[otherCPName]
	org := nodeMap[orgName]

	// TODO: Put it as a hook
	err = cp.PruneResolved()
	assert.NoError(t, err)

	err = otherCP.PruneResolved()
	assert.NoError(t, err)

	t.Log("got with pruning resolved")

	tc = tc.WithNodeMap(nodeMap)

	// only compute providers needs Docker
	// launch goroutine while setting up capabilities
	// docker should be available before DMS starts
	t.Log("put docker to install")

	g := new(errgroup.Group)
	g.Go(func() error {
		return cp.InstallDocker()
	})

	g.Go(func() error {
		return otherCP.InstallDocker()
	})

	spUserCtx, spDmsCtx, err := sp.InitialCaps(spName)
	assert.NoError(t, err)
	assert.NotNil(t, spUserCtx)
	assert.NotNil(t, spDmsCtx)

	cpUserCtx, cpDmsCtx, err := cp.InitialCaps(cpName)
	assert.NoError(t, err)
	assert.NotNil(t, cpUserCtx)
	assert.NotNil(t, cpDmsCtx)

	otherCPUserCtx, otherCPDmsCtx, err := otherCP.InitialCaps(otherCPName)
	assert.NoError(t, err)
	assert.NotNil(t, otherCPUserCtx)
	assert.NotNil(t, otherCPDmsCtx)

	orgCtx, err := org.CreateContext(orgName)
	assert.NoError(t, err, "could not create org")
	assert.NotNil(t, orgCtx)

	// update nodeMap with the latest node objects that have updated contexts
	nodeMap = map[string]*utils.Node{
		spName:      sp,
		cpName:      cp,
		otherCPName: otherCP,
		orgName:     org,
	}
	// update ctx after nodes have been updated with capability contexts
	tc = tc.WithNodeMap(nodeMap)

	err = utils.SetupPrivateNetwork(spUserCtx, spDmsCtx, orgCtx)
	assert.NoError(t, err)

	err = utils.SetupPrivateNetwork(cpUserCtx, cpDmsCtx, orgCtx)
	assert.NoError(t, err)

	err = utils.SetupPrivateNetwork(otherCPUserCtx, otherCPDmsCtx, orgCtx)
	assert.NoError(t, err)

	t.Log("got private network")

	err = spDmsCtx.Run()
	assert.NoError(t, err)

	// wait for dms to start on sp
	require.Eventually(t, func() bool {
		return sp.IsDMSRunning(9999)
	}, 20*time.Second, 500*time.Millisecond)

	assert.NoError(t, g.Wait())
	t.Log("docker finished")

	err = cpDmsCtx.Run()
	assert.NoError(t, err)

	err = otherCPDmsCtx.Run()
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		return cp.IsDMSRunning(9999)
	}, 20*time.Second, 500*time.Millisecond)

	require.Eventually(t, func() bool {
		return otherCP.IsDMSRunning(9999)
	}, 20*time.Second, 500*time.Millisecond)

	spInfo, err := spDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, spInfo)

	cpInfo, err := cpDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, cpInfo)

	otherCPInfo, err := otherCPDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, otherCPInfo)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, cpAddr)

	otherCPAddr, err := utils.MultiaddrFromCLI(otherCPInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, otherCPAddr)

	err = spDmsCtx.Connect(cpAddr)
	assert.NoError(t, err)

	err = spDmsCtx.Connect(otherCPAddr)
	assert.NoError(t, err)

	err = cpDmsCtx.Onboard()
	assert.NoError(t, err)

	err = otherCPDmsCtx.Onboard()
	assert.NoError(t, err)

	file := utils.FindTestdata("ensembles/multiple_nginx.yaml")
	ensemble, err := utils.UploadEnsemble(sp, file)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	ensembleID, err := spDmsCtx.Deploy(ensemble)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	tc = tc.WithEnsembleID(ensembleID)

	require.Eventually(t, func() bool {
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return status == "Running"
	}, 60*time.Second, 1*time.Second)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)

	tc = tc.WithManifest(manifest)

	return tc.Unwrap(), nil
}

func serviceTriesToCommunicateWith(ctx context.Context, cpName, otherCPName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	cpName = strings.ToLower(cpName)
	otherCPName = strings.ToLower(otherCPName)

	cp := nodeMap[cpName]
	otherCP := nodeMap[otherCPName]

	manifest, err := tc.Manifest()
	assert.NoError(t, err)
	assert.NotNil(t, manifest)

	cpDmsCtx := cp.Contexts[cpName+utils.DefaultDMSSuffix]
	otherCPDmsCtx := otherCP.Contexts[otherCPName+utils.DefaultDMSSuffix]

	cpAllocs, err := cpDmsCtx.AllocationList()
	assert.NoError(t, err)
	assert.NotEmpty(t, cpAllocs)

	otherCPAllocs, err := otherCPDmsCtx.AllocationList()
	assert.NoError(t, err)
	assert.NotEmpty(t, otherCPAllocs)

	allocMap := make(map[string][]jobs.AllocationInfo)

	allocMap[cpName] = cpAllocs
	allocMap[otherCPName] = otherCPAllocs

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)

	type execution struct {
		dns         string
		publicPorts []int
		executionID string
		alloc       string
		node        *utils.Node
	}
	var execs []execution
	for owner, allocs := range allocMap {
		for _, alloc := range allocs {
			if ensembleID != types.EnsembleIDFromAllocationID(alloc.ID) ||
				alloc.ExecutionID == "" {
				continue
			}

			name := types.AllocationNameFromID(alloc.ID)

			info, ok := manifest.Allocations[name]
			assert.True(t, ok)

			var ports []int
			for public := range info.Ports {
				ports = append(ports, public)
			}

			node, ok := nodeMap[owner]
			if !ok {
				continue
			}
			exec := execution{
				dns:         info.DNSName,
				publicPorts: ports,
				executionID: alloc.ExecutionID,
				alloc:       name,
				node:        node,
			}
			execs = append(execs, exec)
		}
	}
	assert.NotEmpty(t, execs)

	var responses []string
	for _, client := range execs {
		for _, server := range execs {
			if client.alloc == server.alloc {
				continue
			}

			if len(server.publicPorts) == 0 {
				continue
			}

			portStr := fmt.Sprintf(":%d", server.publicPorts[0])
			cmd := []string{
				"docker", "exec", client.executionID,
				"curl", "-s", "-o", "/dev/null",
				"-w", "'%{http_code}'",
				"-m", "5", // 5 second timeout
				"http://" + server.dns + portStr,
			}
			out, err := client.node.RunCMD(cmd)
			assert.NoError(t, err)
			assert.NotEmpty(t, out)

			codeQuotes := strings.TrimSpace(out)
			httpCode := strings.Trim(codeQuotes, `'`)
			assert.Equal(t, "200", httpCode)

			responses = append(responses, httpCode)
		}
	}
	assert.NotEmpty(t, responses)
	tc = tc.WithAllocationResponses(responses)

	return tc.Unwrap(), nil
}

func shouldGetAOKResponse(ctx context.Context) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	responses, err := tc.AllocationResponses()
	assert.NoError(t, err)
	assert.NotEmpty(t, responses)

	for _, resp := range responses {
		assert.Equal(t, "200", resp)
	}

	return nil
}
