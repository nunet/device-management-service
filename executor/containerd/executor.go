// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

var ErrNotInstalled = fmt.Errorf("containerd is not installed")

type Executor struct {
	ID string

	containerdClient *client.Client
	cfg              config.Containerd
	networkMgr       *networkManager

	executions utils.SyncMap[string, *executionState]
}

var _ types.Executor = (*Executor)(nil)

func NewExecutor(ctx context.Context, id string, cfg config.Containerd) (*Executor, error) {
	cli, err := client.New(cfg.SocketPath)
	if err != nil {
		return nil, err
	}

	e := &Executor{
		ID:               id,
		containerdClient: cli,
		cfg:              cfg,
	}

	if !e.IsAvailable(ctx) {
		return nil, ErrNotInstalled
	}

	netMgr, err := newNetworkManager(cfg)
	if err != nil {
		log.Warnw("containerd CNI network manager unavailable - skipping network config",
			"error", err,
			"config", filepath.Join(cfg.CNINetConfDir, cfg.CNINetworkName+".conflist"),
			"plugins", cfg.CNIPluginDir,
		)
	} else {
		e.networkMgr = netMgr
	}

	return e, nil
}

func (e *Executor) GetID() string {
	return e.ID
}

func (e *Executor) IsAvailable(_ context.Context) bool {
	if _, err := exec.LookPath("containerd"); err != nil {
		return false
	}

	if _, err := exec.LookPath("containerd-shim-runc-v2"); err != nil {
		return false
	}

	if _, err := os.Stat(e.cfg.ConfigPath); err != nil {
		return false
	}

	if err := checkContainerdSocketAccess(e.cfg.SocketPath); err != nil {
		return false
	}

	return true
}

