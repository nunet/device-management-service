// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package steps

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
)

// NAT registers all step definitions for NAT feature
func NAT(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		// Cleanup any existing NAT networks and instances
		if err := cleanupNATInfrastructure(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		// Save logs and cleanup
		if err := hooks.SaveLogs(ctx); err != nil {
			return ctx, err
		}
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}

		if err := cleanupNATInfrastructure(); err != nil {
			return ctx, err
		}

		return ctx, nil
	})

	ctx.Step(`^I have (\d+) DMS nodes on isolated NAT networks$`, haveNodesOnIsolatedNATNetworks)
	ctx.Step(`^I create a relay node "([^"]*)" on public network$`, createRelayNodeOnPublicNetwork)
	ctx.Step(`^"([^"]*)" retrieves its libp2p address$`, nodeRetrievesLibp2pAddress)
	ctx.Step(`^"([^"]*)" connects to relay "([^"]*)"$`, nodeConnectsToRelay)
	ctx.Step(`^I wait (\d+) seconds for AutoNAT and relay circuits to establish$`, waitForRelayCircuits)
	ctx.Step(`^"([^"]*)" should have a relay address advertised$`, nodeShouldHaveRelayAddress)
	ctx.Step(`^"([^"]*)" attempts to connect to "([^"]*)"$`, nodeAttemptsToConnect)
	ctx.Step(`^the connection should fail due to NAT$`, connectionShouldFailDueToNAT)
	ctx.Step(`^the connection should succeed via relay$`, connectionShouldSucceed)
}

type natNodeInfo struct {
	name    string
	network string
}

func createRelayNodeOnPublicNetwork(ctx context.Context, relayName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	conf, err := config.Get()
	assert.NoError(t, err)

	clients, err := utils.ConnectToClients(conf)
	assert.NoError(t, err)

	client := clients[0]

	fmt.Printf("[SETUP] Creating relay node %s on default network (accessible to all)...\n", relayName)

	// Create relay node on the default incusbr0 network (accessible to all NAT'd nodes)
	instanceName := fmt.Sprintf("%s-%s", conf.VMsPrefix, strings.ToLower(relayName))
	err = utils.CreateInstance(client, utils.GetInstanceType(), instanceName)
	assert.NoError(t, err)

	relayNode := &utils.Node{
		Name:     instanceName,
		Client:   client,
		Contexts: make(map[string]*utils.Context),
	}

	fmt.Printf("[SETUP] Waiting for relay node %s to be ready...\n", instanceName)
	err = relayNode.WaitForInstanceReady()
	assert.NoError(t, err)

	// Upload DMS binary
	here := dutils.CurrentFileDirectory()
	remoteDMSPath := "/usr/local/bin/nunet"
	localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")

	fmt.Printf("[SETUP] Uploading DMS binary to relay %s...\n", instanceName)
	err = relayNode.UploadFile(localPath, remoteDMSPath, 0o755)
	assert.NoError(t, err)

	_, err = relayNode.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"})
	assert.NoError(t, err)

	// Configure VM networking
	err = relayNode.ConfigureVMNetworkingForQUIC()
	assert.NoError(t, err)

	// Create DMS contexts for relay
	fmt.Printf("[SETUP] Creating DMS contexts for relay %s...\n", relayName)
	userCtx, dmsCtx, err := relayNode.InitialCaps(strings.ToLower(relayName))
	assert.NoError(t, err)
	assert.NotNil(t, userCtx)
	assert.NotNil(t, dmsCtx)

	// Run DMS on relay
	fmt.Printf("[SETUP] Starting DMS on relay %s...\n", relayName)
	err = dmsCtx.Run(t)
	assert.NoError(t, err)

	require.Eventually(t, func() bool {
		running := relayNode.IsDMSRunning(9999)
		if running {
			fmt.Printf("[SETUP] Relay DMS is running on %s\n", relayName)
		}
		return running
	}, 30*time.Second, 500*time.Millisecond)

	// Give relay a moment to stabilize and start advertising itself
	fmt.Printf("[SETUP] Waiting for relay %s to stabilize...\n", relayName)
	time.Sleep(3 * time.Second)

	// Get relay address
	relayInfo, err := dmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, relayInfo)

	relayAddr, err := utils.MultiaddrFromCLI(relayInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, relayAddr)

	// Get existing nodeMap and add relay to it
	nodeMap, err := tc.NodeMap()
	if err != nil || nodeMap == nil {
		nodeMap = make(map[string]*utils.Node)
	}
	nodeMap[strings.ToLower(relayName)] = relayNode
	tc = tc.WithNodeMap(nodeMap)

	return tc.Unwrap(), nil
}

