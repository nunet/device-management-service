package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
	"golang.org/x/sync/errgroup"
)

// SetupNodes connects to Incus, spin up number of specified machines and
// uploads DMS binary to all of them. Finally, updates Context
func SetupNodes(ctx context.Context, count int) (context.Context, error) {
	config, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}

	clients, err := utils.ConnectToClients(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to incus clients: %w", err)
	}

	// cleanup all leftovers if any
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
	nodes, err := utils.CreateNodes(clients, count, utils.DefaultImage, utils.DefaultVMPrefix)
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
}

// TeardownNodes fetch all nodes from Context and destroy them
func TeardownNodes(ctx context.Context) (context.Context, error) {
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
}

// SaveLogs saves all DMS logs locally to help debugging
func SaveLogs(ctx context.Context) error {
	fmt.Println("saving logs...")
	tc := utils.NewTestCtx(ctx)

	nodes, _ := tc.Nodes()
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

		filename := fmt.Sprintf("dms_logs_node_%d.txt", i)
		path := filepath.Join(dest, filename)

		err = os.WriteFile(path, []byte(logs), 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}
