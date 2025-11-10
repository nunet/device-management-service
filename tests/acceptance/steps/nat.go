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
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
	"golang.org/x/sync/errgroup"
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
	ctx.Step(`^"([^"]*)" attempts to connect to "([^"]*)"$`, nodeAttemptsToConnect)
	ctx.Step(`^the connection should fail if no relay$`, connectionFailIfNoRelay)
	ctx.Step(`^Target "([^"]*)" waits to perform AutoNAT and obtain relay circuits$`, waitForRelayCircuits)
	ctx.Step(`^"([^"]*)" should have a relay address advertised$`, nodeShouldHaveRelayAddress)
	ctx.Step(`^the connection should succeed via relay$`, connectionShouldSucceed)
}

type natNodeInfo struct {
	name    string
	network string
}

func haveNodesOnIsolatedNATNetworks(ctx context.Context, count int, table *godog.Table) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap := make(map[string]*utils.Node, count)

	natNodes, err := parseNATNodesTable(table)
	assert.NoError(t, err)
	assert.Len(t, natNodes, count)

	subnets := []string{
		fmt.Sprintf("10.%d.100.1/24", rand.Intn(250)+2),
		fmt.Sprintf("10.%d.90.1/24", rand.Intn(250)+2),
	}

	instanceIPs := make(map[string]string)
	instanceNets := make([]string, 0, len(instanceIPs))

	config, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}

	clients, err := utils.ConnectToClients(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clients: %w", err)
	}

	// Create network then instances with the network
	fmt.Println("creating instances with networks...")
	start := time.Now()
	g := new(errgroup.Group)
	for idx, nodeInfo := range natNodes {
		g.Go(func() error {
			client := clients[idx%len(clients)]
			fmt.Println("spinning up instances...")
			instName := fmt.Sprintf("%s-nat-test-%d", config.VMsPrefix, idx+1)
			err := utils.CreateInstanceWithNetwork(
				client,
				utils.GetInstanceType(),
				instName,
				nodeInfo.network,
				subnets[idx],
			)
			if err != nil {
				return err
			}

			instance := &utils.Instance{
				Name:     instName,
				Client:   client,
				Contexts: make(map[string]*utils.Context),
			}

			node := &utils.Node{
				Name:      nodeInfo.name,
				Role:      "SP",
				Org:       "none",
				Onboarded: false,
				Instance:  instance,
			}

			here := dutils.CurrentFileDirectory()
			remoteDMSPath := "/usr/local/bin/nunet"
			localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")

			if err := instance.WaitForInstanceReady(); err != nil {
				return fmt.Errorf("instance %s not ready: %w", node.Name, err)
			}
			if err := instance.UploadFile(localPath, remoteDMSPath, 0o755); err != nil {
				return fmt.Errorf("failed to upload file to node %s: %w", node.Name, err)
			}

			if _, err := instance.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"}); err != nil {
				return fmt.Errorf("failed to make dms executable at node %s: %w", node.Name, err)
			}

			err = node.InitialCaps()
			if err != nil {
				return err
			}

			if err := node.DMS().Run(t); err != nil {
				return err
			}

			assert.Eventually(t, func() bool {
				return instance.IsDMSRunning(9999)
			}, 20*time.Second, 500*time.Millisecond)

			nodeMap[nodeInfo.name] = node
			return nil
		})

		instanceIPs[nodeInfo.name] = subnets[idx]
		instanceNets = append(instanceNets, nodeInfo.network)

	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	fmt.Printf("finished setting up instances, time elapsed: %.1fs\n", time.Since(start).Seconds())

	err = utils.BlockInterInstanceTraffic(clients[0], fmt.Sprintf("%s-block-inter-instance", config.ACLPrefix), instanceIPs, instanceNets)
	if err != nil {
		return nil, fmt.Errorf("failed to block inter-instance traffic: %w", err)
	}

	fmt.Printf("finished adding acl to block inter-instance traffic\n")

	tc = tc.WithNodes(nodeMap)

	return tc.Unwrap(), nil
}

func waitForRelayCircuits(ctx context.Context, targetNode string) (context.Context, error) {
	fmt.Printf("[RELAY] Waiting for %s to perform AutoNAT detection and obtain relay circuits\n", targetNode)

	// Wait for relay circuits to establish
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.Nodes()
	assert.NoError(t, err)

	_, DMSCtx := utils.NodeWithDMS(nodeMap, targetNode)
	assert.NotNil(t, DMSCtx, fmt.Sprintf("Target node %s not found", targetNode))

	tCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		select {
		case <-tCtx.Done():
			return ctx, fmt.Errorf("timeout waiting for node %s to obtain relay address", targetNode)
		default:
		}

		time.Sleep(5 * time.Second)
		targetInfo, err := DMSCtx.PeerAddr()
		assert.NoError(t, err)
		assert.NotNil(t, targetInfo)

		if strings.Contains(targetInfo.Address, "p2p-circuit") {
			fmt.Printf("[RELAY] Node %s has obtained relay address: %s\n", targetNode, targetInfo.Address)
			return ctx, nil
		}

		fmt.Printf("[RELAY] Node %s has not yet obtained relay address, retrying...\n", targetNode)
	}
}

func nodeShouldHaveRelayAddress(ctx context.Context, nodeName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.Nodes()
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

	nodeMap, err := tc.Nodes()
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

	fmt.Printf("  CONN ATTEMPT: %+v\n", connectionAttempt)

	return tc.WithConnectionAttempt(connectionAttempt).Unwrap(), nil
}

func connectionFailIfNoRelay(ctx context.Context) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodeMap, err := tc.Nodes()
	assert.NoError(t, err)

	attempt, err := tc.ConnectionAttempt()
	assert.NoError(t, err)
	assert.NotNil(t, attempt, "Connection attempt should be recorded")

	targetNode := attempt["target"].(string)
	connErr, _ := attempt["error"].(error)

	_, targetDmsCtx := utils.NodeWithDMS(nodeMap, targetNode)
	assert.NotNil(t, targetDmsCtx, fmt.Sprintf("Target node %s not found", targetNode))

	targetInfo, err := targetDmsCtx.PeerAddr()
	assert.NoError(t, err)
	assert.NotNil(t, targetInfo)

	targetAddr, err := utils.MultiaddrFromCLI(targetInfo)
	assert.NoError(t, err)
	assert.NotEmpty(t, targetAddr, "Target multiaddr should not be empty")

	fmt.Printf("target listening addr: %+v\n", targetAddr)

	hasRelay := strings.Contains(targetAddr, "p2p-circuit")

	if connErr == nil {
		if hasRelay {
			fmt.Printf("Relay obtained quickly - connection succeeded on first try\n")
			return nil
		}
		fmt.Printf("Connection succeeded without relay - unexpected\n")
		return fmt.Errorf("expected connection to fail without relay, but it succeeded")
	}

	fmt.Printf("Connection failed: %v\n", connErr)

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

		// delete acls
		err = utils.DeleteACL(client, fmt.Sprintf("%s-block-inter-instance", conf.ACLPrefix))
		if err != nil {
			fmt.Printf("[CLEANUP] Warning: failed to delete ACL %s: %v\n", fmt.Sprintf("%s-block-inter-instance", conf.ACLPrefix), err)
		}
	}

	fmt.Println("[CLEANUP] NAT infrastructure cleanup complete")
	return nil
}
