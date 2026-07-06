// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	cmd "gitlab.com/nunet/device-management-service/cmd/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/types"
)

func DeploymentTests(suite *TestSuite) {
	suite.Run("allocation of type TASK: deploy docker hello-world", func() {
		deployer := suite.nodes[1]
		deployment2Result := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "ensembles", "hello.yaml"),
			"2m",
		)
		suite.Contains(deployment2Result, `"Status": "OK"`)
		manifestID := extractEnsembleID(deployment2Result)

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Second deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Hello-world deployment did not reach Running status")

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.dmsContext, deployer.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
	})
}

func DeploymentWithRedundancyTest(suite *TestSuite) {
	suite.Run("must be able to deploy nginx with redundancy", func() {
		// deploy nginx.yaml to node1 using deployer's orchestrator
		node1 := suite.nodes[0]
		node2 := suite.nodes[1]
		node3 := suite.nodes[2]
		srcFile := filepath.Join(suite.testDataDir, "ensembles", "nginx-with-redundancy.yaml")

		freeResourcesBefore1, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)

		freeResourcesBefore2, err := node2.client.freeResources(suite.T(), node2.dmsContext, node2.password)
		suite.Require().NoError(err)

		freeResourcesBefore3, err := node3.client.freeResources(suite.T(), node3.dmsContext, node3.password)
		suite.Require().NoError(err)

		// Deploy nginx.yaml from deployer's orchestrator.
		deployer := suite.nodes[3]
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password,
			srcFile,
			"2m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, manifestID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 3*60*time.Second, 5*time.Second, "Deployment did not reach Running status")

		suite.Require().Eventually(func() bool {
			fr1, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
			if err != nil {
				return false
			}
			fr2, err := node2.client.freeResources(suite.T(), node2.dmsContext, node2.password)
			if err != nil {
				return false
			}
			fr3, err := node3.client.freeResources(suite.T(), node3.dmsContext, node3.password)
			if err != nil {
				return false
			}
			return !fr1.Equal(freeResourcesBefore1) || !fr2.Equal(freeResourcesBefore2) || !fr3.Equal(freeResourcesBefore3)
		}, 60*time.Second, 2*time.Second, "resources were not allocated after deployment reached Running")

		// Ensure resources are allocated.
		freeResourcesDuring1, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring1.Equal(freeResourcesBefore1))

		freeResourcesDuring2, err := node2.client.freeResources(suite.T(), node2.dmsContext, node2.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring2.Equal(freeResourcesBefore2))

		freeResourcesDuring3, err := node3.client.freeResources(suite.T(), node3.dmsContext, node3.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring3.Equal(freeResourcesBefore3))

		// TODO: check port mapping

		// Shutdown the nginx deployment.
		shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, manifestID)
		suite.Contains(shutdownRes, `"Error": ""`)

		suite.waitDeploymentCompleted(deployer, deployer.userContext, manifestID, 3*60*time.Second)

		// Ensure resources are freed.
		freeResourcesAfter1, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter1.Equal(freeResourcesBefore1))

		freeResourcesAfter2, err := node2.client.freeResources(suite.T(), node2.dmsContext, node2.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter2.Equal(freeResourcesBefore2))

		freeResourcesAfter3, err := node3.client.freeResources(suite.T(), node3.dmsContext, node3.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter3.Equal(freeResourcesBefore3))
	})
}

