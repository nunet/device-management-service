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

	"gitlab.com/nunet/device-management-service/network/libp2p"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"go.uber.org/mock/gomock"
)

func TestAllocation(t *testing.T) {
	t.Parallel()

	allocationID := "allocation-1"

	t.Run("must be able to initialise the allocation", func(t *testing.T) {
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

		allocation, err := NewAllocation(allocationID, af, workDir, testActor, AllocationDetails{}, mockNetwork, mockExecutor)
		require.NoError(t, err)
		require.NotNil(t, allocation)
	})

	t.Run("must be able to start and run the allocation then terminate it", func(t *testing.T) {
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

		allocation, err := NewAllocation(allocationID, af, workDir, testActor, AllocationDetails{}, mockNetwork, mockExecutor)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockNetwork.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Start()
		require.NoError(t, err)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != Running {
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

	t.Run("must be able to get the status of the allocation", func(t *testing.T) {
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

		allocation, err := NewAllocation(allocationID, af, workDir, testActor, AllocationDetails{}, mockNetwork, mockExecutor)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockNetwork.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Start()
		require.NoError(t, err)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != Running {
				return fmt.Errorf("allocation not running")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)

		status := allocation.Status(context.Background())
		require.Equal(t, Running, status.Status)
	})

	t.Run("must be able to stop the allocation", func(t *testing.T) {
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

		allocation, err := NewAllocation(allocationID, af, workDir, testActor, AllocationDetails{}, mockNetwork, mockExecutor)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		// ensure that the allocation is running
		err = retry.Do(func() error {
			if allocation.status != Running {
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
			if allocation.status != Stopped {
				return fmt.Errorf("allocation not completed")
			}
			return nil
		},
			retry.Attempts(5),
			retry.Delay(1*time.Second),
		)
		require.NoError(t, err)
	})

	t.Run("must be able to restart the allocation", func(t *testing.T) {
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

		allocation, err := NewAllocation(allocationID, af, workDir, testActor, AllocationDetails{}, mockNetwork, mockExecutor)
		require.NoError(t, err)
		require.NotNil(t, allocation)

		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Run(context.Background(), "", "", nil)
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != Running {
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
		mockExecutor.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockExecutor.EXPECT().Cancel(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		err = allocation.Restart(context.Background())
		require.NoError(t, err)

		err = retry.Do(func() error {
			if allocation.status != Running {
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