func haveNodesOnIsolatedNATNetworks(ctx context.Context, count int, table *godog.Table) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := parseNATNodesTable(table)
	assert.NoError(t, err)
	assert.Len(t, nodes, count)

	conf, err := config.Get()
	assert.NoError(t, err)

	clients, err := utils.ConnectToClients(conf)
	assert.NoError(t, err)

	// Get existing nodeMap (may include relay node)
	nodeMap, err := tc.NodeMap()
	if err != nil || nodeMap == nil {
		nodeMap = make(map[string]*utils.Node)
	}

	natRouters := make([]*utils.NATRouterContainer, 0)
	networksCreated := make(map[string]bool)
	// Use 172.16.x.x range (safe, won't conflict with typical home networks 192.168.x.x)
	networkToSubnet := map[string]string{
		"nat-net-1": "172.16.1.1/24", // Private range, unlikely to conflict
		"nat-net-2": "172.16.2.1/24", // Private range, unlikely to conflict
	}

	fmt.Printf("[SETUP] Creating %d DMS nodes behind container-based NAT routers...\n", count)

	// First pass: Create private networks and NAT router containers
	for idx, nodeInfo := range nodes {
		if !networksCreated[nodeInfo.network] {
			client := clients[idx%len(clients)]
			subnet := networkToSubnet[nodeInfo.network]

			// Build list of OTHER private subnets to block
			blockedSubnets := make([]string, 0)
			for otherNet, otherSubnet := range networkToSubnet {
				if otherNet != nodeInfo.network {
					blockedSubnets = append(blockedSubnets, otherSubnet)
				}
			}
			fmt.Printf("[SETUP] Network %s will block: %v\n", nodeInfo.network, blockedSubnets)

			// Create private network WITH firewall rules to block other private networks
			err := utils.CreatePrivateNetwork(client, nodeInfo.network, subnet, blockedSubnets)
			assert.NoError(t, err)

			// Create NAT router container with firewall rules
			routerName := fmt.Sprintf("%s-nat-router-%d", conf.VMsPrefix, idx+1)
			router, err := utils.CreateNATRouterContainer(
				client,
				routerName,
				"incusbr0",       // External network (public-like)
				nodeInfo.network, // Internal network (private)
				subnet,
				blockedSubnets, // Block other NAT networks
			)
			assert.NoError(t, err)
			natRouters = append(natRouters, router)

			networksCreated[nodeInfo.network] = true
		}
	}

	// Second pass: Create DMS nodes behind NAT routers
	for i, nodeInfo := range nodes {
		client := clients[i%len(clients)]
		gatewayIP := networkToSubnet[nodeInfo.network][:len(networkToSubnet[nodeInfo.network])-4] + "1" // x.x.x.1

		// Create instance behind NAT router
		instanceName := fmt.Sprintf("%s-%s", conf.VMsPrefix, nodeInfo.name)
		fmt.Printf("[SETUP] Creating instance: %s behind NAT on network: %s\n", instanceName, nodeInfo.network)

		err = utils.CreateInstanceBehindNAT(client, utils.GetInstanceType(), instanceName, nodeInfo.network, gatewayIP)
		assert.NoError(t, err)

		node := &utils.Node{
			Name:     instanceName,
			Client:   client,
			Contexts: make(map[string]*utils.Context),
		}

		// Wait for instance to be ready
		fmt.Printf("[SETUP] Waiting for instance %s to be ready...\n", instanceName)
		err = node.WaitForInstanceReady()
		assert.NoError(t, err)

		// Upload DMS binary
		here := dutils.CurrentFileDirectory()
		remoteDMSPath := "/usr/local/bin/nunet"
		localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")

		fmt.Printf("[SETUP] Uploading DMS binary to %s...\n", instanceName)
		err = node.UploadFile(localPath, remoteDMSPath, 0o755)
		assert.NoError(t, err)

		_, err = node.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"})
		assert.NoError(t, err)

		// Configure VM networking for QUIC (if VM)
		err = node.ConfigureVMNetworkingForQUIC()
		assert.NoError(t, err)

		// No need to configure NAT in the node itself - the NAT router handles it!
		// The node is truly behind NAT now with private IP (192.168.x.x)

		// Create DMS contexts
		fmt.Printf("[SETUP] Creating DMS contexts for %s...\n", nodeInfo.name)
		userCtx, dmsCtx, err := node.InitialCaps(nodeInfo.name)
		assert.NoError(t, err)
		assert.NotNil(t, userCtx)
		assert.NotNil(t, dmsCtx)

		// Run DMS
		fmt.Printf("[SETUP] Starting DMS on %s...\n", nodeInfo.name)
		err = dmsCtx.Run(t)
		assert.NoError(t, err)

		// Wait for DMS to be running
		require.Eventually(t, func() bool {
			running := node.IsDMSRunning(9999)
			if running {
				fmt.Printf("[SETUP] DMS is running on %s\n", nodeInfo.name)
			}
			return running
		}, 30*time.Second, 500*time.Millisecond)

		nodeMap[nodeInfo.name] = node
	}

	tc = tc.WithNodeMap(nodeMap)
	tc = tc.WithNATRouters(natRouters)

	fmt.Printf("[SETUP] All %d NAT nodes created behind container NAT routers with true symmetrical NAT\n", count)

	return tc.Unwrap(), nil
}