// DeploymentAssertSubnet deploys multiple_nginx.yaml using 4 nodes:
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
func DeploymentAssertSubnet(suite *TestSuite) {
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
	ensembleCfg, err := cmd.ProcessEnsembleYaml(afero.Afero{Fs: afero.NewOsFs()}, env.NewOSEnvironment(), ensemblePath)
	suite.Require().NoError(err)

	// 2. start deployment
	deploymentResult := deployer.client.deploy(
		suite.T(), deployer.userContext,
		deployer.password,
		ensemblePath,
		"2m",
	)
	suite.Contains(deploymentResult, `"Status": "OK"`)
	ensembleID := extractEnsembleID(deploymentResult)

	// Wait until the deployment status is "Running".
	suite.Require().Eventually(func() bool {
		status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		if err != nil {
			suite.T().Logf("Error getting deployment status: %v", err)
			return false
		}
		suite.T().Log("Deployment status:", extractStatus(status))
		return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
	}, 2*60*time.Second, 5*time.Second, "Deployment did not reach Running status")

	//  3. assert running phase:
	providers := []*mockNode{bobProvider, aliceProvider, carlProvider}
	suite.assertRunningPhase(*ensembleCfg, ensembleID, deployer, providers)

	// track executions IDs so that we cna check if it was shutdown
	executions := make([]executionInfo, 0, len(providers))
	for _, provider := range providers {
		executions = append(executions, suite.getExecutions(provider)...)
	}

	// 4. Shutdown: assert if everything was freed
	shutdownRes := deployer.client.shutdownDeployment(
		suite.T(), deployer.userContext, deployer.password, ensembleID)
	suite.Contains(shutdownRes, `"Error": ""`)

	suite.waitDeploymentCompleted(deployer, deployer.dmsContext, ensembleID, 60*time.Second)

	// resources freed and allocations stopped
	for _, node := range suite.nodes {
		suite.assertFreeResourcesFull(node)
		suite.assertNoAllocationsRunning(node, executions...)
	}
}

