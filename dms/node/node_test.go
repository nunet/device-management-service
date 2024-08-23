package node

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/resources"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		id              string
		net             network.Network
		benchmarker     benchmarker
		resourceManager resources.Manager

		expErr string
	}{
		"no id": {
			expErr: "id is nil",
		},
		"no network": {
			id:     "123",
			expErr: "network is nil",
		},
		"no benchmarker": {
			id:     "123",
			net:    &libp2p.Libp2p{},
			expErr: "benchmarker is nil",
		},
		"no resource manager": {
			id:          "123",
			net:         createNetwork(t, nil, "14950"),
			benchmarker: &benchmarkerStub{},
			expErr:      "resource manager is nil",
		},
		"success": {
			id:              "123",
			net:             createNetwork(t, nil, "14950"),
			benchmarker:     &benchmarkerStub{},
			resourceManager: &resourceManagerMock{},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			act, err := New(context.TODO(), tt.id, tt.net, tt.benchmarker, tt.resourceManager)
			if tt.expErr != "" {
				assert.Nil(t, act)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, act)
			}
		})
	}
}

func TestNodeSendMessage(t *testing.T) {
	net := createNetwork(t, nil, "14951")
	node, err := New(context.TODO(), net.Host.ID().String(), net, &benchmarkerStub{}, &resourceManagerMock{})
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// send a message to itself
	err = node.SendMessage(context.TODO(), node.actor.Address(), &dms.Message{
		Type:   "GenericAction1",
		Sender: node.actor.Address().InboxAddress,
		Data:   []byte("nunet actor"),
	})
	assert.NoError(t, err)

	msg1 := <-node.actor.Messages()
	assert.Equal(t, "nunet actor", string(msg1.Data))

	err = node.SendMessage(context.TODO(), node.actor.Address(), &dms.Message{
		Type:   "Hello",
		Sender: node.actor.Address().InboxAddress,
		Data:   []byte("nunet node"),
	})
	assert.NoError(t, err)

	msg2 := <-node.actor.Messages()
	assert.Equal(t, "nunet node", string(msg2.Data))
}

type benchmarkerStub struct {
	cap *types.Capability
	err error
}

func (r *benchmarkerStub) Benchmark(_ context.Context) (*types.Capability, error) {
	return r.cap, r.err
}

func createNetwork(t *testing.T, bootstrap []multiaddr.Multiaddr, port string) *libp2p.Libp2p {
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	net, err := network.NewNetwork(&types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrap,
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               backgroundtasks.NewScheduler(1),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/127.0.0.1/tcp/" + port},
			PeerCountDiscoveryLimit: 40,
			PrivateNetwork: types.PrivateNetworkConfig{
				WithSwarmKey: false,
			},
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