func checkContainerdSocketAccess(socketPath string) error {
	info, err := os.Stat(socketPath)
	if err != nil {
		return fmt.Errorf("containerd socket %q: %w", socketPath, err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	if stat.Uid == 0 && os.Geteuid() != 0 {
		return fmt.Errorf(
			"containerd socket %q is owned by root; run DMS as root to use the containerd executor",
			socketPath,
		)
	}

	return nil
}

func (e *Executor) Start(ctx context.Context, request *types.ExecutionRequest) error {
	if request == nil {
		return fmt.Errorf("execution request cannot be nil")
	}

	if request.ExecutionID == "" {
		return fmt.Errorf("execution ID cannot be empty")
	}

	if st, found := e.executions.Get(request.ExecutionID); found {
		if st.running.Load() {
			return fmt.Errorf("execution is already started")
		}
		return fmt.Errorf("execution is already completed")
	}

	spec, err := DecodeSpec(request.EngineSpec)
	if err != nil {
		return fmt.Errorf("failed to decode containerd engine spec: %w", err)
	}

	nsCtx := e.withNamespace(ctx)

	image, err := e.containerdClient.GetImage(nsCtx, spec.Image)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to get image %q: %w", spec.Image, err)
		}
		image, err = e.containerdClient.Pull(nsCtx, spec.Image, client.WithPullUnpack)
		if err != nil {
			return fmt.Errorf("failed to pull image %q: %w", spec.Image, err)
		}
	}

	containerID := request.ExecutionID
	snapshotID := request.ExecutionID + "-snapshot"

	var netSetup *networkSetup
	if len(request.PortsToBind) > 0 {
		if e.networkMgr == nil {
			return fmt.Errorf("unable to do port forwarding, networking requires CNI")
		}
		netSetup, err = e.networkMgr.setup(nsCtx, containerID, request.PortsToBind)
		if err != nil {
			return fmt.Errorf("failed to setup CNI network for execution %q: %w", request.ExecutionID, err)
		}
	}

	args := commandArgs(spec.Entrypoint, spec.Cmd)
	specOpts := []oci.SpecOpts{oci.WithImageConfig(image)}
	if len(args) > 0 {
		specOpts = append(specOpts, oci.WithProcessArgs(args...))
	}
	if len(spec.Environment) > 0 {
		specOpts = append(specOpts, oci.WithEnv(spec.Environment))
	}
	if spec.WorkingDirectory != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(spec.WorkingDirectory))
	}
	if netSetup != nil {
		specOpts = append(specOpts, oci.WithLinuxNamespace(specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: netSetup.netNS.GetPath(),
		}))
	}

	mounts, err := makeMounts(request.Inputs, request.Outputs, request.ResultsDir)
	if err != nil {
		if netSetup != nil && e.networkMgr != nil {
			_ = e.networkMgr.teardown(nsCtx, netSetup)
		}
		return fmt.Errorf("failed to create container mounts: %w", err)
	}
	if len(mounts) > 0 {
		specOpts = append(specOpts, oci.WithMounts(mounts))
	}

	containerOpts := []client.NewContainerOpts{
		client.WithImage(image),
		client.WithNewSnapshot(snapshotID, image),
		client.WithNewSpec(specOpts...),
		client.WithRuntime(spec.RuntimeName(), nil),
	}

	container, err := e.containerdClient.NewContainer(nsCtx, containerID, containerOpts...)
	if err != nil {
		if netSetup != nil && e.networkMgr != nil {
			_ = e.networkMgr.teardown(nsCtx, netSetup)
		}
		return fmt.Errorf("failed to create container %q: %w", containerID, err)
	}

	state := &executionState{
		executionID:         request.ExecutionID,
		container:           container,
		network:             netSetup,
		image:               spec.Image,
		persistLogsDuration: request.PersistLogsDuration,
		running:             &atomic.Bool{},
		doneCh:              make(chan struct{}),
	}

	cleanup := func() {
		state.closeLogFiles()
		if state.network != nil && e.networkMgr != nil {
			_ = e.networkMgr.teardown(nsCtx, state.network)
			state.network = nil
		}
		_ = container.Delete(nsCtx, client.WithSnapshotCleanup)
	}

	if request.ResultsDir != "" {
		if err := state.openLogFiles(request.ResultsDir); err != nil {
			cleanup()
			return fmt.Errorf("prepare execution log files: %w", err)
		}
	}

	stdoutW, stderrW := state.logWriters()

	task, err := container.NewTask(nsCtx, cio.NewCreator(cio.WithStreams(nil, stdoutW, stderrW)))
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to create task for execution %q: %w", request.ExecutionID, err)
	}

	waitCh, err := task.Wait(nsCtx)
	if err != nil {
		_, _ = task.Delete(nsCtx, client.WithProcessKill)
		cleanup()
		return fmt.Errorf("failed to wait on task for execution %q: %w", request.ExecutionID, err)
	}

	if err := task.Start(nsCtx); err != nil {
		_, _ = task.Delete(nsCtx, client.WithProcessKill)
		cleanup()
		return fmt.Errorf("failed to start task for execution %q: %w", request.ExecutionID, err)
	}

	state.task = task
	state.running.Store(true)
	e.executions.Put(request.ExecutionID, state)

	go func(executionID string, st *executionState) {
		defer close(st.doneCh)

		finish := func(result *types.ExecutionResult) {
			st.running.Store(false)
			st.closeLogFiles()
			if result != nil && result.STDOUT == "" && result.STDERR == "" {
				result.STDOUT, result.STDERR = st.readLogs()
			}
			st.setResult(result)
			st.scheduleLogDeletion(st.persistLogsDuration)
		}

		status, ok := <-waitCh
		if !ok {
			finish(types.NewFailedExecutionResult(fmt.Errorf("execution (%s) wait channel closed", executionID)))
			return
		}

		exitCode, _, err := status.Result()
		if err != nil {
			finish(types.NewFailedExecutionResult(fmt.Errorf("execution (%s) wait failed: %w", executionID, err)))
			return
		}

		finish(&types.ExecutionResult{
			ExitCode: int(exitCode),
		})
	}(request.ExecutionID, state)
	return nil
}

