// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

func TestTrustContext(t *testing.T) {
	ctx := NewTrustContext()

	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")

	pubDID := FromPublicKey(pubk)
	anchor, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor.DID(), pubDID, "compare anchor DID with pubk DID")

	anchor2, err := ctx.GetAnchor(pubDID)
	require.NoError(t, err, "get key anchor")
	require.Equal(t, anchor2, anchor, "cached anchor must equal the initial")

	provider, err := ProviderFromPrivateKey(privk)
	require.NoError(t, err, "provider from public key")
	ctx.AddProvider(provider)

	provider2, err := ctx.GetProvider(provider.DID())
	require.NoError(t, err, "get key provider")
	require.Equal(t, provider2, provider, "cached provider must equal the initial")

	require.Equal(t, ctx.Anchors(), []DID{pubDID}, "anchor list")
	require.Equal(t, ctx.Providers(), []DID{pubDID}, "provider list")
}
