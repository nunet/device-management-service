package itest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	cmd "gitlab.com/nunet/device-management-service/cmd/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

func DeploymentTest(suite *TestSuite) {
	suite.Run("must be able to deploy nginx demo with storage", func() {
		// deploy nginx.yaml to node1 using node2's orchestrator
		node1 := suite.nodes[0]
		hostname, err := os.Hostname()
		suite.Require().NoError(err)
		srcFile := filepath.Join(suite.testDataDir, "ensembles", "nginx-storage.yaml")
		destinationFile := filepath.Join(node1.config.WorkDir, "nginx-storage.yaml")
		err = copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)
		err = replaceHostnameInFile(destinationFile, hostname)
		suite.Require().NoError(err)

		freeResourcesBefore, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)

		// Deploy nginx.yaml from node2's orchestrator.
		node2 := suite.nodes[1]
		deploymentResult := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(node1.config.WorkDir, "nginx-storage.yaml"))
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.userContext, node2.password, manifestID)
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")
		time.Sleep(2 * time.Second)

		// Ensure resources are allocated.
		freeResourcesDuring, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring.Equal(freeResourcesBefore))

		// TODO: check port mapping

		// Shutdown the nginx deployment.
		shutdownRes := node2.client.shutdownDeployment(suite.T(), node2.userContext, node2.password, manifestID)
		suite.Contains(shutdownRes, `"Error": ""`)

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.dmsContext, node2.password, manifestID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")

		// Ensure resources are freed.
		freeResourcesAfter, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter.Equal(freeResourcesBefore))
	})

	suite.Run("allocation of type TASK: deploy docker hello-world", func() {
		node2 := suite.nodes[1]
		deployment2Result := node2.client.deploy(
			suite.T(), node2.userContext, node2.password,
			filepath.Join(suite.testDataDir, "ensembles", "hello.yaml"),
		)
		suite.Contains(deployment2Result, `"Status": "OK"`)
		manifestID := extractEnsembleID(deployment2Result)

		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.userContext, node2.password, manifestID)
			suite.T().Log("Second deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Hello-world deployment did not reach Running status")

		// Shutdown the hello-world deployment.
		shutdownRes := node2.client.shutdownDeployment(suite.T(), node2.dmsContext, node2.password, manifestID)
		suite.Require().Contains(shutdownRes, `"Error": ""`)

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.dmsContext, node2.password, manifestID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
	})
}

