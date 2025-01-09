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
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/oschwald/geoip2-golang"
	clover "github.com/ostafen/clover/v2"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/api"
	clover_db "gitlab.com/nunet/device-management-service/db/repositories/clover"
	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"

	"go.elastic.co/apm/module/apmgin/v2"
)

//go:embed data/GeoLite2-Country.mmdb
var geoLite2Country []byte

type DMS struct {
	P2P        *libp2p.Libp2p
	Node       *node.Node
	RestServer *api.Server
}

func initialize(gcfg *config.Config) {
	fs := afero.NewOsFs()

	workDir := gcfg.WorkDir
	if workDir != "" {
		err := fs.MkdirAll(workDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create work directory: %v", err)
		}
	}

	dataDir := gcfg.DataDir
	if dataDir != "" {
		err := fs.MkdirAll(dataDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create data directory: %v", err)
		}
	}

	userDir := gcfg.UserDir
	if userDir != "" {
		err := fs.MkdirAll(userDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create user directory: %v", err)
		}
	}

	libp2pLogging := os.Getenv("DMS_CONN_LOGS")
	if libp2pLogging == "false" || libp2pLogging == "" {
		err := silenceConnLogs()
		if err != nil {
			log.Warnf("unable to set libp2p logging: %v", err)
		}
	}
}

func NewDMS(gcfg *config.Config, ksPassphrase, contextName string) (*DMS, error) {
	if contextName == "" {
		contextName = node.DefaultContextName
	}

	initialize(gcfg)

	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	if err != nil {
		return nil, fmt.Errorf("unable to load geoip2 database: %w", err)
	}
	log.Debugf("loaded geoip2 database: %v", geoip2db)

	fs := afero.NewOsFs()

	keyStoreDir := filepath.Join(gcfg.UserDir, node.KeystoreDir)
	keyStore, err := keystore.New(fs, keyStoreDir)
	if err != nil {
		return nil, fmt.Errorf("unable to create keystore: %w", err)
	}

	var priv crypto.PrivKey
	ksPrivKey, err := keyStore.Get(contextName, ksPassphrase)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			priv, err = GenerateAndStorePrivKey(keyStore, ksPassphrase, contextName)
			if err != nil {
				return nil, fmt.Errorf("couldn't generate and store priv key into keystore: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get private key from keystore; Error: %v", err)
		}
	} else {
		priv, err = ksPrivKey.PrivKey()
		if err != nil {
			return nil, fmt.Errorf("unable to convert key from keystore to private key: %v", err)
		}
	}
	pubKey := priv.GetPublic()

	db, err := NewDMSDB(gcfg.General.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	hardwareManager := hardware.NewHardwareManager()
	repos := resources.ManagerRepos{
		OnboardedResources: clover_db.NewOnboardedResources(db),
		ResourceAllocation: clover_db.NewResourceAllocation(db),
	}
	resourceManager, err := resources.NewResourceManager(repos, hardwareManager)
	if err != nil {
		return nil, fmt.Errorf("unable to create resource manager: %w", err)
	}

	onboardR := clover_db.NewOnboardingConfig(db)
	orchestR := clover_db.NewOrchestratorView(db)
	contractR := clover_db.NewContractRepo(db)

	onboardingManager, err := onboarding.New(context.Background(), resourceManager, hardwareManager, onboardR)
	if err != nil {
		return nil, fmt.Errorf("unable to create onboarding manager: %w", err)
	}

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

	p2pNet, err := libp2p.New(cfg, fs)
	if err != nil {
		return nil, fmt.Errorf("unable to create libp2p instance: %v", err)
	}

	if err = p2pNet.Init(gcfg); err != nil {
		return nil, fmt.Errorf("unable to initialize libp2p: %v", err)
	}

	trustCtx, err := did.NewTrustContextWithPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	capStoreDir := filepath.Join(gcfg.UserDir, node.CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", contextName))
	var capCtx ucan.CapabilityContext

	if _, err := os.Stat(capStoreFile); err != nil {
		if err := fs.MkdirAll(capStoreDir, os.FileMode(0o700)); err != nil {
			return nil, fmt.Errorf("unable to create capability context directory: %w", err)
		}
		// does not exist; create it
		rootDID := did.FromPublicKey(pubKey)
		capCtx, err = ucan.NewCapabilityContextWithName(contextName, trustCtx, rootDID, nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
		if err != nil {
			return nil, fmt.Errorf("unable to create capability context: %w", err)
		}

		// Save it!
		f, err := os.Create(capStoreFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create capability context file: %w", err)
		}

		err = ucan.SaveCapabilityContext(capCtx, f)
		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("unable to save capability context: %w", err)
		}
	} else {
		f, err := os.Open(capStoreFile)
		if err != nil {
			return nil, fmt.Errorf("unable to open capability context: %w", err)
		}

		capCtx, err = ucan.LoadCapabilityContextWithName(contextName, trustCtx, f)
		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("unable to load capability context: %w", err)
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

	hostID := p2pNet.Host.ID().String()
	node, err := node.New(*gcfg, afero.Afero{Fs: fs}, onboardingManager,
		capCtx, hostID, p2pNet, resourceManager, cfg.Scheduler, hardwareManager,
		orchestR, geoip2db, hostLocation, portConfig,
		contractR,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %s", err)
	}

	// initialize rest api server
	restConfig := api.ServerConfig{
		P2P:         p2pNet,
		Onboarding:  onboardingManager,
		Resource:    resourceManager,
		Middlewares: nil,
		Port:        gcfg.Rest.Port,
		Addr:        gcfg.Rest.Addr,
	}

	// Add APM middleware by appending to restConfig.MidW
	restConfig.Middlewares = append(restConfig.Middlewares, apmgin.Middleware(gin.Default()))

	rServer := api.NewServer(&restConfig)
	rServer.SetupRoutes()

	return &DMS{
		P2P:        p2pNet,
		Node:       node,
		RestServer: rServer,
	}, nil
}

func (d *DMS) Run() error {
	if err := d.P2P.Start(); err != nil {
		return fmt.Errorf("unable to start libp2p: %v", err)
	}

	err := d.Node.Start()
	if err != nil {
		return fmt.Errorf("failed to start node: %s", err)
	}

	go func() {
		err := d.RestServer.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

func (d *DMS) Stop() {
	err := d.Node.Stop()
	if err != nil {
		log.Errorf("failed to stop node: %s", err)
	}
	err = d.P2P.Stop()
	if err != nil {
		log.Errorf("failed to stop libp2p network: %s", err)
	}
	log.Infof("Shutting down after receiving")
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

// NewDMSDB creates a clover database with all known dms collections
func NewDMSDB(path string) (*clover.DB, error) {
	return clover_db.NewDB(
		path,
		[]string{
			"free_resources",
			"request_tracker",
			"onboarded_resources",
			"machine_resources",
			"onboarding_config",
			"resource_allocation",
			"orchestrator_view",
			"gpu",
			"contract",
		},
	)
}
