// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	repo "gitlab.com/nunet/device-management-service/db/repositories/clover"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNew(t *testing.T) {
	t.Parallel()

	rootCap := createRootCapabilityContext(t)
	cases := map[string]struct {
		rootCap             ucan.CapabilityContext
		hostID              string
		net                 network.Network
		mockResourceManager func(ctrl *gomock.Controller) types.ResourceManager
		scheduler           *bt.Scheduler
		onboarder           *onboarding.Onboarding
		geoip               types.GeoIPLocator
		hostLocation        HostGeolocation
		portConfig          PortConfig

		expErr string
	}{
		"no onboarer": {
			expErr: "onboarder is nil",
		},
		"no root capability": {
			onboarder: &onboarding.Onboarding{},
			expErr:    "root capability context is nil",
		},
		"no id": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			expErr:    "host id is nil",
		},
		"no key": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			hostID:    "123",
			expErr:    "network is nil",
		},

		"no resource manager": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			hostID:    "123",
			net:       createNetwork(t, nil, "14950"),
			expErr:    "resource manager is nil",
			mockResourceManager: func(_ *gomock.Controller) types.ResourceManager {
				return nil
			},
		},
		"no scheduler": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			hostID:    "123",
			net:       createNetwork(t, nil, "14950"),
			expErr:    "scheduler is nil",
		},
		"no geoip": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			hostID:    "123",
			net:       createNetwork(t, nil, "14950"),
			scheduler: bt.NewScheduler(1),
			expErr:    "geoip is nil",
		},
		"success": {
			onboarder: &onboarding.Onboarding{},
			rootCap:   rootCap,
			hostID:    "123",
			net:       createNetwork(t, nil, "14950"),
			scheduler: bt.NewScheduler(1),
			geoip:     &geoipMock{},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			var resourceManager types.ResourceManager
			if tt.mockResourceManager == nil {
				resourceManager = NewMockResourceManager(ctrl)
			} else {
				resourceManager = tt.mockResourceManager(ctrl)
			}

			hardwareManager := NewMockHardwareManager(ctrl)

			path, err := tempDir()
			assert.NoError(t, err)
			defer os.RemoveAll(path)

			collections := []string{"orchestrator_view"}

			db, err := repo.NewDB(path, collections)
			assert.NoError(t, err)
			assert.NotNil(t, db)

			act, err := New(
				*config.GetConfig(), afero.Afero{Fs: afero.NewMemMapFs()},
				tt.onboarder, tt.rootCap, tt.hostID, tt.net, resourceManager,
				tt.scheduler, hardwareManager, repo.NewOrchestratorView(db),
				tt.geoip, tt.hostLocation, tt.portConfig,
			)
			if tt.expErr != "" {
				assert.Nil(t, act)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, act)
				assert.NoError(t, err)
			}
		})
	}
}

func TestNodeAllocationMessaging(t *testing.T) {
	rootCap := createRootCapabilityContext(t)
	net := createNetwork(t, []multiaddr.Multiaddr{}, "14951")

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	resourceManager := NewMockResourceManager(ctrl)
	hardwareManager := NewMockHardwareManager(ctrl)

	path, err := tempDir()
	assert.NoError(t, err)
	defer os.RemoveAll(path)

	collections := []string{"orchestrator_view"}

	db, err := repo.NewDB(path, collections)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	node1, err := New(
		*config.GetConfig(), afero.Afero{Fs: afero.NewMemMapFs()},
		&onboarding.Onboarding{}, rootCap, net.Host.ID().String(), net,
		resourceManager, bt.NewScheduler(1), hardwareManager, repo.NewOrchestratorView(db),
		&geoip2.Reader{}, HostGeolocation{}, PortConfig{AvailableRangeFrom: 49152, AvailableRangeTo: 65535},
	)
	assert.NoError(t, err)
	assert.NotNil(t, node1)
	err = node1.Start()
	assert.NoError(t, err)
	require.Greater(t, len(node1.executors), 0)
	// 	alloc, err := node1.CreateAllocation(jobs.Job{ID: "123"})
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, alloc)
	// 	err = alloc.Start()
	// 	assert.NoError(t, err)

	// 	envChan := make(chan actor.Envelope)
	// 	err = node1.actor.AddBehavior("/test/ping", func(msg actor.Envelope) {
	// 		defer msg.Discard()
	// 		envChan <- msg
	// 	})
	// 	type payload struct{ Name, Type string }

	// 	assert.NoError(t, err)
	// 	msg, err := actor.Message(
	// 		alloc.Actor.Handle(),
	// 		node1.actor.Handle(),
	// 		"/test/ping",
	// 		payload{Name: "random name", Type: "x"},
	// 	)
	// 	assert.NoError(t, err)

	// 	err = alloc.Actor.Send(msg)
	// 	assert.NoError(t, err)

	// received := <-envChan
	// assert.Equal(t, string(received.Message), "{\"Name\":\"random name\",\"Type\":\"x\"}")
}

func createRootCapabilityContext(t *testing.T) ucan.CapabilityContext {
	privk, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	provider, err := did.ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")

	trustCtx := did.NewTrustContext()
	trustCtx.AddProvider(provider)

	capCtx, err := ucan.NewCapabilityContext(trustCtx, provider.DID(), nil, ucan.TokenList{}, ucan.TokenList{})
	require.NoError(t, err, "make capability context")

	return capCtx
}

func createNetwork(t *testing.T, bootstrap []multiaddr.Multiaddr, port string) *libp2p.Libp2p {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	assert.NoError(t, err)
	net, err := network.NewNetwork(&types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrap,
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               bt.NewScheduler(1),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/127.0.0.1/tcp/" + port},
			PeerCountDiscoveryLimit: 40,
		},
	}, afero.NewMemMapFs())
	assert.NoError(t, err)
	err = net.Init(&config.Config{})
	assert.NoError(t, err)

	err = net.Start()
	assert.NoError(t, err)

	libp2pInstance, _ := net.(*libp2p.Libp2p)
	return libp2pInstance
}

func tempDir() (string, error) {
	dir, err := os.MkdirTemp("", "nunet-test-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return dir, nil
}

type geoipMock struct {
	country *geoip2.Country
	city    *geoip2.City
	err     error
}

func (g *geoipMock) Country(_ net.IP) (*geoip2.Country, error) {
	return g.country, g.err
}

func (g *geoipMock) City(_ net.IP) (*geoip2.City, error) {
	return g.city, g.err
}
