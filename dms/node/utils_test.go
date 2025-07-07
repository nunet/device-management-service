package node

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "embed"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/oschwald/geoip2-golang"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	cloverDB "gitlab.com/nunet/device-management-service/db/clover"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/executor/docker"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	portRangeFrom = 3000
	portRangeTo   = 3100

	MockTotalCPU  = 12
	MockTotalRAM  = 32 * 1024 * 1024 * 1024  // 32 GB
	MockTotalDisk = 100 * 1024 * 1024 * 1024 // 100 GB

)

//go:embed data/GeoLite2-Country.mmdb
var geoLite2Country []byte

// createKey creates a key in the keystore.
func createKey(t *testing.T, fs afero.Fs, basePath, contextKey, passphrase string) {
	t.Helper()

	keyStoreDir := filepath.Join(basePath, KeystoreDir)
	ks, err := keystore.New(fs, keyStoreDir)
	require.NoError(t, err)

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)

	rawPriv, err := crypto.MarshalPrivateKey(priv)
	require.NoError(t, err)

	_, err = ks.Save(
		contextKey,
		rawPriv,
		passphrase,
	)
	require.NoError(t, err)
}

func setupTestNetwork(t *testing.T, substrate *network.Substrate) (network.Network, crypto.PrivKey) {
	t.Helper()
	priv, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)
	peerID, err := peer.IDFromPublicKey(pubKey)
	require.NoError(t, err)

	return substrate.AddWiredPeer(peerID), priv
}

func newLibp2pNetwork(t *testing.T, mockFs afero.Fs, bootstrap []multiaddr.Multiaddr, scheduler *backgroundtasks.Scheduler) (*libp2p.Libp2p, crypto.PrivKey) {
	t.Helper()

	// config
	dcfg := config.DefaultConfig
	dcfg.Observability.ElasticsearchEnabled = false

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)

	net, err := network.NewNetwork(&types.NetworkConfig{
		Type: types.Libp2pNetwork,
		Libp2pConfig: types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrap,
			Rendezvous:              "nunet-randevouz",
			Server:                  false,
			Scheduler:               scheduler,
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           []string{"/ip4/0.0.0.0/tcp/0"},
			GracePeriodMs:           1000,
			PeerCountDiscoveryLimit: 40,
			GossipMaxMessageSize:    2 << 16,
			Memory:                  dcfg.P2P.Memory,
			FileDescriptors:         dcfg.P2P.FileDescriptors,
		},
	}, mockFs)

	p2pNet, ok := net.(*libp2p.Libp2p)
	require.True(t, ok)

	require.NoError(t, err)
	require.NotNil(t, p2pNet)

	err = p2pNet.Init(&dcfg)
	require.NoError(t, err)

	err = p2pNet.Start()
	require.NoError(t, err)

	return p2pNet, priv
}

func newActor(t *testing.T, priv crypto.PrivKey, net network.Network) (*actor.BasicActor, ucan.CapabilityContext, did.TrustContext, did.DID) {
	t.Helper()

	rootDID, rootTrust := actor.MakeRootTrustContext(t)
	actorDID, actorTrust := actor.MakeTrustContext(t, priv)
	actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)
	actor := actor.CreateActor(t, net, actorCap)

	return actor, actorCap, rootTrust, rootDID
}

