package steps

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	"golang.org/x/sync/errgroup"
)

const orgName = "org"

// Deployment registers all step definitions for deployment feature
func Deployment(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return hooks.SetupNodes(ctx, 3)
	})
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		err := hooks.SaveLogs(ctx)
		if err != nil {
			return ctx, err
		}
		return hooks.TeardownNodes(ctx)
	})

	ctx.Step(`^"([^"]*)" has deployed docker_hello\.yaml on "([^"]*)"$`, hasDeployedDockerHelloOn)
	ctx.Step(`^"([^"]*)" deployment is completed$`, deploymentIsCompleted)
	ctx.Step(`^"([^"]*)" ensemble should return "([^"]*)"$`, ensembleShouldReturn)
}

func hasDeployedDockerHelloOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)

	nodeMap := map[string]*utils.Node{
		spName:  nodes[0],
		cpName:  nodes[1],
		orgName: nodes[2],
	}

	sp := nodeMap[spName]
	cp := nodeMap[cpName]
	org := nodeMap[orgName]

	tc = tc.WithNodeMap(nodeMap)

	// only Bob (compute provider) needs Docker
	// launch goroutine while setting up capabilities
	// docker should be available before DMS starts
	g := new(errgroup.Group)
	g.Go(func() error {
		return cp.InstallDocker()
	})

	spUserCtx, spDmsCtx, err := sp.InitialCaps(spName)
	assert.NoError(t, err)
	assert.NotNil(t, spUserCtx)
	assert.NotNil(t, spDmsCtx)

	cpUserCtx, cpDmsCtx, err := cp.InitialCaps(cpName)
	assert.NoError(t, err)
	assert.NotNil(t, cpUserCtx)
	assert.NotNil(t, cpDmsCtx)

	orgCtx, err := org.CreateContext(orgName)
	assert.NoError(t, err, "could not create org")
	assert.NotNil(t, orgCtx)

	// update nodeMap with the latest node objects that have updated contexts
	nodeMap = map[string]*utils.Node{
		spName:  sp,
		cpName:  cp,
		orgName: org,
	}
	// update ctx after nodes have been updated with capability contexts
	tc = tc.WithNodeMap(nodeMap)

	err = utils.SetupPrivateNetwork(spUserCtx, spDmsCtx, orgCtx)
	assert.NoError(t, err)

	err = utils.SetupPrivateNetwork(cpUserCtx, cpDmsCtx, orgCtx)
	assert.NoError(t, err)

	err = spDmsCtx.Run()
	assert.NoError(t, err)

	// wait for dms to start on sp
	require.Eventually(t, func() bool {
		return sp.IsDMSRunning(9999)
	}, 20*time.Second, 500*time.Millisecond)

	// check if Docker was installed successfully
	assert.NoError(t, g.Wait())

	err = cpDmsCtx.Run()
	assert.NoError(t, err)

	// wait for dms to start on cp
	require.Eventually(t, func() bool {
		return cp.IsDMSRunning(9999)
	}, 20*time.Second, 500*time.Millisecond)

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

	file := utils.FindTestdata("ensembles/docker_hello.yaml")
	ensemble, err := utils.UploadEnsemble(sp, file)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensemble)

	ensembleID, err := spDmsCtx.Deploy(ensemble)
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	tc = tc.WithEnsembleID(ensembleID)
	return tc.Unwrap(), nil
}

func deploymentIsCompleted(ctx context.Context, spName string) (context.Context, error) {
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

	require.Eventually(t, func() bool {
		spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
		assert.True(t, ok)
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return status == "Completed"
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
