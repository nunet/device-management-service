//go:build acceptance || !unit

package acceptance

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
)

type (
	nodesCtxKey    struct{}
	nodeMapCtxKey  struct{}
	ensembleCtxKey struct{}
)

var opts = godog.Options{
	Output:        colors.Colored(os.Stdout),
	Concurrency:   4,
	Format:        "pretty",
	StopOnFailure: true,
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opts)
}

func TestFeatures(t *testing.T) {
	t.Parallel()

	o := opts
	o.TestingT = t

	suite := godog.TestSuite{
		Name:                "main",
		Options:             &o,
		ScenarioInitializer: InitializeScenario,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func hasDeployedDockerHelloOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)

	nodes, ok := ctx.Value(nodesCtxKey{}).([]*utils.Node)
	assert.True(t, ok)

	spName = strings.ToLower(spName)
	cpName = strings.ToLower(cpName)
	orgName := "org"

	nodeMap := map[string]*utils.Node{
		spName:  nodes[0],
		cpName:  nodes[1],
		orgName: nodes[2],
	}

	sp := nodeMap[spName]
	cp := nodeMap[cpName]
	org := nodeMap[orgName]

	ctx = context.WithValue(ctx, nodeMapCtxKey{}, nodeMap)

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

	orgCtx, err := org.CreateContext("org")
	assert.NoError(t, err, "could not create org")
	assert.NotNil(t, orgCtx)

	// update nodeMap with the latest node objects that have updated contexts
	nodeMap = map[string]*utils.Node{
		spName:  sp,
		cpName:  cp,
		orgName: org,
	}
	// update ctx after nodes have been updated with capability contexts
	ctx = context.WithValue(ctx, nodeMapCtxKey{}, nodeMap)

	spGrant, err := spUserCtx.Grant(orgCtx.DID)
	assert.NoError(t, err, "sp failed to grant org", spGrant)
	assert.NotEmpty(t, spGrant)

	cpGrant, err := cpUserCtx.Grant(orgCtx.DID)
	assert.NoError(t, err, "cp failed to grant org", cpGrant)
	assert.NotEmpty(t, cpGrant)

	orgGrantToAlice, err := orgCtx.Grant(spUserCtx.DID)
	assert.NoError(t, err, "org failed to grant sp", spGrant)
	assert.NotEmpty(t, orgGrantToAlice)

	orgGrantToBob, err := orgCtx.Grant(cpUserCtx.DID)
	assert.NoError(t, err, "org failed to grant cp", spGrant)
	assert.NotEmpty(t, orgGrantToBob)

	err = spUserCtx.JoinOrg(spDmsCtx, orgCtx.DID, orgGrantToAlice)
	assert.NoError(t, err, "sp could not join the org")

	err = cpUserCtx.JoinOrg(cpDmsCtx, orgCtx.DID, orgGrantToBob)
	assert.NoError(t, err, "cp could not join the org")

	err = spDmsCtx.Run()
	assert.NoError(t, err)

	// wait for dms to start
	require.Eventually(t, func() bool {
		out, err := sp.RunCMD([]string{"ss", "-tnlp"})
		assert.NoError(t, err)
		return strings.Contains(out, ":9999")
	}, 20*time.Second, 500*time.Millisecond)

	// check if Docker was installed successfully
	assert.NoError(t, g.Wait())

	err = cpDmsCtx.Run()
	assert.NoError(t, err)

	// wait for dms to start
	require.Eventually(t, func() bool {
		out, err := cp.RunCMD([]string{"ss", "-tnlp"})
		assert.NoError(t, err)
		return strings.Contains(out, ":9999")
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

	// upload ensemble configuration to VM
	here := dutils.CurrentFileDirectory()
	localEnsemble := filepath.Join(here, "..", "examples", "docker_hello.yaml")
	spEnsemble := "/root/docker_hello.yaml"

	err = sp.UploadFile(localEnsemble, spEnsemble, 0o755)
	assert.NoError(t, err)

	// update the ensemble configuration to specify compute provider peer ID
	updateCmd := fmt.Sprintf("sed -i 's/failure_recovery: stay_down/failure_recovery: stay_down\\n        peer: %s/' %s",
		cpInfo.ID, spEnsemble)
	_, err = sp.RunCMD([]string{"sh", "-c", updateCmd})
	assert.NoError(t, err)

	ensembleID, err := spDmsCtx.Deploy(spEnsemble)
	assert.NoError(t, err)

	ctx = context.WithValue(ctx, ensembleCtxKey{}, ensembleID)
	return ctx, nil
}

func deploymentIsCompleted(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)

	nodeMap, ok := ctx.Value(nodeMapCtxKey{}).(map[string]*utils.Node)
	assert.True(t, ok)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, ok := ctx.Value(ensembleCtxKey{}).(string)
	assert.True(t, ok)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
		assert.True(t, ok)
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		assert.NoError(t, err)
		return status == "Completed"
	}, 20*time.Second, 1*time.Second)

	return ctx, nil
}

