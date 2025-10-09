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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/oschwald/geoip2-golang"
	clover "github.com/ostafen/clover/v2"
	"github.com/spf13/afero"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/utils/sys"
	"go.elastic.co/apm/module/apmgin/v2"

	"gitlab.com/nunet/device-management-service/api"
	clover_db "gitlab.com/nunet/device-management-service/db/clover"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/dms/resources"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/payment"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/types"
)

//go:embed node/data/GeoLite2-Country.mmdb
var geoLite2Country []byte

type DMS struct {
	P2P        *libp2p.Libp2p
	Node       *node.Node
	RestServer *api.Server
}

func initialize(fs afero.Fs, cfg *config.Config, env env.EnvironmentProvider) {
	workDir := cfg.WorkDir
	if workDir != "" {
		err := fs.MkdirAll(workDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create work directory: %v", err)
		}
	}

	dataDir := cfg.DataDir
	if dataDir != "" {
		err := fs.MkdirAll(dataDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create data directory: %v", err)
		}
	}

	userDir := cfg.UserDir
	if userDir != "" {
		err := fs.MkdirAll(userDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create user directory: %v", err)
		}
	}

	libp2pLogging := env.Getenv("DMS_CONN_LOGS")
	if libp2pLogging == "false" || libp2pLogging == "" {
		err := silenceConnLogs()
		if err != nil {
			log.Warnf("unable to set libp2p logging: %v", err)
		}
	}

	// create the iptables NUNET chain if it doesn't exist, flush any rules in there and create jump rules
	err := sys.CreateNuNetChain()
	if err != nil {
		log.Errorf("unable to create iptables NUNET chain: %v", err)
	}
	err = sys.FlushNuNetChain()
	if err != nil {
		log.Errorf("unable to flush iptables NUNET chain: %v", err)
	}
	err = sys.AddJumpRules()
	if err != nil {
		log.Errorf("unable to add iptables NUNET jump rules: %v", err)
	}
}

func NewDMS(fs afero.Fs, gcfg *config.Config, env env.EnvironmentProvider, ksPassphrase, contextName string) (*DMS, error) {
	log.Debugf("starting dms with config: %+v", gcfg)
	if contextName == "" {
		contextName = node.DefaultContextName
	}

	// if bootstrap peers were passed by env var then override them
	btPeers := env.Getenv("BOOTSTRAP_PEERS")
	if btPeers != "" {
		peers := strings.Split(btPeers, ",")
		gcfg.P2P.BootstrapPeers = peers
	}

	initialize(fs, gcfg, env)

	var volumeController *controller.GlusterController

	if gcfg.StorageMode {
		var err error
		volumeController, err = controller.NewGlusterController(gcfg.StorageGlusterfsHostname, gcfg.StorageBricksDir, gcfg.StorageCADirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to create glusterfs controller: %w", err)
		}

		if !volumeController.IsServerWorking() {
			return nil, errors.New("failed to start in storage mode")
		}
	}

	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	if err != nil {
		return nil, fmt.Errorf("unable to load geoip2 database: %w", err)
	}
	log.Debugf("loaded geoip2 database: %v", geoip2db)

	keyStoreDir := filepath.Join(gcfg.UserDir, node.KeystoreDir)
	keyStore, err := keystore.New(fs, keyStoreDir, false)
	if err != nil {
		return nil, fmt.Errorf("unable to create keystore: %w", err)
	}

	privK, err := GetPrivKeyFromKS(keyStore, ksPassphrase, contextName)
	if err != nil {
		return nil,
			fmt.Errorf("private key from keystore: %w", err)
	}

	pubKey := privK.GetPublic()

	db, err := NewDMSDB(gcfg.General.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	contractStore, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create contract store: %w", err)
	}

	// payment validator
	paymentsStore, err := payment.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create payment store: %w", err)
	}

	usageStore, err := usage.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create usage store: %w", err)
	}

	hardwareManager := hardware.NewHardwareManager()
	repos := resources.ManagerRepos{
		OnboardedResources: clover_db.NewGenericEntityRepository[types.OnboardedResources](db),
		ResourceAllocation: clover_db.NewGenericRepository[types.ResourceAllocation](db),
	}
	resourceManager, err := resources.NewResourceManager(repos, hardwareManager)
	if err != nil {
		return nil, fmt.Errorf("unable to create resource manager: %w", err)
	}

	onboardRepo := clover_db.NewGenericEntityRepository[types.OnboardingConfig](db)

	// Create deployment store for orchestrator registry
	deploymentStore, err := orchestrator.NewCloverDeploymentStore(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create deployment store: %w", err)
	}

	onboardingManager, err := onboarding.New(context.Background(), resourceManager, hardwareManager, onboardRepo)
	if err != nil {
		return nil, fmt.Errorf("unable to create onboarding manager: %w", err)
	}

	bootstrapPeers := make([]ma.Multiaddr, len(gcfg.BootstrapPeers))
	for i, addr := range gcfg.BootstrapPeers {
		bootstrapPeers[i], _ = ma.NewMultiaddr(addr)
	}

	cfg := &types.Libp2pConfig{
		Env:                     gcfg.General.Env,
		PrivateKey:              privK,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-test",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10, 1*time.Second),
		DHTPrefix:               "/nunet",
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           gcfg.P2P.ListenAddress,
		PeerCountDiscoveryLimit: 40,
		GracePeriodMs:           20000, // 20 seconds
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

	trustCtx, err := did.NewTrustContextWithPrivateKey(privK)
	if err != nil {
		return nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	capStoreDir := filepath.Join(gcfg.UserDir, node.CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", contextName))

	capCtx, err := LoadOrCreateCapCtx(
		fs, capStoreFile, trustCtx, contextName, pubKey)
	if err != nil {
		return nil,
			fmt.Errorf(
				"unable to load or create capability context: %w", err)
	}

	trustCtx.Start(time.Hour)
	capCtx.Start(5 * time.Minute)

	hostLocation := geolocation.Geolocation{
		Continent: gcfg.HostContinent,
		Country:   gcfg.HostCountry,
		City:      gcfg.HostCity,
	}

	portConfig := node.PortConfig{
		AvailableRangeFrom: gcfg.PortAvailableRangeFrom,
		AvailableRangeTo:   gcfg.PortAvailableRangeTo,
	}

	volumeTracker := storage.NewVolumeTracker()

	txStore, err := transaction.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction store: %w", err)
	}

	hostID := p2pNet.Host.ID().String()
	node, err := node.New(*gcfg, afero.Afero{Fs: fs}, onboardingManager,
		capCtx, hostID, p2pNet, resourceManager, cfg.Scheduler, hardwareManager,
		geoip2db, hostLocation, portConfig, volumeTracker,
		volumeController,
		contractStore,
		paymentsStore,
		usageStore,
		txStore,
		deploymentStore,
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

	rServer := api.NewServer(&restConfig, gcfg)
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

	err = d.Node.StartContracts()
	if err != nil {
		log.Errorf("failed to start contracts from db: %v", err)
	}

	return nil
}

func (d *DMS) Stop() {
	log.Infof("Shutting down DMS")
	if d.Node != nil {
		if err := d.Node.Stop(); err != nil {
			log.Errorf("failed to stop node: %s", err)
		}
	}
	log.Infof("node stopped")

	if d.P2P != nil {
		if err := d.P2P.Stop(); err != nil {
			log.Errorf("failed to stop libp2p network: %s", err)
		}
	}
	log.Infof("network stopped")

	observability.Shutdown()
	// TODO: stop rest server
}

// GenerateAndStorePrivKey generates a new key pair using Secp256k1,
// storing the private key into user's keystore.
func GenerateAndStorePrivKey(ks keystore.KeyStore, passphrase string, keyID string) (crypto.PrivKey, error) {
	privK, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 256)
	if err != nil {
		return nil, fmt.Errorf("unable to generate key pair: %w", err)
	}

	rawPriv, err := crypto.MarshalPrivateKey(privK)
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

	return privK, nil
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
			"deployments",
			"gpu",
			"contracts",
			"contracts_keys",
			"contracts_payments",
			"service_provider_transactions",
			"contracts_usage",
			"usage_metadata",
		},
	)
}

