// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hooks

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	dmsnode "gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
	"golang.org/x/sync/errgroup"
)

// SetupNodes connects to Incus, spin up number of specified machines and
// uploads DMS binary to all of them.
func SetupNodes(count int) ([]*utils.Node, error) {
	config, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}

	clients, err := utils.ConnectToClients(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clients: %w", err)
	}

	start := time.Now()
	fmt.Println("creating nodes...")
	nodes, err := utils.CreateNodes(clients, count, config.VMsPrefix)
	if err != nil {
		return nil, err
	}

	here := dutils.CurrentFileDirectory()
	remoteDMSPath := "/usr/local/bin/nunet"
	localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")

	g := new(errgroup.Group)
	for _, node := range nodes {
		g.Go(func() error {
			if err := node.WaitForInstanceReady(); err != nil {
				return fmt.Errorf("instance %s not ready: %w", node.Name, err)
			}
			if err := node.UploadFile(localPath, remoteDMSPath, 0o755); err != nil {
				return fmt.Errorf("failed to upload file to node %s: %w", node.Name, err)
			}

			if _, err := node.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"}); err != nil {
				return fmt.Errorf("failed to make dms executable at node %s: %w", node.Name, err)
			}
			if err := node.ConfigureVMNetworkingForQUIC(); err != nil {
				return fmt.Errorf("failed to configure VM networking for QUIC on %s: %w", node.Name, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	fmt.Printf("finished setting up nodes, time elapsed: %.1fs\n", time.Since(start).Seconds())

	return nodes, nil
}

// CleanupNodes connects to all Incus clients and remove any leftover instance
// Recommended to call Before and After a scenario. When a test fails, the After
// hook is not called, so only Before can clean it up.
func CleanupNodes() error {
	config, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}

	clients, err := utils.ConnectToClients(config)
	if err != nil {
		return fmt.Errorf("failed to connect to clients: %w", err)
	}

	// cleanup all leftovers if any
	start := time.Now()
	fmt.Println("cleaning up instances")
	for _, c := range clients {
		instances, err := utils.ListInstances(c)
		if err != nil {
			return fmt.Errorf("could not list instances: %w", err)
		}
		if len(instances) == 0 {
			continue
		}
		for _, i := range instances {
			if strings.HasPrefix(i.Name, config.VMsPrefix) {
				err := utils.DeleteInstance(c, i.Name)
				if err != nil {
					return fmt.Errorf("failed to delete instance %s: %w", i.Name, err)
				}
			}
		}
	}
	fmt.Printf("finished cleaning up. time elapsed: %.1fs\n", time.Since(start).Seconds())
	return nil
}

// SaveLogs saves all DMS logs locally to help debugging
func SaveLogs(ctx context.Context) error {
	fmt.Println("saving logs...")
	tc := utils.NewTestCtx(ctx)
	t := godog.T(ctx)

	timestamp := time.Now().Unix()
	nodes, _ := tc.Nodes()
	if len(nodes) == 0 {
		nm, _ := tc.NodeMap()
		nodes = slices.Collect(maps.Values(nm))
	}

	for i, n := range nodes {
		// init destination
		dest := filepath.Join(dutils.CurrentFileDirectory(), "..", "tests", "acceptance", "testdata", "logs")
		err := os.MkdirAll(dest, 0o755)
		if err != nil {
			return err
		}

		// terminal output
		logs, err := n.RunCMD([]string{"cat", "dms-logs.txt"})
		if err != nil {
			t.Errorf("no stdout for %d", i)
			continue
		}

		filename := fmt.Sprintf("dms_logs_node_%d-%d.txt", i, timestamp)
		path := filepath.Join(dest, filename)

		err = os.WriteFile(path, []byte(logs), 0o644)
		if err != nil {
			return err
		}

		// JSONL logs
		src := "/root/nunet/logs/nunet-dms-logs.jsonl"
		target := filepath.Join(dest, fmt.Sprintf("dms_logs_node_%d-%d.jsonl", i, timestamp))
		if err := utils.DownloadFile(n.Client, n.Name, src, target); err != nil {
			t.Errorf("failed to download jsonl logs for %d: %s", i, err)
		}

		// flight recorder
		src = "/root/nunet/logs/flightrec.trace"
		names := []string{"alice", "bob", "charlie"}
		if os.Getenv(dmsnode.EnvFlightrecSec) != "" {

			// create dump and download
			// TODO dont guess, reduce non-determinism (with unified file/ctx names?)
			var nodeName string
			for _, name := range names {
				if _, err = n.RunDMSCmd(fmt.Sprintf("nunet -c %s-dms actor cmd /dms/debug/flightrec", name)); err == nil {
					t.Logf("created flight recorder dump for %s (%d)", name, i)
					nodeName = name
					break
				}
			}
			if nodeName == "" {
				t.Errorf("failed to create flight recorder dump %d: %s", i, err)
			}

			target = filepath.Join(dest, fmt.Sprintf("flightrec_%d-%d.%s.trace", i, timestamp, nodeName))

			// download
			if err := utils.DownloadFile(n.Client, n.Name, src, target); err != nil {
				t.Errorf("failed to download flight recorder dump %d: %s", i, err)
			}
		}
	}
	return nil
}