// DeploymentDeploymentInfoTest deploys two tasks and one service (each on its own node),
// queries deployment info with usage, verifies the result, stops the deployment,
// then queries again and verifies there is no usage after allocations have stopped.
func DeploymentDeploymentInfoTest(suite *TestSuite) {
	suite.Require().Len(suite.nodes, 4)
	deployer := suite.nodes[0]
	// nodes[1], nodes[2], nodes[3] are the three onboarded compute providers

	ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "two-tasks-one-service.yaml")

	// 1. Deploy
	deploymentResult := deployer.client.deploy(
		suite.T(), deployer.userContext, deployer.password,
		ensemblePath, "2m",
	)
	suite.Contains(deploymentResult, `"Status": "OK"`)
	ensembleID := extractEnsembleID(deploymentResult)

	// 2. Wait until deployment is Running
	suite.Require().Eventually(func() bool {
		status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		if err != nil {
			suite.T().Logf("Error getting deployment status: %v", err)
			return false
		}
		suite.T().Log("Deployment status:", extractStatus(status))
		return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
	}, 2*60*time.Second, 5*time.Second, "Deployment did not reach Running status")

	// 3. Query deployment info with usage and verify result is sane
	infoRunning, err := deployer.client.deploymentInfo(suite.T(), deployer.userContext, deployer.password, ensembleID, true)
	suite.Require().NoError(err)
	suite.Require().Empty(infoRunning.Error, "deployment info should have no error")
	suite.Require().Equal(jobtypes.DeploymentStatusRunning.String(), infoRunning.Status)
	suite.Require().NotNil(infoRunning.Manifest, "manifest should be present when running")
	suite.Require().NotEmpty(infoRunning.Manifest.Allocations, "manifest should list allocations")
	suite.Require().NotEmpty(infoRunning.Allocations, "allocations details should be present when running")
	// With IncludeUsage=true we expect usage data for running allocations (may be empty for very fresh allocs)
	suite.Require().True(
		len(infoRunning.Allocations) >= 1,
		"at least one allocation should be reported when deployment is running",
	)

	// 4. Stop the deployment
	shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
	suite.Contains(shutdownRes, `"Error": ""`)

	suite.waitDeploymentCompleted(deployer, deployer.userContext, ensembleID, 60*time.Second)

	// 6. Query deployment info again with usage; assert usage was not included since allocations were shut down
	infoStopped, err := deployer.client.deploymentInfo(suite.T(), deployer.userContext, deployer.password, ensembleID, true)
	suite.Require().NoError(err)
	suite.Require().Empty(infoStopped.Error)
	suite.Require().Equal(jobtypes.DeploymentStatusCompleted.String(), infoStopped.Status)

	// Usage must not be included: allocations are shut down so there is nothing to report
	suite.Require().Empty(infoStopped.Usage,
		"usage should be empty after deployment has stopped and allocations are torn down",
	)
	for allocID, details := range infoStopped.Allocations {
		suite.Require().Nil(details.ExecutorStats,
			"allocation %q must not have ExecutorStats (usage) after shutdown", allocID,
		)
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

func DeploymentUpdates(suite *TestSuite) {
	// testRemoveNode tests removing a node from a multi-node ensemble
	suite.Run("RemoveNode", func() {
		// Deploy initial ensemble with multiple nodes
		deployer := suite.nodes[0]
		aliceProvider := suite.nodes[1]
		bobProvider := suite.nodes[2]

		// Get initial free resources to compare later
		aliceResourcesBefore, err := aliceProvider.client.freeResources(suite.T(), aliceProvider.dmsContext, aliceProvider.password)
		suite.Require().NoError(err)
		bobResourcesBefore, err := bobProvider.client.freeResources(suite.T(), bobProvider.dmsContext, bobProvider.password)
		suite.Require().NoError(err)

		// deploy
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "multiple.yaml"),
			"2m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		defer func() {
			// Clean up and test shutdown, and if resources were freed
			shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Contains(shutdownRes, `"Error": ""`)

			suite.Require().Eventually(func() bool {
				return checkResourcesEqual(suite.T(), aliceProvider, aliceResourcesBefore)
			}, 5*30*time.Second, 5*time.Second, "Resources were not freed after shutdown on Alice")

			suite.Require().Eventually(func() bool {
				return checkResourcesEqual(suite.T(), bobProvider, bobResourcesBefore)
			}, 5*30*time.Second, 5*time.Second, "Resources were not freed after shutdown on Bob")
		}()

		suite.waitDeploymentRunning(deployer, deployer.userContext, ensembleID, 5*60*time.Second)

		bobResourcesAfterDeployment, err := bobProvider.client.freeResources(
			suite.T(), bobProvider.dmsContext, bobProvider.password,
		)
		suite.NoError(err)
		aliceResourcesAfterDeployment, err := aliceProvider.client.freeResources(
			suite.T(), aliceProvider.dmsContext, aliceProvider.password,
		)
		suite.NoError(err)

		// Verify free resources decreased on at least one node after deployment
		suite.True(checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) ||
			checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore))

		// Update the ensemble by removing a node
		updateResult := deployer.client.update(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "remove_node.yaml"), ensembleID,
		)
		suite.Require().Contains(updateResult, `"OK": true`)

		suite.waitDeploymentRunning(deployer, deployer.userContext, ensembleID, 60*time.Second)

		// Deployment status should still be "Running"
		status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Require().NoError(err)
		suite.Require().Equal(extractStatus(status), jobtypes.DeploymentStatusRunning.String())

		suite.Require().Eventually(func() bool {
			aliceIncreased := checkResourcesIncreased(suite.T(), aliceProvider, aliceResourcesAfterDeployment)
			bobIncreased := checkResourcesIncreased(suite.T(), bobProvider, bobResourcesAfterDeployment)
			return aliceIncreased || bobIncreased
		}, 5*30*time.Second, 5*time.Second, "Free resources were not increased on any node after removing a node")
	})

	// testAddNode tests adding a node to a single-node ensemble
	suite.Run("AddNode", func() {
		// Deploy initial ensemble with a single node
		deployer := suite.nodes[0]
		aliceProvider := suite.nodes[1]
		bobProvider := suite.nodes[2]

		// Get initial free resources to compare later
		aliceResourcesBefore, err := aliceProvider.client.freeResources(
			suite.T(), aliceProvider.dmsContext, aliceProvider.password,
		)
		suite.Require().NoError(err)

		bobResourcesBefore, err := bobProvider.client.freeResources(
			suite.T(), bobProvider.dmsContext, bobProvider.password,
		)
		suite.Require().NoError(err)

		// Deploy the initial ensemble with a single node
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "single_node.yaml"),
			"2m",
		)
		suite.Require().Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		defer func() {
			// cleanup: shutdown and check if resources were freed
			shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Require().Contains(shutdownRes, `"Error": ""`)

			suite.Require().Eventually(func() bool {
				return checkResourcesEqual(suite.T(), aliceProvider, aliceResourcesBefore) &&
					checkResourcesEqual(suite.T(), bobProvider, bobResourcesBefore)
			}, 30*time.Second, 5*time.Second, "Resources were not freed after shutdown on Alice and Bob")
		}()

		// wait until deploymnet is runnning
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Initial deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Initial deployment did not reach Running status")

		// Verify resources were decreased on at least one node after initial deployment
		suite.Require().True(
			checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) ||
				checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore),
		)

		// Update the ensemble by adding a node
		updateResult := deployer.client.update(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "add_node.yaml"), ensembleID,
		)
		suite.Require().Contains(updateResult, `"OK": true`)

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Initial deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Initial deployment did not reach Running status")

		// Deployment status should still be "Running"
		status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Require().NoError(err)
		suite.Require().Equal(extractStatus(status), jobtypes.DeploymentStatusRunning.String())

		// Validate both nodes have decreased resources after the update
		suite.Require().Eventually(func() bool {
			return checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) &&
				checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore)
		}, 30*time.Second, 5*time.Second,
			"free resources should be decreased in relation with the initial state (for both nodes)")
	})

	// testAddAllocation tests adding and removing an allocation to a two-node ensemble
	suite.Run("AllocationUpdate", func() {
		// Deploy initial ensemble with a single node
		deployer := suite.nodes[0]
		aliceProvider := suite.nodes[1]
		bobProvider := suite.nodes[2]

		// Get initial free resources to compare later
		aliceResourcesBefore, err := aliceProvider.client.freeResources(
			suite.T(), aliceProvider.dmsContext, aliceProvider.password,
		)
		suite.Require().NoError(err)

		bobResourcesBefore, err := bobProvider.client.freeResources(
			suite.T(), bobProvider.dmsContext, bobProvider.password,
		)
		suite.Require().NoError(err)

		// Deploy the initial ensemble with a two nodes
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "multiple.yaml"),
			"2m",
		)
		suite.Require().Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		defer func() {
			// cleanup: shutdown and check if resources were freed
			shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Require().Contains(shutdownRes, `"Error": ""`)

			suite.Require().Eventually(func() bool {
				return checkResourcesEqual(suite.T(), aliceProvider, aliceResourcesBefore) &&
					checkResourcesEqual(suite.T(), bobProvider, bobResourcesBefore)
			}, 30*time.Second, 25*time.Second, "Resources were not freed after shutdown on Alice and Bob")
		}()

		// wait until deploymnet is runnning
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Initial deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Initial deployment did not reach Running status")

		// Verify resources were decreased on both nodes after initial deployment
		suite.Require().True(
			checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) &&
				checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore),
		)

		// Update the ensemble by adding an allocation
		addUpdateResult := deployer.client.update(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "add-allocation-to-multi.yaml"), ensembleID,
		)
		suite.Require().Contains(addUpdateResult, `"OK": true`)

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Updated deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Updated deployment did not reach Running status")

		// Deployment addStatus should still be "Running"
		addStatus, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Require().NoError(err)
		suite.Require().Equal(extractStatus(addStatus), jobtypes.DeploymentStatusRunning.String())

		// Validate both nodes have decreased resources after the update
		suite.Require().Eventually(func() bool {
			return checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) &&
				checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore)
		}, 30*time.Second, 5*time.Second,
			"free resources should be decreased in relation with the initial state (for both nodes)")

		// Update the ensemble by adding an allocation
		rmUpdateResult := deployer.client.update(
			suite.T(), deployer.userContext, deployer.password,
			filepath.Join(suite.testDataDir, "remove-allocation-from-multi.yaml"), ensembleID,
		)
		suite.Require().Contains(rmUpdateResult, `"OK": true`)

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Updated deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Updated deployment did not reach Running status")

		// Deployment status should still be "Running"
		status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Require().NoError(err)
		suite.Require().Equal(extractStatus(status), jobtypes.DeploymentStatusRunning.String())

		// Validate both nodes have decreased resources after the update
		suite.Require().Eventually(func() bool {
			return checkResourcesDecreased(suite.T(), aliceProvider, aliceResourcesBefore) &&
				checkResourcesDecreased(suite.T(), bobProvider, bobResourcesBefore)
		}, 30*time.Second, 5*time.Second,
			"free resources should be decreased in relation with the initial state (for both nodes)")
	})
}