func (e *Executor) Run(ctx context.Context, request *types.ExecutionRequest) (*types.ExecutionResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}

	resultCh, errCh := e.Wait(ctx, request.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	}
}

func (e *Executor) Pause(_ context.Context, _ string) error {
	return fmt.Errorf("pause not supported by containerd executor yet")
}

func (e *Executor) Resume(_ context.Context, _ string) error {
	return fmt.Errorf("resume not supported by containerd executor yet")
}

func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *types.ExecutionResult, <-chan error) {
	resultCh := make(chan *types.ExecutionResult, 1)
	errCh := make(chan error, 1)

	state, found := e.executions.Get(executionID)
	if !found {
		errCh <- fmt.Errorf("execution (%s) not found", executionID)
		close(resultCh)
		close(errCh)
		return resultCh, errCh
	}

	go func() {
		defer close(resultCh)
		defer close(errCh)

		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
		case <-state.doneCh:
			result := state.getResult()
			if result == nil {
				errCh <- fmt.Errorf("execution (%s) result is nil", executionID)
				return
			}
			resultCh <- result
		}
	}()

	return resultCh, errCh
}

func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	state, found := e.executions.Get(executionID)
	if !found {
		return fmt.Errorf("failed to cancel execution (%s). execution not found", executionID)
	}

	if !state.running.Load() {
		return nil
	}

	if err := state.task.Kill(e.withNamespace(ctx), syscall.SIGTERM); err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to cancel execution (%s): %w", executionID, err)
	}

	return nil
}

func (e *Executor) Remove(executionID string, timeout time.Duration) error {
	state, found := e.executions.Get(executionID)
	if !found {
		return fmt.Errorf("execution (%s) not found", executionID)
	}

	ctx := e.withNamespace(context.Background())
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if state.running.Load() {
		_ = state.task.Kill(ctx, syscall.SIGKILL)
	}

	if _, err := state.task.Delete(ctx, client.WithProcessKill); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to delete task for execution (%s): %w", executionID, err)
	}

	if err := state.container.Delete(ctx, client.WithSnapshotCleanup); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to delete container for execution (%s): %w", executionID, err)
	}

	if state.network != nil && e.networkMgr != nil {
		if err := e.networkMgr.teardown(ctx, state.network); err != nil {
			return fmt.Errorf("failed to teardown network for execution (%s): %w", executionID, err)
		}
		state.network = nil
	}

	e.executions.Delete(executionID)
	return nil
}