// DeploymentFullAssertion deploys multiple_nginx.yaml using 4 nodes:
// 1 deployer and 3 providers (bob, alice, carl).
//
// - All allocations are services (TODO: test with a task?)
// - One of the nodes will have two allocations.
//
// What is asserted here?
// - Allocations are running (including check of executions)
// - Subnet conns between peers
// - Resources allocation (before and after deployment)
// - Manifest changes
func DeploymentFullAssertion(suite *TestSuite) {
	suite.Require().Len(suite.nodes, 4)
	deployer := suite.nodes[0]
	bobProvider := suite.nodes[1]
	aliceProvider := suite.nodes[2]
	carlProvider := suite.nodes[3]

	// 1. ensure all nodes have free resources and no allocations running before anything
	for _, node := range suite.nodes {
		suite.assertFreeResourcesFull(node)
		suite.assertNoAllocationsRunning(node)
	}

	ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "multiple_nginx.yaml")

	// process ensemble cfg as a helper for later assertions
	r, err := cmd.ProcessEnsembleYaml(afero.Afero{Fs: afero.NewOsFs()}, ensemblePath)
	suite.Require().NoError(err)
	ensembleCfg := r.Ensemble

	// 2. start deployment
	deploymentResult := deployer.client.deploy(
		suite.T(), deployer.userContext,
		deployer.password,
		ensemblePath,
	)
	suite.Contains(deploymentResult, `"Status": "OK"`)
	ensembleID := extractEnsembleID(deploymentResult)

	executions := []executionInfo{}

	// 4. Shutdown: assert if everything was freed
	defer func() {
		shutdownRes := deployer.client.shutdownDeployment(
			suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Contains(shutdownRes, `"Error": ""`)

		suite.Require().Eventually(func() bool {
			status := deployer.client.deploymentStatus(
				suite.T(), deployer.dmsContext, deployer.password, ensembleID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
		time.Sleep(3 * time.Second)

		// resources freed and allocations stopped
		for _, node := range suite.nodes {
			suite.assertFreeResourcesFull(node)
			suite.assertNoAllocationsRunning(node, executions...)
		}
	}()

	// Wait until the deployment status is "Running".
	suite.Require().Eventually(func() bool {
		status := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.T().Log("Deployment status:", extractStatus(status))
		return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
	}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")
	time.Sleep(2 * time.Second)

	//  3. assert running phase:
	providers := []*mockNode{bobProvider, aliceProvider, carlProvider}
	suite.assertRunningPhase(ensembleCfg, ensembleID, deployer, providers)

	// track executions IDs so that we cna check if it was shutdown
	for _, provider := range providers {
		executions = append(executions, suite.getExecutions(provider)...)
	}
}

//  3. assert running phase:
//     3.1. allocations are running on the correct providers
//     3.2. resources are allocated correctly
//     3.3. manifest
//     3.4. subnet conns between containers
func (suite *TestSuite) assertRunningPhase(
	ensembleCfg jobtypes.EnsembleConfig, ensembleID string,
	deployer *mockNode, providers []*mockNode,
) {
	nodeToProvider := suite.getNodeProviderMapping(deployer, ensembleID, providers)

	// 3.1. allocations are running on the correct providers
	for nodeName, nodeConfig := range ensembleCfg.Nodes() {
		provider, ok := nodeToProvider[nodeName]
		suite.Require().True(ok, "No provider found for ensemble node %s", nodeName)

		suite.assertAllocationsRunning(provider, ensembleID, nodeConfig.Allocations)
	}

	// 3.2. resources are allocated correctly
	// All allocations use the same resources from nginx_wrapper
	nginxWrapper := jobtypes.AllocationConfig{}
	for _, alloc := range ensembleCfg.Allocations() {
		nginxWrapper = alloc
		break
	}
	suite.Require().NotEmpty(nginxWrapper.Resources)

	for nodeName, nodeConfig := range ensembleCfg.Nodes() {
		provider := nodeToProvider[nodeName]

		resources := types.Resources{}
		for range len(nodeConfig.Allocations) {
			err := resources.Add(nginxWrapper.Resources)
			suite.Assert().NoError(err)
		}

		suite.assertResourcesAfterDeployment(provider, resources)
	}

	// 3.3. manifest
	suite.assertManifestAfterDeployment(deployer, providers, ensembleCfg, ensembleID)

	// 3.4. subnet conns between containers
	suite.assertSubnetConns(ensembleID, ensembleCfg, deployer, providers)
}

func (suite *TestSuite) getExecutions(node *mockNode) []executionInfo {
	executions := []executionInfo{}

	allocations, err := node.client.allocationsList(
		node.userContext, node.password)
	suite.Require().NoError(err)

	for _, allocation := range allocations {
		if allocation.ExecutionID != "" {
			executions = append(executions, executionInfo{
				executor: allocation.Executor,
				id:       allocation.ExecutionID,
			})
		}
	}
	return executions
}

func (suite *TestSuite) getNodeProviderMapping(
	deployer *mockNode, ensembleID string, providers []*mockNode,
) map[string]*mockNode {
	manifest, err := deployer.client.deploymentManifest(
		deployer.userContext, deployer.password, ensembleID)
	suite.Require().NoError(err)

	nodeToProvider := make(map[string]*mockNode)
	for nodeName, nodeManifest := range manifest.Nodes {
		for _, provider := range providers {
			if nodeManifest.Peer == provider.peerID {
				nodeToProvider[nodeName] = provider
				break
			}
		}
	}
	suite.Require().Len(nodeToProvider, len(providers))

	return nodeToProvider
}

// assertSubnetConns asserts that allocations are within the same subnet
// and reachable.
//
// How: it permutates between all pairs of allocations (excluding self but
// including allocs witin the same node), and emit curl requests to the nginx
// containers
func (suite *TestSuite) assertSubnetConns(
	ensembleID string, ensembleCfg jobtypes.EnsembleConfig,
	deployer *mockNode, providers []*mockNode,
) {
	manifest, err := deployer.client.deploymentManifest(
		deployer.userContext, deployer.password, ensembleID)
	suite.Require().NoError(err)

	// Get the mapping from node names to providers
	nodeProviders := suite.getNodeProviderMapping(deployer, ensembleID, providers)

	// gather all executions information so that we can permutate later
	allDeployedExecutions := make([]allocExecution, 0, len(ensembleCfg.Allocations()))
	for node := range manifest.Nodes {
		provider, ok := nodeProviders[node]
		suite.Require().True(ok, "Provider not found for node: %s", node)
		suite.Require().NotNil(provider, "Provider not found for node: %s", node)

		allocsExecutions := suite.getRunningExecutions(ensembleCfg, provider, node, ensembleID)
		if len(allocsExecutions) == 0 {
			suite.Assert().True(true, "No allocations running for node: %s", node)
			continue
		}

		allDeployedExecutions = append(allDeployedExecutions, allocsExecutions...)
	}

	for _, client := range allDeployedExecutions {
		for _, server := range allDeployedExecutions {
			// don't skip inter-node conn, unless it's the same allocation
			if client.alloc == server.alloc {
				continue
			}

			// Test connectivity from client to server
			err := curlExecution(suite.T(), client, server)
			suite.Assert().NoError(err, "Failed to connect from %s to %s", client.alloc, server.alloc)
		}
	}
}

type allocExecution struct {
	execution   executionInfo
	dnsName     string
	publicPorts []int
	alloc       string
	node        string
}

// getRunningExecutions returns a list of running executions for a provider
func (suite *TestSuite) getRunningExecutions(
	cfg jobtypes.EnsembleConfig, provider *mockNode,
	nodeName, ensembleID string,
) []allocExecution {
	allocations, err := provider.client.allocationsList(
		provider.userContext, provider.password)
	suite.Require().NoError(err)

	executions := make([]allocExecution, 0, len(allocations))
	for _, alloc := range allocations {
		if ensembleID == types.EnsembleIDFromAllocationID(alloc.ID) &&
			alloc.ExecutionID != "" {
			allocName := types.AllocationNameFromID(alloc.ID)

			// dns
			allocConfig, ok := cfg.Allocation(allocName)
			if !ok {
				suite.Assert().True(true, "Allocation not found: %s", allocName)
				continue
			}

			// ports
			ports := cfg.PortsForAllocation(allocName)
			publicPorts := make([]int, 0, len(ports))
			for _, port := range ports {
				publicPorts = append(publicPorts, port.Public)
			}

			executions = append(executions, allocExecution{
				execution: executionInfo{
					executor: alloc.Executor,
					id:       alloc.ExecutionID,
				},
				dnsName:     allocConfig.DNSName,
				publicPorts: publicPorts,
				alloc:       allocName,
				node:        nodeName,
			})
		}
	}

	return executions
}

func curlExecution(t *testing.T, client, server allocExecution) error {
	t.Helper()
	if len(server.publicPorts) == 0 {
		return fmt.Errorf("server %s has no public ports", server.alloc)
	}
	portStr := fmt.Sprintf(":%d", server.publicPorts[0])
	curlCmd := []string{
		"docker", "exec", client.execution.id,
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"-m", "5", // 5 second timeout
		"http://" + server.dnsName + portStr,
	}

	// Execute the command
	output, err := exec.Command(curlCmd[0], curlCmd[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("curl from %s to %s failed: %w, output: %s",
			client.alloc, server.alloc, err, string(output))
	}

	// Check if we got HTTP 200
	httpCode := strings.TrimSpace(string(output))
	if httpCode != "200" {
		return fmt.Errorf("curl from %s to %s returned HTTP %s, expected 200",
			client.alloc, server.alloc, httpCode)
	}

	return nil
}
