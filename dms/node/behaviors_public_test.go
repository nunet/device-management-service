package node

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	cloverDB "gitlab.com/nunet/device-management-service/db/clover"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleStatusDiscoveryBehavior(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	t.Run("invoke status discovery behavior", func(t *testing.T) {
		t.Parallel()

		node, sActor := newMockNodeWithSender(t)

		// add onboarding info
		totalResources, err := node.hardware.GetMachineResources()
		require.NoError(t, err)

		cpuToOnboard := totalResources.CPU.Cores * 0.4
		ramToOnboard := totalResources.RAM.Size / 2
		_, err = node.onboarding.Onboard(
			context.Background(),
			types.OnboardingConfig{
				IsOnboarded: true,
				OnboardedResources: types.Resources{
					CPU:  types.CPU{Cores: cpuToOnboard},
					RAM:  types.RAM{Size: ramToOnboard},
					Disk: types.Disk{Size: 100},
				},
			},
		)
		require.NoError(t, err)

		// Add behavior to test
		err = node.actor.AddBehavior(behaviors.StatusDiscoveryBehavior, node.handleStatusDiscoveryBehavior)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.StatusDiscoveryBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DiscoveryStatusResponse
		err = json.Unmarshal(reply.Message, &resp)

		assert.NotEmpty(t, resp)

		// filter for self peerID
		peerResp := resp[node.network.GetHostID().String()]

		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().DID.String(), peerResp.DID)
		assert.Empty(t, peerResp.Errors)
	})

	t.Run("expect a reply with introduced errors", func(t *testing.T) {
		t.Parallel()

		node, sActor := newMockNodeWithSender(t)

		// introduce errors
		node.hardware = nil
		node.onboarding = nil

		// Add behavior to test
		err := node.actor.AddBehavior(behaviors.StatusDiscoveryBehavior, node.handleStatusDiscoveryBehavior)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.StatusDiscoveryBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		// we should still get a reply with errors even if all information can't be fetched
		var resp DiscoveryStatusResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NotEmpty(t, resp)

		// filter for self peerID
		peerResp := resp[node.network.GetHostID().String()]

		assert.NoError(t, err)
		assert.Equal(t, node.actor.Handle().DID.String(), peerResp.DID)
		assert.Equal(t, 2, len(peerResp.Errors))
	})
}

func newActor(t *testing.T, priv crypto.PrivKey, net network.Network) (*actor.BasicActor, ucan.CapabilityContext, did.TrustContext, did.DID) {
	t.Helper()

	rootDID, rootTrust := actor.MakeRootTrustContext(t)
	actorDID, actorTrust := actor.MakeTrustContext(t, priv)
	actorCap := actor.MakeCapabilityContext(t, actorDID, rootDID, actorTrust, rootTrust)
	actor := actor.CreateActor(t, net, actorCap)

	return actor, actorCap, rootTrust, rootDID
}

func newMockNodeWithSender(t *testing.T) (*Node, *actor.BasicActor) {
	t.Helper()

	// sendor actor
	sAddr, sPriv, sNet := actor.NewLibp2pNetwork(t, []multiaddr.Multiaddr{})
	sActor, sActorCap, sRootTrust, sRootDID := newActor(t, sPriv, sNet)

	// node
	_, nPriv, nNet := actor.NewLibp2pNetwork(t, sAddr)
	nActor, nActorCap, nRootTrust, nRootDID := newActor(t, nPriv, nNet)

	actor.AllowReciprocal(t, sActorCap, sRootTrust, sRootDID, nRootDID, behaviors.StatusDiscoveryBehavior)
	actor.AllowReciprocal(t, nActorCap, nRootTrust, nRootDID, sRootDID, behaviors.StatusDiscoveryBehavior)

	node := &Node{}

	node.network = nNet
	node.actor = nActor

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

	hardwareManager := hardware.NewHardwareManager()
	resourceMan, err := resources.NewResourceManager(repos, hardwareManager)
	require.NoError(t, err)
	node.resourceManager = resourceMan
	node.hardware = hardwareManager

	onboardR := cloverDB.NewGenericEntityRepository[types.OnboardingConfig](db)

	onboardingManager, err := onboarding.New(context.Background(), resourceMan, hardwareManager, onboardR)
	require.NoError(t, err)

	node.onboarding = onboardingManager

	err = sActor.Start()
	require.NoError(t, err)
	err = nActor.Start()
	require.NoError(t, err)

	return node, sActor
}
