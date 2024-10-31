// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dms

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/api"
	"gitlab.com/nunet/device-management-service/db"
	gdb "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/internal"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	DefaultContextName = "dms"
	UserContextName    = "user"
	KeystoreDir        = "key/"
	CapstoreDir        = "cap/"
)

//go:embed data/GeoLite2-Country.mmdb
var geoLite2Country []byte

// NewP2P is stub, real implementation is needed in order to pass it to
// routers which access them in some handlers.
func NewP2P() libp2p.Libp2p {
	return libp2p.Libp2p{}
}

// QUESTION(dms-initialization): should the db instance be constructed here?
func Run(ksPassphrase string, contextName string) error {
	if contextName == "" {
		contextName = DefaultContextName
	}

	gcfg := config.GetConfig()

	// load geoip2 database
	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	if err != nil {
		return fmt.Errorf("unable to load geoip2 database: %w", err)
	}

	fs := afero.NewOsFs()

	keyStoreDir := filepath.Join(gcfg.UserDir, KeystoreDir)
	keyStore, err := keystore.New(fs, keyStoreDir)
	if err != nil {
		return fmt.Errorf("unable to create keystore: %w", err)
	}

	var priv crypto.PrivKey
	ksPrivKey, err := keyStore.Get(contextName, ksPassphrase)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			priv, err = GenerateAndStorePrivKey(keyStore, ksPassphrase, contextName)
			if err != nil {
				return fmt.Errorf("couldn't generate and store priv key into keystore: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get private key from keystore; Error: %v", err)
		}
	} else {
		priv, err = ksPrivKey.PrivKey()
		if err != nil {
			return fmt.Errorf("unable to convert key from keystore to private key: %v", err)
		}
	}
	pubKey := priv.GetPublic()

	db, err := db.ConnectDatabase(gcfg.WorkDir)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	hardwareManager := hardware.NewHardwareManager()
	repos := resources.ManagerRepos{
		OnboardedResources: gdb.NewOnboardedResources(db),
		ResourceAllocation: gdb.NewResourceAllocation(db),
	}
	resourceManager, err := resources.NewResourceManager(repos, hardwareManager)
	if err != nil {
		return fmt.Errorf("unable to create resource manager: %w", err)
	}

	onboardR := gdb.NewOnboardingConfig(db)

	onboard := onboarding.New(&onboarding.Config{
		Fs:              afero.Afero{Fs: fs},
		ConfigRepo:      onboardR,
		Hardware:        hardwareManager,
		ResourceManager: resourceManager,
		WorkDir:         gcfg.WorkDir,
		DatabasePath:    fmt.Sprintf("%s/nunet.db", gcfg.WorkDir),
	})

	var p2pNet *libp2p.Libp2p

	bootstrapPeers := make([]multiaddr.Multiaddr, len(gcfg.P2P.BootstrapPeers))
	for i, addr := range gcfg.P2P.BootstrapPeers {
		bootstrapPeers[i], _ = multiaddr.NewMultiaddr(addr)
	}

	cfg := &types.Libp2pConfig{
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-test",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10),
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           gcfg.P2P.ListenAddress,
		PeerCountDiscoveryLimit: 40,
		Memory:                  gcfg.P2P.Memory,
		FileDescriptors:         gcfg.P2P.FileDescriptors,
	}

	p2p, err := libp2p.New(cfg, fs)
	if err != nil {
		return fmt.Errorf("unable to create libp2p instance: %v", err)
	}

	if err = p2p.Init(); err != nil {
		return fmt.Errorf("unable to initialize libp2p: %v", err)
	}

	if err = p2p.Start(); err != nil {
		return fmt.Errorf("unable to start libp2p: %v", err)
	}

	p2pNet = p2p

	trustCtx, err := did.NewTrustContextWithPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("unable to create trust context: %w", err)
	}

	capStoreDir := filepath.Join(gcfg.UserDir, CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", contextName))
	var capCtx ucan.CapabilityContext

	if _, err := os.Stat(capStoreFile); err != nil {
		if err := fs.MkdirAll(capStoreDir, os.FileMode(0o700)); err != nil {
			return fmt.Errorf("unable to create capability context directory: %w", err)
		}
		// does not exist; create it
		rootDID := did.FromPublicKey(pubKey)
		capCtx, err = ucan.NewCapabilityContext(trustCtx, rootDID, nil, ucan.TokenList{}, ucan.TokenList{})
		if err != nil {
			return fmt.Errorf("unable to create capability context: %w", err)
		}

		// Save it!
		f, err := os.Create(capStoreFile)
		if err != nil {
			return fmt.Errorf("unable to create capability context file: %w", err)
		}

		err = ucan.SaveCapabilityContext(capCtx, f)
		_ = f.Close()

		if err != nil {
			return fmt.Errorf("unable to save capability context: %w", err)
		}
	} else {
		f, err := os.Open(capStoreFile)
		if err != nil {
			return fmt.Errorf("unable to open capability context: %w", err)
		}

		capCtx, err = ucan.LoadCapabilityContext(trustCtx, f)
		_ = f.Close()

		if err != nil {
			return fmt.Errorf("unable to load capability context: %w", err)
		}
	}

	trustCtx.Start(time.Hour)
	capCtx.Start(5 * time.Minute)

	hostLocation := node.HostGeolocation{
		HostCountry:   gcfg.HostCountry,
		HostCity:      gcfg.HostCity,
		HostContinent: gcfg.HostContinent,
	}

	portConfig := node.PortConfig{
		AvailableRangeFrom: gcfg.PortAvailableRangeFrom,
		AvailableRangeTo:   gcfg.PortAvailableRangeTo,
	}

	hostID := p2p.Host.ID().String()
	node, err := node.New(onboard, capCtx, hostID, p2p, resourceManager, cfg.Scheduler, hardwareManager, geoip2db, hostLocation, portConfig)
	if err != nil {
		return fmt.Errorf("failed to create node: %s", err)
	}

	err = node.Start()
	if err != nil {
		return fmt.Errorf("failed to start node: %s", err)
	}

	// initialize rest api server
	restConfig := api.RESTServerConfig{
		P2P:        p2pNet,
		Onboarding: onboard,
		Logger:     logger.New("rest-server"),
		Resource:   resourceManager,
		MidW:       nil,
		Port:       gcfg.Rest.Port,
		Addr:       gcfg.Rest.Addr,
	}
	rServer := api.NewRESTServer(&restConfig)
	rServer.InitializeRoutes()

	go func() {
		err := rServer.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// wait for SIGINT or SIGTERM
	sig := <-internal.ShutdownChan

	// clean up
	go func() {
		err = node.Stop()
		if err != nil {
			log.Errorf("failed to stop node: %s", err)
		}
		err = p2p.Stop()
		if err != nil {
			log.Errorf("failed to stop libp2p network: %s", err)
		}
		log.Infof("Shutting down after receiving %v...\n", sig)
		os.Exit(0)
	}()

	sig = <-internal.ShutdownChan
	log.Infof("Shutting down after receiving %v...\n", sig)
	os.Exit(1)
	return nil
}

// GenerateAndStorePrivKey generates a new key pair using Secp256k1,
// storing the private key into user's keystore.
func GenerateAndStorePrivKey(ks keystore.KeyStore, passphrase string, keyID string) (crypto.PrivKey, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	if err != nil {
		return nil, fmt.Errorf("unable to generate key pair: %w", err)
	}

	rawPriv, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %w", err)
	}

	_, err = ks.Save(
		keyID,
		rawPriv,
		passphrase,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to save private key into the keystore: %w", err)
	}

	return priv, nil
}

func ValidateOnboarding(oConf *types.OnboardingConfig) {
	// Check 1: Check if payment address is valid
	err := utils.ValidateAddress(oConf.PublicKey)
	if err != nil {
		log.Errorf("the payment address %s is not valid", oConf.PublicKey)
		log.Error("exiting DMS")
		return
	}
}
