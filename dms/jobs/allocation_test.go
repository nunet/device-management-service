// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/avast/retry-go"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

func TestAllocation(t *testing.T) {
	dummyRelease := func() error { return nil }
	allocationID := "allocation-1"

	withTimeout(t, "must be able to initialise the allocation", 30*time.Second, func(_ context.Context, t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		af := afero.Afero{Fs: fs}
		workDir := t.TempDir()

		_, priv1, _ := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
		rootDID, rootTrust := actor.MakeRootTrustContext(t)
		actorDID, actorTrust := actor.MakeTrustContext(t, priv1)
		actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)

		ctrl := gomock.NewController(t)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)

		testActor := actor.CreateActor(t, mockNetwork, actorCap)
		mockExecutor := NewMockExecutor(ctrl)

		allocation, err := NewAllocation(
			allocationID, jobtypes.AllocationTypeService,
			actor.Handle{}, af, workDir, testActor,
			AllocationDetails{}, mockNetwork, mockExecutor, dummyRelease,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)
	})

	withTimeout(t, "must be able to start and run the allocation then terminate it", 31*time.Second, func(_ context.Context, t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		af := afero.Afero{Fs: fs}
		workDir := t.TempDir()

		_, priv1, _ := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
		rootDID, rootTrust := actor.MakeRootTrustContext(t)
		actorDID, actorTrust := actor.MakeTrustContext(t, priv1)
		actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)

		ctrl := gomock.NewController(t)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)

		testActor := actor.CreateActor(t, mockNetwork, actorCap)
		mockExecutor := NewMockExecutor(ctrl)

		allocation, err := NewAllocation(
			allocationID, jobtypes.AllocationTypeService,
			actor.Handle{}, af, workDir, testActor,
			AllocationDetails{}, mockNetwork, mockExecutor, dummyRelease,
		)

		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockNetwork.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Start()
		require.NoError(t, err)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != AllocationRunning {
				return fmt.Errorf("allocation not running")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)

		mockNetwork.EXPECT().UnregisterMessageHandler(gomock.Any()).Return().Times(1)
		mockExecutor.EXPECT().Cancel(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockExecutor.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Terminate(context.Background())
		require.NoError(t, err)
	})

	withTimeout(t, "must be able to get the status of the allocation", 30*time.Second, func(_ context.Context, t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		af := afero.Afero{Fs: fs}
		workDir := t.TempDir()

		_, priv1, _ := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
		rootDID, rootTrust := actor.MakeRootTrustContext(t)
		actorDID, actorTrust := actor.MakeTrustContext(t, priv1)
		actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)

		ctrl := gomock.NewController(t)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)

		testActor := actor.CreateActor(t, mockNetwork, actorCap)
		mockExecutor := NewMockExecutor(ctrl)

		allocation, err := NewAllocation(
			allocationID, jobtypes.AllocationTypeService,
			actor.Handle{}, af, workDir, testActor,
			AllocationDetails{}, mockNetwork, mockExecutor, dummyRelease,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockNetwork.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Start()
		require.NoError(t, err)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != AllocationRunning {
				return fmt.Errorf("allocation not running")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)

		status := allocation.Status(context.Background())
		require.Equal(t, AllocationRunning, status.Status)
	})

	withTimeout(t, "must be able to stop the allocation", 30*time.Second, func(_ context.Context, t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		af := afero.Afero{Fs: fs}
		workDir := t.TempDir()

		_, priv1, _ := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
		rootDID, rootTrust := actor.MakeRootTrustContext(t)
		actorDID, actorTrust := actor.MakeTrustContext(t, priv1)
		actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)

		ctrl := gomock.NewController(t)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)

		testActor := actor.CreateActor(t, mockNetwork, actorCap)
		mockExecutor := NewMockExecutor(ctrl)

		allocation, err := NewAllocation(
			allocationID, jobtypes.AllocationTypeService,
			actor.Handle{}, af, workDir, testActor,
			AllocationDetails{}, mockNetwork, mockExecutor, dummyRelease,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		// ensure that the allocation is running
		err = retry.Do(func() error {
			if allocation.status != AllocationRunning {
				return fmt.Errorf("allocation not completed")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)

		mockExecutor.EXPECT().Cancel(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Stop(context.Background())
		require.NoError(t, err)

		// ensure that the allocation is stopped
		err = retry.Do(func() error {
			if allocation.status != AllocationStopped {
				return fmt.Errorf("allocation not completed")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)
	})

	withTimeout(t, "must be able to restart the allocation", 30*time.Second, func(_ context.Context, t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		af := afero.Afero{Fs: fs}
		workDir := t.TempDir()

		_, priv1, _ := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
		rootDID, rootTrust := actor.MakeRootTrustContext(t)
		actorDID, actorTrust := actor.MakeTrustContext(t, priv1)
		actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)

		ctrl := gomock.NewController(t)
		mockNetwork := NewMockNetwork(ctrl)
		mockNetwork.EXPECT().GetHostID().Return(libp2p.PeerID("hostID")).Times(1)

		testActor := actor.CreateActor(t, mockNetwork, actorCap)
		mockExecutor := NewMockExecutor(ctrl)

		allocation, err := NewAllocation(
			allocationID, jobtypes.AllocationTypeService,
			actor.Handle{}, af, workDir, testActor,
			AllocationDetails{}, mockNetwork, mockExecutor, dummyRelease,
		)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != AllocationRunning {
				return fmt.Errorf("allocation not running")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)

		allocation.state.subnetIP = "dummy-port"
		mockNetwork.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockExecutor.EXPECT().Cancel(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err = allocation.Restart(context.Background())
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != AllocationRunning {
				return fmt.Errorf("allocation not running")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)
	})
}

// Export this from somewhere as a helper (used also by the orchestrator)
func withTimeout(t *testing.T, name string, duration time.Duration, testFunc func(ctx context.Context, t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		t.Helper()

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		done := make(chan struct{})
		go func() {
			testFunc(ctx, t)
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			t.Fatalf("test %q timed out after %s", name, duration)
		}
	})
}