// DeploymentRestorationOrchestratorPostReboot tests deployment persistence across orchestrator restarts.
func DeploymentRestorationOrchestratorPostReboot(suite *TestSuite) {
	suite.Run("DeploymentRestorationOrchestratorPostReboot", func() {
		// We need at least 2 nodes: 1 deployer and 1 provider
		suite.Require().Len(suite.nodes, 2)
		deployerIDX := 1
		deployer := suite.nodes[deployerIDX]

		// 1. Deploy a simple ensemble (using service-based ensemble to avoid task monitoring issues)
		ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "single-nginx.yaml")
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password, ensemblePath,
			"2m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		// 2. Wait for deployment to reach Running status
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")

		// Capture pre-restart allocations snapshot
		_, err := deployer.client.allocationsList(deployer.userContext, deployer.password)
		suite.Require().NoError(err)

		// 3. Verify deployment is in the list before restart
		deploymentsBefore, err := deployer.client.deploymentList(suite.T(), deployer.userContext, deployer.password)
		suite.Require().NoError(err)
		suite.Require().Contains(deploymentsBefore, ensembleID, "Deployment should be in list before restart")
		suite.Require().Equal(jobtypes.DeploymentStatusRunning.String(), deploymentsBefore[ensembleID], "Deployment should be Running before restart")

		// 4. Shutdown the deployer node (simulate restart)
		suite.stopNode(deployerIDX) // Stop the deployer node (index deployerIDX)

		// 5. Restart the deployer node
		go suite.startNode(deployerIDX)

		// Wait for the node to be ready again
		var networkStats types.NetworkStats
		suite.Require().Eventually(func() bool {
			var err error
			networkStats, err = deployer.client.self(suite.T(), deployer.dmsContext, deployer.password)
			return err == nil && networkStats.ID != ""
		}, 30*time.Second, 3*time.Second, "Deployer node should be ready after restart")

		// Update the peer ID in case it changed
		deployer.peerID = networkStats.ID

		// 6. Verify deployment is still in the list after restart
		suite.Require().Eventually(func() bool {
			deploymentsAfter, err := deployer.client.deploymentList(suite.T(), deployer.userContext, deployer.password)
			if err != nil {
				return false
			}
			return len(deploymentsAfter) > 0 && deploymentsAfter[ensembleID] != ""
		}, 60*time.Second, 5*time.Second, "Deployment should be in list after restart")

		// Capture post-restart allocations snapshot and logs for nginx2
		_, err = deployer.client.allocationsList(deployer.userContext, deployer.password)
		suite.Require().NoError(err)
		_, _ = deployer.client.deploymentLogs(deployer.userContext, deployer.password, ensembleID, "nginx2")

		// 7. Verify deployment status is still Running
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status after restart: %v", err)
				return false
			}
			suite.T().Log("Deployment status after restart:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment should still be Running after restart")

		time.Sleep(10 * time.Second)

		// 8. Clean up - shutdown the deployment
		shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Contains(shutdownRes, `"Error": ""`)

		// Wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.dmsContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status after shutdown: %v", err)
				return false
			}
			suite.T().Log("deployment status after shutdown:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
	})
}