func nodeRetrievesLibp2pAddress(ctx context.Context, nodeName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)

	node, dmsCtx := utils.NodeWithDMS(nodeMap, nodeName)
	assert.NotNil(t, node, fmt.Sprintf("Node %s not found", nodeName))
	assert.NotNil(t, dmsCtx, fmt.Sprintf("DMS context for %s not found", nodeName))

	peerInfo, err := dmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, peerInfo)
	assert.NotEmpty(t, peerInfo.Address, "Peer address should not be empty")
	assert.NotEmpty(t, peerInfo.ID, "Peer ID should not be empty")

	return ctx, nil
}

func nodeConnectsToRelay(ctx context.Context, nodeName, relayName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)

	_, nodeDmsCtx := utils.NodeWithDMS(nodeMap, nodeName)
	assert.NotNil(t, nodeDmsCtx, fmt.Sprintf("Node %s not found", nodeName))

	_, relayDmsCtx := utils.NodeWithDMS(nodeMap, relayName)
	assert.NotNil(t, relayDmsCtx, fmt.Sprintf("Relay %s not found", relayName))

	relayInfo, err := relayDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, relayInfo)

	relayAddr, err := utils.MultiaddrFromCLI(relayInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, relayAddr)

	fmt.Printf("[RELAY] %s connecting to relay %s at %s...\n", nodeName, relayName, relayAddr)

	err = nodeDmsCtx.Connect(relayAddr)
	assert.NoError(t, err, fmt.Sprintf("%s should be able to connect to relay %s", nodeName, relayName))

	fmt.Printf("[RELAY] %s successfully connected to relay %s\n", nodeName, relayName)

	return ctx, nil
}

func waitForRelayCircuits(ctx context.Context, seconds int) (context.Context, error) {
	fmt.Printf("[RELAY] Waiting %d seconds for AutoNAT detection and relay circuits\n", seconds)

	// Wait for relay circuits to establish
	time.Sleep(time.Duration(seconds) * time.Second)
	return ctx, nil
}

func nodeShouldHaveRelayAddress(ctx context.Context, nodeName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)

	_, dmsCtx := utils.NodeWithDMS(nodeMap, nodeName)
	assert.NotNil(t, dmsCtx, fmt.Sprintf("Node %s not found", nodeName))

	peerInfo, err := dmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, peerInfo)
	assert.NotEmpty(t, peerInfo.Address)

	// Check if there's a relay address (p2p-circuit)
	addresses := strings.Split(peerInfo.Address, ", ")
	hasRelayAddr := false

	for _, addr := range addresses {
		if strings.Contains(addr, "/p2p-circuit") {
			hasRelayAddr = true
		}
	}

	if !hasRelayAddr {
		return ctx, fmt.Errorf("node %s does not have relay address advertised", nodeName)
	}

	return ctx, nil
}

func nodeAttemptsToConnect(ctx context.Context, sourceNode, targetNode string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.NodeMap()
	assert.NoError(t, err)

	_, sourceDmsCtx := utils.NodeWithDMS(nodeMap, sourceNode)
	assert.NotNil(t, sourceDmsCtx, fmt.Sprintf("Source node %s not found", sourceNode))

	_, targetDmsCtx := utils.NodeWithDMS(nodeMap, targetNode)
	assert.NotNil(t, targetDmsCtx, fmt.Sprintf("Target node %s not found", targetNode))

	targetInfo, err := targetDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, targetInfo)

	targetAddr, err := utils.MultiaddrFromCLI(targetInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, targetAddr, "Target multiaddr should not be empty")

	// Attempt connection
	connectionStart := time.Now()
	err = sourceDmsCtx.Connect(targetAddr)
	connectionDuration := time.Since(connectionStart)

	// Store connection attempt details in context
	connectionAttempt := map[string]interface{}{
		"source":   sourceNode,
		"target":   targetNode,
		"address":  targetAddr,
		"error":    err,
		"duration": connectionDuration,
	}

	return tc.WithConnectionAttempt(connectionAttempt).Unwrap(), nil
}