func (e *Executor) Cleanup(_ context.Context) error {
	var errs []error
	e.executions.Iter(func(executionID string, _ *executionState) bool {
		if err := e.Remove(executionID, 15*time.Second); err != nil {
			errs = append(errs, err)
		}
		return true
	})

	if err := e.containerdClient.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	var sb strings.Builder
	for i, err := range errs {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("containerd cleanup failed: %s", sb.String())
}

func (e *Executor) GetLogStream(_ context.Context, request types.LogStreamRequest) (io.ReadCloser, error) {
	state, found := e.executions.Get(request.ExecutionID)
	if !found {
		return nil, fmt.Errorf("execution (%s) not found", request.ExecutionID)
	}

	combined := state.combinedLogs()
	return io.NopCloser(strings.NewReader(combined)), nil
}

func (e *Executor) List() []types.ExecutionListItem {
	items := make([]types.ExecutionListItem, 0)
	e.executions.Iter(func(executionID string, state *executionState) bool {
		items = append(items, types.ExecutionListItem{
			ExecutionID: executionID,
			Running:     state.running.Load(),
		})
		return true
	})
	return items
}

func (e *Executor) GetStatus(ctx context.Context, executionID string) (types.ExecutionStatus, error) {
	state, found := e.executions.Get(executionID)
	if !found {
		return "", fmt.Errorf("execution (%s) not found", executionID)
	}

	if !state.running.Load() {
		if result := state.getResult(); result != nil {
			if result.ExitCode == types.ExecutionStatusCodeSuccess {
				return types.ExecutionStatusSuccess, nil
			}
			return types.ExecutionStatusFailed, nil
		}
		return types.ExecutionStatusPending, nil
	}

	status, err := state.task.Status(e.withNamespace(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get task status for execution (%s): %w", executionID, err)
	}

	statusString := strings.ToLower(string(status.Status))
	switch statusString {
	case "created":
		return types.ExecutionStatusPending, nil
	case "running":
		return types.ExecutionStatusRunning, nil
	case "paused", "pausing":
		return types.ExecutionStatusPaused, nil
	case "stopped":
		if result := state.getResult(); result != nil && result.ExitCode == types.ExecutionStatusCodeSuccess {
			return types.ExecutionStatusSuccess, nil
		}
		return types.ExecutionStatusFailed, nil
	default:
		return "", fmt.Errorf("unknown containerd task status: %s", string(status.Status))
	}
}

func (e *Executor) WaitForStatus(
	ctx context.Context,
	executionID string,
	status types.ExecutionStatus,
	timeout *time.Duration,
) error {
	waitTimeout := 10 * time.Second
	if timeout != nil {
		waitTimeout = *timeout
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("execution (%s) did not reach status %s", executionID, status)
		case <-ticker.C:
			s, err := e.GetStatus(ctx, executionID)
			if err != nil {
				return err
			}
			if s == status {
				return nil
			}
		}
	}
}

func (e *Executor) Exec(_ context.Context, _ string, _ []string) (int, string, string, error) {
	return 0, "", "", fmt.Errorf("exec not supported by containerd executor yet")
}

func (e *Executor) Stats(_ context.Context, _ string) (*types.ExecutorStats, error) {
	return nil, fmt.Errorf("stats not supported by containerd executor yet")
}

func (e *Executor) GetInfo(ctx context.Context, executionID string) (*types.ExecutorInfo, error) {
	state, found := e.executions.Get(executionID)
	if !found {
		return nil, fmt.Errorf("execution (%s) not found", executionID)
	}

	status, err := e.GetStatus(ctx, executionID)
	if err != nil {
		return nil, err
	}

	netInfo, err := e.GetNetInfo(ctx, executionID)
	if err != nil {
		return nil, err
	}

	return &types.ExecutorInfo{
		ExecutionID: executionID,
		ContainerID: executionID,
		Image:       state.image,
		Runtime:     types.ExecutorTypeContainerd,
		Status:      status,
		Net:         *netInfo,
	}, nil
}

func (e *Executor) GetNetInfo(_ context.Context, executionID string) (*types.ExecutorNetInfo, error) {
	state, found := e.executions.Get(executionID)
	if !found {
		return nil, fmt.Errorf("execution (%s) not found", executionID)
	}

	if state.network == nil {
		return &types.ExecutorNetInfo{}, nil
	}

	netInfo := state.network.netInfo
	return &netInfo, nil
}

func commandArgs(entrypoint, cmd []string) []string {
	args := make([]string, 0, len(entrypoint)+len(cmd))
	args = append(args, entrypoint...)
	args = append(args, cmd...)
	return args
}

func (e *Executor) withNamespace(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return namespaces.WithNamespace(ctx, e.cfg.Namespace)
}

func IsShimAvailable(runtime string) bool {
	runtime = strings.TrimSpace(strings.ToLower(runtime))
	switch runtime {
	case "", "runc", DefaultRuntime:
		_, err := exec.LookPath("containerd-shim-runc-v2")
		return err == nil
	case "kata", KataRuntime:
		_, err := exec.LookPath("containerd-shim-kata-v2")
		return err == nil
	default:
		// Runtime names follow io.containerd.<shim>.v2 convention.
		if strings.HasPrefix(runtime, "io.containerd.") && strings.HasSuffix(runtime, ".v2") {
			shimName := strings.TrimPrefix(runtime, "io.containerd.")
			shimName = strings.TrimSuffix(shimName, ".v2")
			_, err := exec.LookPath(filepath.Clean("containerd-shim-" + shimName + "-v2"))
			return err == nil
		}
		return false
	}
}
