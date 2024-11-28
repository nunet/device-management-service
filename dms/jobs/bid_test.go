// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
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
		V1: &job_types.BidV1{
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