func connectionShouldFailDueToNAT(ctx context.Context) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	attempt, err := tc.ConnectionAttempt()
	assert.NoError(t, err)
	assert.NotNil(t, attempt, "Connection attempt should be recorded")

	connErr, _ := attempt["error"].(error)

	// ASSERT: Connection must fail (connErr must NOT be nil)
	assert.Error(t, connErr, "Expected connection to FAIL due to NAT, but it SUCCEEDED")

	if connErr == nil {
		return fmt.Errorf("connection unexpectedly succeeded without relay - NAT isolation not working")
	}

	return nil
}

func connectionShouldSucceed(ctx context.Context) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	attempt, err := tc.ConnectionAttempt()
	assert.NoError(t, err)
	assert.NotNil(t, attempt, "Connection attempt should be recorded")

	connErr, _ := attempt["error"].(error)

	// ASSERT: Connection must succeed (connErr must be nil)
	assert.NoError(t, connErr, "Expected connection to SUCCEED via relay, but it FAILED")

	if connErr != nil {
		return fmt.Errorf("connection failed even with relay: %v", connErr)
	}

	return nil
}

func parseNATNodesTable(table *godog.Table) ([]natNodeInfo, error) {
	if len(table.Rows) < 2 {
		return nil, fmt.Errorf("table must have header and at least one data row")
	}

	expectedHeaders := []string{"name", "network"}
	header := table.Rows[0]

	if len(header.Cells) != len(expectedHeaders) {
		return nil, fmt.Errorf("expected %d columns, got %d", len(expectedHeaders), len(header.Cells))
	}

	for i, expected := range expectedHeaders {
		if header.Cells[i].Value != expected {
			return nil, fmt.Errorf("expected header '%s' at column %d, got '%s'",
				expected, i, header.Cells[i].Value)
		}
	}

	var nodes []natNodeInfo
	for i := 1; i < len(table.Rows); i++ {
		row := table.Rows[i]

		node := natNodeInfo{
			name:    strings.ToLower(row.Cells[0].Value),
			network: strings.ToLower(row.Cells[1].Value),
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func cleanupNATInfrastructure() error {
	conf, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	clients, err := utils.ConnectToClients(conf)
	if err != nil {
		return fmt.Errorf("failed to connect to clients: %w", err)
	}

	fmt.Println("[CLEANUP] Cleaning up NAT infrastructure...")

	// Remove host-level iptables rules first
	err = utils.RemoveHostCrossNATBlocking()
	if err != nil {
		return fmt.Errorf("failed to remove host cross NAT blocking: %w", err)
	}

	for _, client := range clients {
		// Delete NAT test instances
		instances, err := utils.ListInstances(client)
		if err != nil {
			fmt.Printf("[CLEANUP] Warning: could not list instances: %v\n", err)
			continue
		}

		for _, inst := range instances {
			if strings.Contains(inst.Name, conf.VMsPrefix) {
				fmt.Printf("[CLEANUP] Deleting instance: %s\n", inst.Name)
				err := utils.DeleteInstance(client, inst.Name)
				if err != nil {
					fmt.Printf("[CLEANUP] Warning: failed to delete instance %s: %v\n", inst.Name, err)
				}
			}
		}

		// Delete NAT test networks
		networks, err := client.GetNetworkNames()
		if err != nil {
			fmt.Printf("[CLEANUP] Warning: could not list networks: %v\n", err)
			continue
		}

		for _, netName := range networks {
			if strings.HasPrefix(netName, "nat-net-") || strings.HasPrefix(netName, "private-net-") {
				fmt.Printf("[CLEANUP] Deleting network: %s\n", netName)
				err := client.DeleteNetwork(netName)
				if err != nil {
					fmt.Printf("[CLEANUP] Warning: failed to delete network %s: %v\n", netName, err)
				}
			}
		}
	}

	fmt.Println("[CLEANUP] NAT infrastructure cleanup complete")
	return nil
}