// DeploymentRestorationProviderPostReboot tests provider-side allocation restoration after provider restart.
func DeploymentRestorationProviderPostReboot(suite *TestSuite) {
	suite.Run("DeploymentRestorationProviderPostReboot", func() {
		suite.Require().Len(suite.nodes, 2)
		providerIDX := 0
		deployerIDX := 1
		provider := suite.nodes[providerIDX]
		deployer := suite.nodes[deployerIDX]

		ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "single-nginx.yaml")
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext, deployer.password, ensemblePath,
			"2m",
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("Error getting deployment status: %v", err)
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")

		preRestartAllocs, err := provider.client.allocationsList(provider.userContext, provider.password)
		suite.Require().NoError(err)

		preRestartEnsembleAllocs := make(map[string]struct{})
		for _, alloc := range preRestartAllocs {
			if types.EnsembleIDFromAllocationID(alloc.ID) == ensembleID && alloc.ExecutionID != "" {
				preRestartEnsembleAllocs[alloc.ID] = struct{}{}
			}
		}
		suite.Require().NotEmpty(preRestartEnsembleAllocs, "expected at least one running allocation on provider before restart")

		suite.killNode(providerIDX)

		// ensure cp can connect to orch after restart since test nodes don't use the global bootstrap nodes
		bootstrapPeers := make([]string, 0, len(suite.nodes)-1)
		for i := 0; i < len(suite.nodes); i++ {
			if i == providerIDX {
				continue
			}
			otherNode := suite.nodes[i]
			otherStats, err := otherNode.client.self(suite.T(), otherNode.dmsContext, otherNode.password)
			suite.Require().NoError(err)
			for _, addr := range strings.Split(otherStats.ListenAddr, ", ") {
				bootstrapPeers = append(bootstrapPeers, fmt.Sprintf("%s/p2p/%s", addr, otherStats.ID))
			}
		}
		provider.config.BootstrapPeers = bootstrapPeers

		go suite.startNode(providerIDX)

		// XXX: bad delay - use a restore status indicator
		time.Sleep(30 * time.Second) // time to restore

		suite.Require().Eventually(func() bool {
			stats, err := provider.client.self(suite.T(), provider.dmsContext, provider.password)
			if err != nil {
				return false
			}
			return stats.ID != ""
		}, 60*time.Second, 2*time.Second, "provider node should be ready after restart")

		// reconnect restarted provider to the other nodes in the network
		for i := 0; i < len(suite.nodes); i++ {
			if i == providerIDX {
				continue
			}
			otherNode := suite.nodes[i]
			otherHostID, err := otherNode.client.self(suite.T(), otherNode.dmsContext, otherNode.password)
			suite.Require().NoError(err)
			result := provider.client.connect(suite.T(), provider.userContext, provider.password, otherHostID.ID)
			suite.Contains(result, `"Status": "CONNECTED"`)
		}

		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 2*time.Minute, 5*time.Second, "deployment should remain Running after provider restart")

		suite.Require().Eventually(func() bool {
			postRestartAllocs, err := provider.client.allocationsList(provider.userContext, provider.password)
			if err != nil {
				return false
			}
			for _, alloc := range postRestartAllocs {
				if types.EnsembleIDFromAllocationID(alloc.ID) == ensembleID && alloc.ExecutionID != "" {
					if _, ok := preRestartEnsembleAllocs[alloc.ID]; ok {
						return true
					}
				}
			}
			return false
		}, 2*time.Minute, 5*time.Second, "provider allocation was not restored after provider restart")

		shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Contains(shutdownRes, `"Error": ""`)
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.dmsContext, deployer.password, ensembleID)
			if err != nil {
				return false
			}
			return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
		}, 2*time.Minute, 5*time.Second, "deployment did not reach Completed status")
	})
}

