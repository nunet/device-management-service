package itest

import (
	"os"
	"path/filepath"
	"time"

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
}
