package itest

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"

	cmd "gitlab.com/nunet/device-management-service/cmd/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
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

	suite.Run("must be able to deploy nginx demo", func() {
		// deploy nginx.yaml to node1 using node2's orchestrator
		node1 := suite.nodes[0]
		hostname, err := os.Hostname()
		suite.Require().NoError(err)
		srcFile := filepath.Join(suite.testDataDir, "ensembles", "nginx.yaml")
		destinationFile := filepath.Join(node1.config.WorkDir, "nginx.yaml")
		err = copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)
		err = replaceHostnameInFile(destinationFile, hostname)
		suite.Require().NoError(err)

		freeResourcesBefore, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)

		// Deploy nginx.yaml from node2's orchestrator.
		node2 := suite.nodes[1]
		deploymentResult := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(node1.config.WorkDir, "nginx.yaml"))
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

	suite.Run("must be able to deploy docker hello-world", func() {
		node2 := suite.nodes[1]
		deployment2Result := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(suite.testDataDir, "ensembles", "hello.yaml"))
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

	suite.Run("must completely assert deployment with nginx", func() {
		// This test will be a model for other tests. It concerns
		// with asserting every possible effect, meaning it
		// tries to be a complete test
		deployer := suite.nodes[0]
		bobProvider := suite.nodes[1]
		aliceProvider := suite.nodes[2]

		suite.assertFreeResourcesFull(bobProvider)
		suite.assertFreeResourcesFull(aliceProvider)

		suite.assertNoAllocationsRunning(bobProvider)
		suite.assertNoAllocationsRunning(aliceProvider)

		ensemblePath := filepath.Join(suite.testDataDir, "ensembles", "nginx.yaml")
		alloc1 := "nginx1"

		r, err := cmd.ProcessEnsembleYaml(afero.Afero{Fs: afero.NewOsFs()}, ensemblePath)
		suite.Require().NoError(err)
		ensembleCfg := r.Ensemble

		// start deployment
		deploymentResult := deployer.client.deploy(
			suite.T(), deployer.userContext,
			deployer.password,
			ensemblePath,
		)
		suite.Contains(deploymentResult, `"Status": "OK"`)
		ensembleID := extractEnsembleID(deploymentResult)

		executions := []execution{}

		// defer shutdown and assert its effects
		defer func() {
			// Shutdown the nginx deployment.
			shutdownRes := deployer.client.shutdownDeployment(
				suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.Contains(shutdownRes, `"Error": ""`)

			// wait for the ensemble to be shutdown
			suite.Require().Eventually(func() bool {
				status := deployer.client.deploymentStatus(
					suite.T(), deployer.dmsContext, deployer.password, ensembleID)
				suite.T().Log("deployment status:", extractStatus(status))
				return extractStatus(status) == jobtypes.DeploymentStatusCompleted.String()
			}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
			time.Sleep(3 * time.Second)

			// Ensure no allocations
			suite.assertNoAllocationsRunning(bobProvider, executions...)
			suite.assertNoAllocationsRunning(aliceProvider, executions...)

			// Ensure resources are freed.
			suite.assertFreeResourcesFull(bobProvider)
			suite.assertFreeResourcesFull(aliceProvider)
		}()

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status := deployer.client.deploymentStatus(suite.T(), deployer.userContext, deployer.password, ensembleID)
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusRunning.String()
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")
		time.Sleep(2 * time.Second)

		// Assert allocation is running
		suite.assertOnlyOneAllocationRunning(bobProvider)
		suite.assertOnlyOneAllocationRunning(aliceProvider)

		// Ensure resources are allocated.
		alloc, ok := ensembleCfg.Allocation(alloc1)
		suite.Require().True(ok)
		suite.assertResourcesAfterDeployment(bobProvider, alloc.Resources)
		suite.assertResourcesAfterDeployment(aliceProvider, alloc.Resources)

		suite.assertManifestAfterDeployment(
			deployer, []*mockNode{bobProvider, aliceProvider},
			ensembleCfg, ensembleID)

		// track executions IDs
		executions = append(
			executions, suite.getExecutions(bobProvider)...)
		executions = append(
			executions, suite.getExecutions(aliceProvider)...)
	})
}

func (s *TestSuite) getExecutions(node *mockNode) []execution {
	executions := []execution{}

	allocations, err := node.client.allocationsList(
		node.userContext, node.password)
	s.Require().NoError(err)

	for _, allocation := range allocations {
		if allocation.Container != "" {
			executions = append(executions, execution{
				executor: allocation.Executor,
				id:       allocation.Container,
			})
		}
	}
	return executions
}