// DeploymentRestorationFromProvisioning tests restoration when crash occurs at Provisioning
// XXX this test is not very reliable. it ends up being skipped on most runs because catching the
// provisioning status is difficult. Even if it was caught, between the time the status was seen
// and the time the node is killed, it most likely reaches a running state. It's kept because it's
// important but we definitely need a better way to test this scenario.
func DeploymentRestorationFromProvisioning(suite *TestSuite) {
	suite.Run("DeploymentRestorationFromProvisioning", func() {
		suite.Require().Len(suite.nodes, 2)
		deployerIDX := 1
		deployer := suite.nodes[deployerIDX]
		runner := suite.nodes[0]

		ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "single-nginx.yaml")
		deployRes := deployer.client.deploy(suite.T(), deployer.userContext, deployer.password, ensemblePath, "2m")
		suite.Contains(deployRes, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deployRes)

		cleanup := func() {
			shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Contains(shutdownRes, `"Error": ""`)
			suite.Require().Eventually(func() bool {
				status, err := deployer.client.deploymentStatus(suite.T(), deployer.dmsContext, deployer.password, ensembleID)
				return err == nil && extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
			}, 5*60*time.Second, 2*time.Second)
		}

		// Wait for deployment to reach Provisioning status and crash immediately
		suite.T().Log("Waiting for deployment to reach Provisioning status...")
		deadline := time.Now().Add(60 * time.Second)
		seen := make([]string, 0, 16)
		last := ""

		for time.Now().Before(deadline) {
			statusStr, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err == nil {
				cur := extractStatus(statusStr)
				if cur != last {
					seen = append(seen, cur)
					last = cur
					suite.T().Log("Deployment status:", cur)
				}
				if cur == jobtypes.DeploymentStatusProvisioning.String() {
					suite.T().Log("Deployment reached Provisioning status, crashing orchestrator immediately...")
					suite.killNode(1)
					break
				}
			}
		}

		// Provisioning status can be too quick to catch. If at running status, skip the test.
		if last != jobtypes.DeploymentStatusProvisioning.String() {
			if last == jobtypes.DeploymentStatusRunning.String() {
				cleanup()
				suite.T().Skipf("deployment %s Provisioning status was not caught. Seen %v", ensembleID, seen)
			}
			suite.T().Fatalf("deployment %s did not reach Provisioning status within 60s (seen: %v)", ensembleID, seen)
		}

		// Restart orchestrator node
		go suite.startNode(deployerIDX)

		// Wait until restarted node is ready
		suite.Require().Eventually(func() bool {
			stats, err := deployer.client.self(suite.T(), deployer.dmsContext, deployer.password)
			if err != nil {
				suite.T().Logf("Node not ready yet, error: %v", err)
				return false
			}
			suite.T().Logf("Node ready, ID: %s", stats.ID)
			return stats.ID != ""
		}, 5*60*time.Second, 2*time.Second)

		// Reconnect the restarted node to the existing network
		// This is crucial for the node to be able to send bid requests
		for i := 0; i < len(suite.nodes); i++ {
			if i == deployerIDX {
				continue // Skip the restarted node itself
			}
			otherNode := suite.nodes[i]
			otherHostID, err := otherNode.client.self(suite.T(), otherNode.dmsContext, otherNode.password)
			suite.Require().NoError(err)

			result := deployer.client.connect(suite.T(), deployer.userContext, deployer.password, otherHostID.ID)
			suite.Contains(result, `"Status": "CONNECTED"`)
		}

		// checkc status on CP side - if running, we didn't catch the status fast enough
		allocInfo, err := runner.client.allocationsList(runner.userContext, runner.password)
		suite.Require().NoError(err)
		for _, alloc := range allocInfo {
			if alloc.Status == jobtypes.AllocationRunning.String() && types.EnsembleIDFromAllocationID(alloc.ID) == ensembleID {
				suite.T().Skipf("deployment %s Provisioning status was not caught. Alloc Status %v", ensembleID, alloc.Status)
			}
		}

		// After restoration, the deployment should have progressed from Provisioning to Running
		// This is expected behavior - the orchestrator automatically continues the deployment process
		suite.T().Log("=== PHASE 4: Checking deployment status after restoration ===")
		suite.waitDeploymentRunning(deployer, deployer.userContext, ensembleID, 5*time.Minute)

		// cleanup
		cleanup()
	})
}

