package node

import (
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/did"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
)

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

func getMockEnsembleConfig(t *testing.T) jobtypes.EnsembleConfig {
	t.Helper()

	return jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"allocation1": {
					Executor: "docker",
					Resources: types.Resources{
						CPU: types.CPU{
							ClockSpeed: 2.4,
							Cores:      2,
							Model:      "Intel Core i7",
							Vendor:     "Intel",
						},
						GPUs: []types.GPU{
							{
								Model:      "NVIDIA GeForce GTX 1080",
								Vendor:     "NVIDIA",
								VRAM:       8,
								Index:      0,
								PCIAddress: "0000:01:00.0",
							},
						},
						RAM: types.RAM{
							Size:       16,
							ClockSpeed: 2400,
						},
						Disk: types.Disk{
							Size:       256,
							Model:      "Samsung 970 EVO",
							Vendor:     "Samsung",
							Type:       "SSD",
							Interface:  "NVMe",
							ReadSpeed:  3500,
							WriteSpeed: 2500,
						},
					},
					Execution: types.SpecConfig{
						Type: "docker",
						Params: map[string]interface{}{
							"image": "nginx",
						},
					},
				},
			},
			Nodes: map[string]jobtypes.NodeConfig{
				"node1": {
					Allocations: []string{"allocation1"},
				},
			},
		},
	}
}

func getMockActorHandle(t *testing.T) actor.Handle {
	_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)
	id, err := pubK.Raw()
	require.NoError(t, err)
	testDID := did.FromPublicKey(pubK)

	return actor.Handle{
		ID: actor.ID{
			PublicKey: id,
		},
		DID: testDID,
		Address: actor.Address{
			HostID:       "hostID",
			InboxAddress: "inboxAddress",
		},
	}
}

func getMockDeploymentRequest(t *testing.T) actor.Envelope {
	t.Helper()

	request := NewDeploymentRequest{
		Ensemble: getMockEnsembleConfig(t),
	}
	replyTo := "test/behavior"

	msg, err := actor.Message(getMockActorHandle(t), getMockActorHandle(t), behaviors.NewDeploymentBehavior, request,
		actor.WithMessageReplyTo(replyTo), actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Minute).UnixNano())))
	require.NoError(t, err)

	return msg
}
