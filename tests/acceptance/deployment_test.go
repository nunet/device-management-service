// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
)

var opts = godog.Options{
	Output:        colors.Colored(os.Stdout),
	Format:        "pretty",
	StopOnFailure: false,
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opts)
}

func TestDeployment(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/deployment.feature"}

	suite := godog.TestSuite{
		Name:                "deployment",
		Options:             &o,
		ScenarioInitializer: steps.Deployment,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func hasDeployedDockerHelloOn(ctx context.Context, spName, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)

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

	tc = tc.WithNodeMap(nodeMap)

	spUserCtx, spDmsCtx, err := sp.InitialCaps(spName)
	require.NoError(t, err)
	assert.NotNil(t, spUserCtx)
	assert.NotNil(t, spDmsCtx)

	cpUserCtx, cpDmsCtx, err := cp.InitialCaps(cpName)
	require.NoError(t, err)
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
	tc = tc.WithNodeMap(nodeMap)

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

	err = spDmsCtx.Run(t)
	require.NoError(t, err)

	// wait for dms to start
	require.Eventually(t, func() bool {
		out, err := sp.RunCMD([]string{"ss", "-tnlp"})
		require.NoError(t, err)
		return strings.Contains(out, ":9999")
	}, 20*time.Second, 500*time.Millisecond)

	err = cpDmsCtx.Run(t)
	require.NoError(t, err)

	// wait for dms to start
	require.Eventually(t, func() bool {
		out, err := cp.RunCMD([]string{"ss", "-tnlp"})
		require.NoError(t, err)
		return strings.Contains(out, ":9999")
	}, 20*time.Second, 500*time.Millisecond)

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

	err = cpDmsCtx.Onboard()
	require.NoError(t, err)

	// upload ensemble configuration to VM
	here := dutils.CurrentFileDirectory()
	localEnsemble := filepath.Join(here, "..", "examples", "docker_hello.yaml")
	spEnsemble := "/root/docker_hello.yaml"

	err = sp.UploadFile(localEnsemble, spEnsemble, 0o755)
	require.NoError(t, err)

	// update the ensemble configuration to specify compute provider peer ID
	updateCmd := fmt.Sprintf("sed -i 's/failure_recovery: stay_down/failure_recovery: stay_down\\n        peer: %s/' %s",
		cpInfo.ID, spEnsemble)
	_, err = sp.RunCMD([]string{"sh", "-c", updateCmd})
	require.NoError(t, err)

	ensembleID, err := spDmsCtx.Deploy(spEnsemble)
	require.NoError(t, err)

	tc = tc.WithEnsembleID(ensembleID)
	return tc.Unwrap(), nil
}

func deploymentIsCompleted(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	require.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	require.Eventually(t, func() bool {
		spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
		assert.True(t, ok)
		status, err := spDmsCtx.EnsembleStatus(ensembleID)
		require.NoError(t, err)
		return status == "Completed"
	}, 20*time.Second, 1*time.Second)

	return tc.Unwrap(), nil
}

func ensembleShouldReturn(ctx context.Context, spName, expected string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	require.NoError(t, err)
	assert.NotEmpty(t, nodeMap)

	spName = strings.ToLower(spName)
	sp := nodeMap[spName]

	ensembleID, err := tc.EnsembleID()
	require.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	spDmsCtx, ok := sp.Contexts[spName+utils.DefaultDMSSuffix]
	assert.True(t, ok)

	manifest, err := spDmsCtx.Manifest(ensembleID)
	require.NoError(t, err)
	assert.NotNil(t, manifest)

	path, err := spDmsCtx.LogsFromAllocation(ensembleID, "node1.alloc1")
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// TODO: keep it consistent on DMS, rename log file as stdout.log instead
	out, err := sp.RunCMD([]string{"cat", filepath.Join(path, "stdout.log")})
	require.NoError(t, err)
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
		nodes, err := utils.CreateNodes(clients, 3, utils.DefaultVMPrefix)
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

		tc := utils.NewTestCtx(ctx)
		tc = tc.WithNodes(nodes)
		tc = tc.WithEnsembleID("")
		return tc.Unwrap(), nil
	})

	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		tc := utils.NewTestCtx(ctx)

		fmt.Println("test finished. destroying machines...")
		start := time.Now()

		nodes, _ := tc.Nodes()
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
		tc = tc.WithNodes(nil)
		return tc.Unwrap(), nil
	})

	ctx.Step(`^"([^"]*)" has deployed docker_hello\.yaml on "([^"]*)"$`, hasDeployedDockerHelloOn)
	ctx.Step(`^"([^"]*)" deployment is completed$`, deploymentIsCompleted)
	ctx.Step(`^"([^"]*)" ensemble should return "([^"]*)"$`, ensembleShouldReturn)
}