func newMockAllocator(t *testing.T, substrate *network.Substrate) (*allocator, network.Network, crypto.PrivKey) {
	t.Helper()

	// Mock dependencies
	mockVolumeTracker := &storage.VolumeTracker{}
	mockPortAllocator := newPortAllocator(
		PortConfig{
			AvailableRangeFrom: portRangeFrom,
			AvailableRangeTo:   portRangeTo,
		},
	)

	db, err := cloverDB.NewMemDB([]string{
		"free_resources",
		"onboarded_resources",
		"machine_resources",
		"onboarding_config",
		"resource_allocation",
	})
	require.NoError(t, err)
	require.NotNil(t, db)

	resourceManRepos := resources.ManagerRepos{
		OnboardedResources: cloverDB.NewGenericEntityRepository[types.OnboardedResources](db),
		ResourceAllocation: cloverDB.NewGenericRepository[types.ResourceAllocation](db),
	}

	hardwareManager := hardware.NewMockHardwareManager(
		types.MachineResources{
			Resources: types.Resources{
				CPU:  types.CPU{Cores: 4},
				RAM:  types.RAM{Size: 8},
				Disk: types.Disk{Size: 100},
			},
		},
		types.Resources{
			CPU:  types.CPU{Cores: 4},
			RAM:  types.RAM{Size: 8},
			Disk: types.Disk{Size: 100},
		},
		types.Resources{
			CPU:  types.CPU{Cores: 0},
			RAM:  types.RAM{Size: 0},
			Disk: types.Disk{Size: 0},
		},
	)
	mockResourceManager, err := resources.NewResourceManager(resourceManRepos, hardwareManager)
	require.NoError(t, err)
	require.NotNil(t, mockResourceManager)

	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
	workDir := "/tmp/test-workdir"
	hostID := "test-host-id"

	p2pNet, priv := setupTestNetwork(t, substrate)

	// seed some onboarded resources
	err = mockResourceManager.UpdateOnboardedResources(
		context.Background(),
		types.Resources{
			CPU:  types.CPU{Cores: 4},
			RAM:  types.RAM{Size: 4},
			Disk: types.Disk{Size: 10},
		},
	)
	require.NoError(t, err)

	alloc := newAllocator(
		mockVolumeTracker,
		mockPortAllocator,
		mockResourceManager,
		hardwareManager,
		p2pNet,
		mockFs,
		workDir,
		hostID,
	)

	return alloc, p2pNet, priv
}

