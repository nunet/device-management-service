// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build e2e || !unit

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/network/utils"
)

// TestE2E is the entry point for the e2e tests.
//
// We need to ensure the following while adding more tests
// 1. Tests should run in parallel
// 2. portIndex should be unique for each test suite
// 3. Runner function should be defined in the respective test file and must be passed to the test suite
func TestE2E(t *testing.T) {
	t.Parallel()
	var (
		testSuites         = 17
		totalPortsRequired = 2 * testSuites
	)

	ports, err := utils.GetMultipleAvailablePorts(totalPortsRequired)
	require.NoError(t, err)
	require.Len(t, ports, totalPortsRequired)

	t.Run("BasicTests", func(t *testing.T) {
		basicTests := &TestSuite{
			numNodes:      3,
			Name:          "basic_tests",
			restPortIndex: ports[0],
			p2pPortIndex:  ports[1],
			runner:        BasicTests,
		}
		suite.Run(t, basicTests)
	})

	t.Run("DeploymentTests", func(t *testing.T) {
		deploymentTests := &TestSuite{
			numNodes:      3,
			Name:          "deployment_tests",
			restPortIndex: ports[2],
			p2pPortIndex:  ports[3],
			runner:        DeploymentTests,
		}
		suite.Run(t, deploymentTests)
	})

	t.Run("DeploymentWithRedundancy", func(t *testing.T) {
		deploymentWithRedundancyTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_redundancy_tests",
			restPortIndex: ports[4],
			p2pPortIndex:  ports[5],
			runner:        DeploymentWithRedundancyTest,
		}
		suite.Run(t, deploymentWithRedundancyTests)
	})

	t.Run("DeploymentWithContracts", func(t *testing.T) {
		deploymentWithContractsTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_tests",
			restPortIndex: ports[6],
			p2pPortIndex:  ports[7],
			runner:        DeployWithContractTest,
		}
		suite.Run(t, deploymentWithContractsTests)
	})

	t.Run("DeploymentUpdates", func(t *testing.T) {
		deploymentUpdates := &TestSuite{
			numNodes:      3,
			Name:          "deployment_updates",
			restPortIndex: ports[8],
			p2pPortIndex:  ports[9],
			runner:        DeploymentUpdates,
		}
		suite.Run(t, deploymentUpdates)
	})

	t.Run("DeploymentAssertSubnet", func(t *testing.T) {
		deploymentFullAssertion := &TestSuite{
			numNodes: 4,
			// TODO unify names
			Name:          "deployment_assert_subnet",
			restPortIndex: ports[10],
			p2pPortIndex:  ports[11],
			runner:        DeploymentAssertSubnet,
		}
		suite.Run(t, deploymentFullAssertion)
	})

	t.Run("DeploymentRestorationPostReboot", func(t *testing.T) {
		deploymentRestoration := &TestSuite{
			numNodes:      2,
			Name:          "deployment_restoration_post_reboot",
			restPortIndex: ports[12],
			p2pPortIndex:  ports[13],
			runner:        DeploymentRestorationPostReboot,
		}
		suite.Run(t, deploymentRestoration)
	})

	// not dependable test since 'Provisioning' status can sometimes be too quick to catch
	// the test is skipped if the status couldn't be caught
	t.Run("DeploymentRestorationFromProvisioning", func(t *testing.T) {
		provisioning := &TestSuite{
			numNodes:      2,
			Name:          "deployment_restoration_from_provisioning",
			restPortIndex: ports[14],
			p2pPortIndex:  ports[15],
			runner:        DeploymentRestorationFromProvisioning,
		}
		suite.Run(t, provisioning)
	})

	t.Run("DeploymentRestorationFromPreparing", func(t *testing.T) {
		preparing := &TestSuite{
			numNodes:      3,
			Name:          "deployment_restoration_from_preparing",
			restPortIndex: ports[16],
			p2pPortIndex:  ports[17],
			runner:        DeploymentRestorationFromPreparing,
		}
		suite.Run(t, preparing)
	})

	// t.Run("DeployWithOnDemandProvisioner", func(t *testing.T) {
	// 	deploymentWithOnDemandProvisionerTests := &TestSuite{
	// 		numNodes:      2,
	// 		Name:          "deployment_with_ondemand_provisioner_tests",
	// 		restPortIndex: ports[18],
	// 		p2pPortIndex:  ports[19],
	// 		runner:        DeployWithOnDemandProvisioner,
	// 	}
	// 	suite.Run(t, deploymentWithOnDemandProvisionerTests)
	// })

	t.Run("DeployWithContractPayPerDeployment", func(t *testing.T) {
		t.Parallel()

		deploymentWithContractPayPerDeploymentTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_pay_per_deployment_tests",
			restPortIndex: ports[20],
			p2pPortIndex:  ports[21],
			runner:        DeployWithContractPayPerDeploymentTest,
		}
		suite.Run(t, deploymentWithContractPayPerDeploymentTests)
	})

	t.Run("DeploymentWithContractsCollectAfterPay", func(t *testing.T) {
		deploymentWithContractsCollectAfterPayTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_collect_after_pay_tests",
			restPortIndex: ports[22],
			p2pPortIndex:  ports[23],
			runner:        DeployWithContractCollectAfterPayTest,
		}
		suite.Run(t, deploymentWithContractsCollectAfterPayTests)
	})

	t.Run("DeployWithContractPayPerTimeUtilization", func(t *testing.T) {
		t.Parallel()

		deploymentWithContractPayPerTimeUtilizationTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_pay_per_time_utilization_tests",
			restPortIndex: ports[24],
			p2pPortIndex:  ports[25],
			runner:        DeployWithContractPayPerTimeUtilizationTest,
		}
		suite.Run(t, deploymentWithContractPayPerTimeUtilizationTests)
	})

	t.Run("DeployWithContractPayPerResourceUtilization", func(t *testing.T) {
		t.Parallel()

		deploymentWithContractPayPerResourceUtilizationTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_pay_per_resource_utilization_tests",
			restPortIndex: ports[26],
			p2pPortIndex:  ports[27],
			runner:        DeployWithContractPayPerResourceUtilizationTest,
		}
		suite.Run(t, deploymentWithContractPayPerResourceUtilizationTests)
	})

	t.Run("DeployWithContractFixedRental", func(t *testing.T) {
		t.Parallel()

		deploymentWithContractFixedRentalTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_fixed_rental_tests",
			restPortIndex: ports[28],
			p2pPortIndex:  ports[29],
			runner:        DeployWithContractFixedRentalTest,
		}
		suite.Run(t, deploymentWithContractFixedRentalTests)
	})
	t.Run("DeployWithContractPeriodic", func(t *testing.T) {
		t.Parallel()

		deploymentWithContractPeriodicTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_periodic_tests",
			restPortIndex: ports[30],
			p2pPortIndex:  ports[31],
			runner:        DeployWithContractPeriodicTest,
		}
		suite.Run(t, deploymentWithContractPeriodicTests)
	})
	t.Run("DeploymentWithContractsEnforcedProviders", func(t *testing.T) {
		deploymentWithContractsEnforcedProvidersTests := &TestSuite{
			numNodes:      4,
			Name:          "deployment_with_contracts_enforced_providers_tests",
			restPortIndex: ports[32],
			p2pPortIndex:  ports[33],
			runner:        DeployWithContractsEnforcedProvidersTest,
		}
		suite.Run(t, deploymentWithContractsEnforcedProvidersTests)
	})

	t.Run("DeploymentWithContractChainTest", func(t *testing.T) {
		t.Parallel()
		deploymentWithContractChainTests := &TestSuite{
			numNodes:            6,
			Name:                "deployment_with_contract_chain_tests",
			restPortIndex:       ports[28],
			p2pPortIndex:        ports[29],
			runner:              DeployWithContractChainTest,
			capabilitiesHandler: setupContractChainCapabilities,
		}
		suite.Run(t, deploymentWithContractChainTests)
	})

	// Disabled because too flaky since 'Provisioning' status is too quick to catch
	// will fix soon - for now DeploymentRestorationFromCommitting covers a very similar
	// test case
	// t.Run("DeploymentRestorationFromProvisioningTaskAllocation", func(t *testing.T) {
	// 	t.Parallel()
	// 	provisioningTask := &TestSuite{
	// 		numNodes:      3,
	// 		Name:          "deployment_restoration_from_provisioning_task",
	// 		restPortIndex: ports[18],
	// 		p2pPortIndex:  ports[19],
	// 		runner:        DeploymentRestorationFromProvisioningTaskAllocation,
	// 	}
	// 	suite.Run(t, provisioningTask)
	// })
}
