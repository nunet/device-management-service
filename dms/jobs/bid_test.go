package jobs

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
)

func TestValidation(t *testing.T) {
	privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)

	peerID, err := peer.IDFromPublicKey(pubK)
	require.NoError(t, err)

	testDID := did.FromPublicKey(pubK)

	testBid := Bid{
		V1: &BidV1{
			EnsembleID: "testEnsembleID",
			NodeID:     "testnodeID",
			Peer:       peerID.String(),
			Location: Location{
				Region:  "testRegion",
				Country: "testCOuntry",
				City:    "testCity",
			},
			Handle: actor.Handle{},
		},
	}

	provider := did.NewProvider(testDID, privK)

	err = testBid.Sign(provider)
	require.NoError(t, err)

	err = testBid.Validate()
	require.NoError(t, err)
}