func ensembleShouldReturn(ctx context.Context, spName, expected string) error {
	t := godog.T(ctx)

	nodeMap, ok := ctx.Value(nodeMapCtxKey{}).(map[string]*utils.Node)
	assert.True(t, ok)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, ok := ctx.Value(ensembleCtxKey{}).(string)
	assert.True(t, ok)
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

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		config, err := config.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch config: %w", err)
		}

		clients, err := utils.ConnectToClients(config)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to incus clients: %w", err)
		}

		// cleanup all remaining instances if any
		start := time.Now()
		fmt.Println("cleaning up instances")
		for _, c := range clients {
			instances, err := utils.ListInstances(c)
			if err != nil {
				return nil, fmt.Errorf("could not list instances: %w", err)
			}
			if len(instances) == 0 {
				continue
			}
			for _, i := range instances {
				err := utils.DeleteInstance(c, i.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to delete instance %s: %w", i.Name, err)
				}
			}
		}
		fmt.Printf("finished cleaning up. time elapsed: %.1fs\n", time.Since(start).Seconds())

		start = time.Now()
		fmt.Println("creating nodes...")
		nodes, err := utils.CreateNodes(clients, 3, utils.DefaultImage, utils.DefaultVMPrefix)
		if err != nil {
			return nil, err
		}

		here := dutils.CurrentFileDirectory()
		remoteDMSPath := "/usr/local/bin/nunet"
		localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")
		for idx, n := range nodes {
			err := n.UploadFile(localPath, remoteDMSPath, 0o755)
			if err != nil {
				return nil, fmt.Errorf("failed to upload file to node %d: %w", idx, err)
			}

			_, err = n.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"})
			if err != nil {
				return nil, fmt.Errorf("failed to make dms executable at node %d: %w", idx, err)
			}
		}

		fmt.Printf("finished setting up nodes, time elapsed: %.1fs\n", time.Since(start).Seconds())
		ctx = context.WithValue(ctx, nodesCtxKey{}, nodes)
		ctx = context.WithValue(ctx, ensembleCtxKey{}, "")
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		fmt.Println("test finished. destroying machines...")
		start := time.Now()

		nodes, _ := ctx.Value(nodesCtxKey{}).([]*utils.Node)
		g := new(errgroup.Group)
		for _, n := range nodes {
			g.Go(func() error {
				return n.Destroy()
			})
		}
		if err := g.Wait(); err != nil {
			return ctx, fmt.Errorf("failed to destroy: %w", err)
		}

		fmt.Printf("teardown done! time elapsed: %.1fs\n", time.Since(start).Seconds())
		ctx = context.WithValue(ctx, nodesCtxKey{}, nil)
		return ctx, nil
	})

	ctx.Step(`^"([^"]*)" has deployed docker_hello\.yaml on "([^"]*)"$`, hasDeployedDockerHelloOn)
	ctx.Step(`^"([^"]*)" deployment is completed$`, deploymentIsCompleted)
	ctx.Step(`^"([^"]*)" ensemble should return "([^"]*)"$`, ensembleShouldReturn)
}