func newMockNode(t *testing.T, substrate *network.Substrate) (*Node, did.TrustContext, did.DID) {
	t.Helper()

	// var geoLite2Country []byte
	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	require.NoError(t, err)
	require.NotNil(t, geoip2db)

	mockFs := afero.Afero{Fs: afero.NewMemMapFs()}

	// config
	dcfg := config.DefaultConfig
	dcfg.Observability.ElasticsearchEnabled = false

	// mock database
	db, err := cloverDB.NewMemDB([]string{
		"free_resources",
		"request_tracker",
		"onboarded_resources",
		"machine_resources",
		"onboarding_config",
		"resource_allocation",
		"orchestrator_view",
	})
	require.NoError(t, err)

	repos := resources.ManagerRepos{
		OnboardedResources: cloverDB.NewGenericEntityRepository[types.OnboardedResources](db),
		ResourceAllocation: cloverDB.NewGenericRepository[types.ResourceAllocation](db),
	}

	mockHardwareManager := hardware.NewMockHardwareManager(
		types.MachineResources{
			Resources: types.Resources{
				CPU:  types.CPU{Cores: MockTotalCPU},
				RAM:  types.RAM{Size: MockTotalRAM},   // 32 GB
				Disk: types.Disk{Size: MockTotalDisk}, // 100 GB
			},
		},
		types.Resources{
			CPU:  types.CPU{Cores: MockTotalCPU},
			RAM:  types.RAM{Size: MockTotalRAM},   // 32 GB
			Disk: types.Disk{Size: MockTotalDisk}, // 100 GB
		},
		types.Resources{
			CPU:  types.CPU{Cores: 2},
			RAM:  types.RAM{Size: 2 * 1024 * 1024 * 1024},  // 2 GB
			Disk: types.Disk{Size: 2 * 1024 * 1024 * 1024}, // 2 GB
		},
	)
	mockResourceManager, err := resources.NewResourceManager(repos, mockHardwareManager)
	require.NoError(t, err)
	require.NotNil(t, mockResourceManager)

	onboardR := cloverDB.NewGenericEntityRepository[types.OnboardingConfig](db)
	orchestR := cloverDB.NewGenericRepository[jobtypes.OrchestratorView](db)

	onboardingManager, err := onboarding.New(context.Background(), mockResourceManager, mockHardwareManager, onboardR)
	require.NoError(t, err)

	hostLocation := geolocation.Geolocation{
		Continent: dcfg.HostContinent,
		Country:   dcfg.HostCountry,
		City:      dcfg.HostCity,
	}

	scheduler := backgroundtasks.NewScheduler(1, time.Second)

	vNet, priv := setupTestNetwork(t, substrate)

	// allocator, nP2PNet, priv := newMockAllocator(t, substrate)
	allocator := newAllocator(
		&storage.VolumeTracker{},
		newPortAllocator(
			PortConfig{
				AvailableRangeFrom: portRangeFrom,
				AvailableRangeTo:   portRangeTo,
			},
		),
		mockResourceManager,
		mockHardwareManager,
		vNet,
		mockFs,
		dcfg.WorkDir,
		vNet.GetHostID().String(),
	)
	nActor, nActorCap, nRootTrust, nRootDID := newActor(t, priv, vNet)

	node := &Node{}
	node.onboarding = onboardingManager
	node.allocator = allocator
	node.network = vNet
	node.actor = nActor
	node.rootCap = nActorCap
	node.hardware = mockHardwareManager
	node.resourceManager = mockResourceManager
	node.scheduler = scheduler
	node.hostID = vNet.GetHostID().String()
	node.orchestratorRepo = orchestR
	node.hostLocation = hostLocation
	node.volumeController = &controller.GlusterController{}
	node.volumeOwners = make(map[string]string)
	node.geoIP = geoip2db
	node.fs = mockFs
	node.ctx, node.cancel = context.WithCancel(context.Background())
	node.orchestratorRepo = cloverDB.NewGenericRepository[jobtypes.OrchestratorView](db)
	node.orchestratorRegistry = orchestrator.NewMockOrchestratorRegistry()
	node.bids = make(map[string]*bidState)
	node.answeredBids = make(map[string][]uint64)
	node.peers = make(map[peer.ID]*peerState)
	node.dmsConfig = dcfg
	node.ctx, node.cancel = context.WithCancel(context.Background())
	node.executors = make(map[string]executorMetadata)
	node.executors[string(jobtypes.ExecutorDocker)] = executorMetadata{
		executor:      &docker.Executor{},
		executionType: jobtypes.ExecutorDocker,
	}

	dmsBehaviors := node.getDMSBehaviors()
	for behavior, handler := range dmsBehaviors {
		err := node.actor.AddBehavior(behavior, handler.fn, handler.opts...)
		require.NoError(t, err, "failed to add behavior %s", behavior)
	}

	err = node.actor.Start()
	require.NoError(t, err)

	return node, nRootTrust, nRootDID
}

func mockOnboarding(t *testing.T, node *Node, cpu float32, ram, disk uint64) {
	t.Helper()
	oc, err := node.onboarding.Onboard(
		context.Background(),
		types.OnboardingConfig{
			IsOnboarded: true,
			OnboardedResources: types.Resources{
				CPU:  types.CPU{Cores: cpu},
				RAM:  types.RAM{Size: ram},
				Disk: types.Disk{Size: disk},
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, oc)
}

func newMockNodeWithSender(t *testing.T, behavior string) (*Node, *actor.BasicActor, network.Network) {
	t.Helper()

	netSubstrate := network.NewSubstrate()
	node, nRootTrust, nRootDID := newMockNode(t, netSubstrate)

	// sender actor
	sNet, sPriv := setupTestNetwork(t, netSubstrate)

	sActor, sActorCap, sRootTrust, sRootDID := newActor(t, sPriv, sNet)
	err := sActor.Start()
	assert.NoError(t, err)

	actor.AllowReciprocal(t, node.actor.Security().Capability(), nRootTrust, nRootDID, sRootDID, behavior)
	actor.AllowReciprocal(t, sActorCap, sRootTrust, sRootDID, nRootDID, behavior)

	return node, sActor, sNet
}
