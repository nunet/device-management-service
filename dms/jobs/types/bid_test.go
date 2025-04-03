// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"testing"

	"gitlab.com/nunet/device-management-service/lib/crypto"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
)

func TestBid(t *testing.T) {
	t.Parallel()

	t.Run("must be able to validate the bid", func(t *testing.T) {
		privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testDID := did.FromPublicKey(pubK)
		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}

		provider := did.NewProvider(testDID, privK)

		err = testBid.Sign(provider)
		require.NoError(t, err)

		err = testBid.Validate()
		require.NoError(t, err)
	})

	t.Run("must be able to get signature data", func(t *testing.T) {
		t.Parallel()

		privK, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testDID := did.FromPublicKey(pubK)
		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}

		provider := did.NewProvider(testDID, privK)

		err = testBid.Sign(provider)
		require.NoError(t, err)

		data, err := testBid.SignatureData()
		require.NoError(t, err)
		require.NotNil(t, data)
		require.NotEmpty(t, data)
	})

	t.Run("must be able to get ensemble id", func(t *testing.T) {
		t.Parallel()

		_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}
		ensembleID := testBid.EnsembleID()
		require.Equal(t, "testEnsembleID", ensembleID)
	})

	t.Run("must be able to get nodeID", func(t *testing.T) {
		t.Parallel()

		_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}
		nodeID := testBid.NodeID()
		require.Equal(t, "testNodeID", nodeID)
	})

	t.Run("must be able to get peerID", func(t *testing.T) {
		t.Parallel()

		_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}
		actualPeerID := testBid.Peer()
		require.Equal(t, peerID.String(), actualPeerID)
	})

	t.Run("must be able to get handle", func(t *testing.T) {
		t.Parallel()

		_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}
		handle := testBid.Handle()
		require.Equal(t, actor.Handle{}, handle)
	})

	t.Run("must be able to get location", func(t *testing.T) {
		t.Parallel()

		_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
		require.NoError(t, err)

		peerID, err := peer.IDFromPublicKey(pubK)
		require.NoError(t, err)

		testBid := Bid{
			V1: &BidV1{
				EnsembleID: "testEnsembleID",
				NodeID:     "testNodeID",
				Peer:       peerID.String(),
				Location: Location{
					Continent: "testContinent",
					Country:   "testCountry",
					City:      "testCity",
				},
				Handle: actor.Handle{},
			},
		}
		location := testBid.Location()
		require.Equal(t, Location{
			Continent: "testContinent",
			Country:   "testCountry",
			City:      "testCity",
		}, location)
	})

	t.Run("must be able to validate ensemble bid request", func(t *testing.T) {
		t.Parallel()

		testBid := EnsembleBidRequest{
			ID: "testID",
			Request: []BidRequest{
				{
					V1: &BidRequestV1{
						NodeID: "testNodeID",
					},
				},
			},
		}

		err := testBid.Validate()
		require.NoError(t, err)
	})
}
