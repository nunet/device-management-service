// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dms

import (
	"os"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	clover "github.com/ostafen/clover/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clover_db "gitlab.com/nunet/device-management-service/db/clover"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func TestNewDMSDB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dms_test_db")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	db, err := NewDMSDB(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// see that collections are created
	collections := []string{
		"free_resources",
		"request_tracker",
		"onboarded_resources",
		"machine_resources",
		"onboarding_config",
		"resource_allocation",
		"deployments",
		"gpu",
		"contracts",
		"contracts_keys",
		"provisioned_resources",
		"contracts_payments",
		"service_provider_transactions",
		"contracts_usage",
		"usage_metadata",
		"payment_quotes",
	}

	for _, coll := range collections {
		exists, err := db.HasCollection(coll)
		assert.NoError(t, err)
		assert.True(t, exists, "Collection %s should exist", coll)
	}

	err = db.Close()
	assert.NoError(t, err)
}

func TestPrepareConfig(t *testing.T) {
	env := env.NewMockEnvironment()
	err := env.Setenv("BOOTSTRAP_PEERS", "peer1,peer2")
	require.NoError(t, err)
	gcfg := &config.Config{
		P2P: config.P2P{
			BootstrapPeers: []string{"original"},
		},
	}

	result := prepareConfig(gcfg, env)

	assert.Equal(t, []string{"peer1", "peer2"}, result.P2P.BootstrapPeers)
}

func TestInitStorage(t *testing.T) {
	// without storage mode
	gcfg := &config.Config{General: config.General{StorageMode: false}}
	vc, err := initStorage(gcfg)
	assert.NoError(t, err)
	assert.Nil(t, vc)

	// this would require actual gluster, so skip or mock?
	// for this test, it fails since not set up. only testing the path
	gcfg.General.StorageMode = true
	gcfg.General.StorageGlusterfsHostname = "invalid"
	vc, err = initStorage(gcfg)
	assert.Error(t, err) // Should fail because gluster is not set up
	assert.Nil(t, vc)
}

func TestGenerateAndStorePrivKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keystore_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := afero.NewOsFs()
	ks, err := keystore.New(fs, tempDir, false)
	require.NoError(t, err)

	privK, err := GenerateAndStorePrivKey(ks, "testpass", "testkey")
	assert.NoError(t, err)
	assert.NotNil(t, privK)

	// test retrieve
	ksPrivKey, err := ks.Get("testkey", "testpass")
	assert.NoError(t, err)
	retrievedPrivK, err := ksPrivKey.PrivKey()
	assert.NoError(t, err)
	assert.Equal(t, privK, retrievedPrivK)
}

func TestInitCrypto(t *testing.T) {
	fs := afero.NewMemMapFs()
	gcfg := &config.Config{
		General: config.General{
			UserDir: "/user",
		},
	}
	ksPassphrase := "testpass"
	contextName := "testcontext"

	privK, err := initCrypto(fs, gcfg, ksPassphrase, contextName)
	assert.NoError(t, err)
	assert.NotNil(t, privK)
}

func TestInitDatabase(t *testing.T) {
	db, err := NewDMSMemDB([]string{})
	require.NoError(t, err)

	stores, err := initStores(db)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, stores)
	assert.NotNil(t, stores.contractStore)
	assert.NotNil(t, stores.paymentsStore)
	assert.NotNil(t, stores.usageStore)
	assert.NotNil(t, stores.txStore)
	assert.NotNil(t, stores.paymentQuoteStore)
	assert.NotNil(t, stores.provisionedResourceStore)

	err = db.Close()
	assert.NoError(t, err)
}

func TestInitManagers(t *testing.T) {
	db, err := NewDMSMemDB([]string{
		"free_resources",
		"onboarded_resources",
		"onboarding_config",
	})
	require.NoError(t, err)
	defer db.Close()

	resourceManager, onboardingManager, deploymentStore, err := initManagers(db)
	assert.NoError(t, err)
	assert.NotNil(t, resourceManager)
	assert.NotNil(t, onboardingManager)
	assert.NotNil(t, deploymentStore)
}

func TestImportAndStorePrivKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keystore_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := afero.NewOsFs()
	ks, err := keystore.New(fs, tempDir, false)
	require.NoError(t, err)

	// generate a key to import
	privK, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	require.NoError(t, err)

	rawPriv, err := crypto.MarshalPrivateKey(privK)
	require.NoError(t, err)

	importedPrivK, err := ImportAndStorePrivKey(ks, rawPriv, "testpass", "importkey")
	assert.NoError(t, err)
	assert.NotNil(t, importedPrivK)

	// verify retrieval
	ksPrivKey, err := ks.Get("importkey", "testpass")
	assert.NoError(t, err)
	retrievedPrivK, err := ksPrivKey.PrivKey()
	assert.NoError(t, err)
	assert.Equal(t, importedPrivK, retrievedPrivK)
}

func TestInitDB(t *testing.T) {
	// test missing collections
	db, err := NewDMSMemDB([]string{
		"free_resources",
		"onboarded_resources",
		// missing "onboarding_config",
	})
	require.NoError(t, err)

	defer db.Close()

	resourceManager, onboardingManager, deploymentStore, err := initManagers(db)
	assert.Error(t, err)
	assert.Nil(t, resourceManager)
	assert.Nil(t, onboardingManager)
	assert.Nil(t, deploymentStore)
}

// NewDMSDB creates a clover database with all known dms collections
func NewDMSMemDB(collections []string) (*clover.DB, error) {
	return clover_db.NewMemDB(
		collections,
	)
}