// DeploymentRestorationFromPreparing tests restoration when crash occurs at Preparing
func DeploymentRestorationFromPreparing(suite *TestSuite) {
	suite.Run("DeploymentRestorationFromPreparing", func() {
		suite.Require().Len(suite.nodes, 3)
		deployer := suite.nodes[1]

		ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "nginx.yaml")
		deployRes := deployer.client.deploy(suite.T(), deployer.userContext, deployer.password, ensemblePath, "2m")
		suite.Contains(deployRes, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deployRes)

		// Wait for deployment to reach Preparing status and crash immediately
		suite.T().Log("Waiting for deployment to reach Preparing status...")
		deadline := time.Now().Add(60 * time.Second)
		seen := make([]string, 0, 16)
		last := ""

		for time.Now().Before(deadline) {
			statusStr, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err == nil {
				cur := extractStatus(statusStr)
				if cur != last {
					seen = append(seen, cur)
					last = cur
					suite.T().Log("Deployment status:", cur)
				}
				if cur == jobtypes.DeploymentStatusPreparing.String() {
					suite.T().Log("Deployment reached Preparing status, crashing orchestrator immediately...")
					suite.stopNode(1)
					break
				}
			}
			// High frequency check to catch Preparing status reliably
			time.Sleep(100 * time.Millisecond)
		}

		// If we didn't find Preparing status, fail the test
		if last != jobtypes.DeploymentStatusPreparing.String() {
			suite.T().Fatalf("deployment %s did not reach Preparing status within 60s (seen: %v)", ensembleID, seen)
		}

		// Restart orchestrator node
		go suite.startNode(1)

		// Wait until restarted node is ready
		suite.Require().Eventually(func() bool {
			stats, err := suite.nodes[1].client.self(suite.T(), suite.nodes[1].dmsContext, suite.nodes[1].password)
			if err != nil {
				suite.T().Logf("Node not ready yet, error: %v", err)
				return false
			}
			suite.T().Logf("Node ready, ID: %s", stats.ID)
			return stats.ID != ""
		}, 60*time.Second, 2*time.Second)

		// Reconnect the restarted node to the existing network
		// This is crucial for the node to be able to send bid requests
		for i := 0; i < len(suite.nodes); i++ {
			if i == 1 {
				continue // Skip the restarted node itself
			}
			otherNode := suite.nodes[i]
			otherHostID, err := otherNode.client.self(suite.T(), otherNode.dmsContext, otherNode.password)
			suite.Require().NoError(err)

			result := deployer.client.connect(suite.T(), deployer.userContext, deployer.password, otherHostID.ID)
			suite.Contains(result, `"Status": "CONNECTED"`)
		}

		time.Sleep(30 * time.Second) // wait for the deployment to be restored

		// After restoration, the deployment should have progressed from Preparing to Running
		// This is expected behavior - the orchestrator automatically continues the deployment process
		suite.T().Log("=== PHASE 4: Checking deployment status after restoration ===")
		statusStr, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		suite.Require().NoError(err)
		currentStatus := extractStatus(statusStr)
		suite.T().Logf("=== DEPLOYMENT STATUS AFTER RESTORATION: %s ===", currentStatus)
		suite.Require().Equal(jobtypes.DeploymentStatusRunning.String(), currentStatus, "expected Running after restoration from Preparing")

		// Then it should stay Running or progress to Completed
		suite.T().Log("=== PHASE 5: Waiting for deployment to stay Running or progress to Completed ===")
		suite.Require().Eventually(func() bool {
			status, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			if err != nil {
				suite.T().Logf("=== ERROR GETTING DEPLOYMENT STATUS: %v ===", err)
				return false
			}
			cur := extractStatus(status)
			suite.T().Logf("=== CURRENT DEPLOYMENT STATUS: %s ===", cur)
			return cur == jobtypes.DeploymentStatusRunning.String() || cur == jobtypes.DeploymentStatusCompleted.String()
		}, 5*60*time.Second, 5*time.Second, "deployment did not stay Running or progress to Completed after restoration")

		// Cleanup (only if not already completed)
		finalStatus, err := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
		if err == nil && extractStatus(finalStatus) != jobtypes.DeploymentStatusCompleted.String() {
			shutdownRes := deployer.client.shutdownDeployment(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Contains(shutdownRes, `"Error": ""`)
			suite.Require().Eventually(func() bool {
				status, err := deployer.client.deploymentStatus(suite.T(), deployer.dmsContext, deployer.password, ensembleID)
				return err == nil && extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
			}, 5*60*time.Second, 2*time.Second)
		}
	})
}
