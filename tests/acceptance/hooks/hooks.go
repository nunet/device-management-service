package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
	"golang.org/x/exp/maps"
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
		return nil, fmt.Errorf("failed to connect to incus clients: %w", err)
	}

	start := time.Now()
	fmt.Println("creating nodes...")
	nodes, err := utils.CreateNodes(clients, count, utils.DefaultImage, config.VMsPrefix)
	if err != nil {
		return nil, err
	}

	here := dutils.CurrentFileDirectory()
	remoteDMSPath := "/usr/local/bin/nunet"
	localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")
	for idx, n := range nodes {
		err := n.WaitForInstanceReady()
		if err != nil {
			return nil, fmt.Errorf("instance not ready: %w", err)
		}
		err = n.UploadFile(localPath, remoteDMSPath, 0o755)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to node %d: %w", idx, err)
		}

		_, err = n.RunCMD([]string{"chmod", "+x", "/usr/local/bin/nunet"})
		if err != nil {
			return nil, fmt.Errorf("failed to make dms executable at node %d: %w", idx, err)
		}
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
		return fmt.Errorf("failed to connect to incus clients: %w", err)
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

	timestamp := time.Now().Unix()
	nodes, _ := tc.Nodes()
	if len(nodes) == 0 {
		nm, _ := tc.NodeMap()
		nodes = maps.Values(nm)
	}

	for i, n := range nodes {
		logs, err := n.RunCMD([]string{"cat", "dms-logs.txt"})
		if err != nil {
			continue
		}

		dest := filepath.Join(dutils.CurrentFileDirectory(), "..", "tests", "acceptance", "testdata")
		err = os.MkdirAll(dest, 0o755)
		if err != nil {
			return err
		}

		filename := fmt.Sprintf("dms_logs_node_%d-%d.txt", i, timestamp)
		path := filepath.Join(dest, filename)

		err = os.WriteFile(path, []byte(logs), 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}