// GetPrivKeyFromKS returns a private key from user's keystore.
// Creates a new one if it does not exist.
func GetPrivKeyFromKS(
	keyStore keystore.KeyStore, ksPassphrase string,
	contextName string,
) (crypto.PrivKey, error) {
	var privK crypto.PrivKey
	ksPrivKey, err := keyStore.Get(contextName, ksPassphrase)
	if err != nil {
		if errors.Is(err, keystore.ErrKeyNotFound) {
			privK, err = GenerateAndStorePrivKey(keyStore, ksPassphrase, contextName)
			if err != nil {
				return nil, fmt.Errorf("couldn't generate and store privK key into keystore: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get private key from keystore; Error: %v", err)
		}
	} else {
		privK, err = ksPrivKey.PrivKey()
		if err != nil {
			return nil, fmt.Errorf("unable to convert key from keystore to private key: %v", err)
		}
	}

	return privK, nil
}

// LoadOrCreateCapCtx loads a capability context from a file or creates a new one
// if it does not exist.
//
// Note: please use afero 'fs' arg instead of 'os'
func LoadOrCreateCapCtx(
	fs afero.Fs,
	capStoreFile string,
	trustCtx did.TrustContext,
	contextName string,
	pubKey crypto.PubKey,
) (ucan.CapabilityContext, error) {
	var capCtx ucan.CapabilityContext
	if _, err := fs.Stat(capStoreFile); err != nil {
		capStoreDir := filepath.Dir(capStoreFile)
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
		f, err := fs.Create(capStoreFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create capability context file: %w", err)
		}

		err = ucan.SaveCapabilityContext(capCtx, f)
		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("unable to save capability context: %w", err)
		}
	} else {
		f, err := fs.Open(capStoreFile)
		if err != nil {
			return nil, fmt.Errorf("unable to open capability context: %w", err)
		}

		capCtx, err = ucan.LoadCapabilityContextWithName(contextName, trustCtx, f)
		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("unable to load capability context: %w", err)
		}
	}

	return capCtx, nil
}
