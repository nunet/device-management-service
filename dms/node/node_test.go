package node

import (
	"context"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
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
		rootCap         ucan.CapabilityContext
		hostID          string
		net             network.Network
		resourceManager resources.Manager
		scheduler       *bt.Scheduler
		onboarder       *onboarding.Onboarding

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
		},
		"no scheduler": {
			onboarder:       &onboarding.Onboarding{},
			rootCap:         rootCap,
			hostID:          "123",
			net:             createNetwork(t, nil, "14950"),
			resourceManager: &resourceManagerMock{},
			expErr:          "scheduler is nil",
		},
		"success": {
			onboarder:       &onboarding.Onboarding{},
			rootCap:         rootCap,
			hostID:          "123",
			net:             createNetwork(t, nil, "14950"),
			resourceManager: &resourceManagerMock{},
			scheduler:       bt.NewScheduler(1),
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			act, err := New(context.TODO(), tt.onboarder, tt.rootCap, tt.hostID, tt.net, tt.resourceManager, tt.scheduler)
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

	node1, err := New(context.TODO(), &onboarding.Onboarding{}, rootCap, net.Host.ID().String(), net, &resourceManagerMock{}, bt.NewScheduler(1))
	assert.NoError(t, err)
	assert.NotNil(t, node1)
	err = node1.Start()
	assert.NoError(t, err)

	alloc, err := node1.CreateAllocation(jobs.Job{ID: "123"})
	assert.NoError(t, err)
	assert.NotNil(t, alloc)
	err = alloc.Start()
	assert.NoError(t, err)

	envChan := make(chan actor.Envelope)
	err = node1.actor.AddBehavior("/test/ping", func(msg actor.Envelope) {
		defer msg.Discard()
		envChan <- msg
	})
	type payload struct{ Name, Type string }

	assert.NoError(t, err)
	msg, err := actor.Message(
		alloc.Actor.Handle(),
		node1.actor.Handle(),
		"/test/ping",
		payload{Name: "random name", Type: "x"},
	)
	assert.NoError(t, err)

	err = alloc.Actor.Send(msg)
	assert.NoError(t, err)

	received := <-envChan
	assert.Equal(t, string(received.Message), "{\"Name\":\"random name\",\"Type\":\"x\"}")
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
	err = net.Init(context.Background())
	assert.NoError(t, err)

	err = net.Start(context.Background())
	assert.NoError(t, err)

	libp2pInstance, _ := net.(*libp2p.Libp2p)
	return libp2pInstance
}

type resourceManagerMock struct {
	freeRes            types.FreeResources
	onboardedResources types.OnboardedResources

	err error
}

func (m *resourceManagerMock) UpdateFreeResources(context.Context) (types.FreeResources, error) {
	return m.freeRes, m.err
}

func (m *resourceManagerMock) UpdateOnboardedResources(context.Context, types.OnboardedResources) error {
	return m.err
}

func (m *resourceManagerMock) GetOnboardedResources(context.Context) (types.OnboardedResources, error) {
	return m.onboardedResources, m.err
}

func (m *resourceManagerMock) GetRequiredResources(context.Context) (types.Resources, error) {
	return m.freeRes.Resources, m.err
}

func (m *resourceManagerMock) SystemSpecs() resources.SystemSpecs {
	return nil
}

func (m *resourceManagerMock) UsageMonitor() resources.UsageMonitor {
	return nil
}
