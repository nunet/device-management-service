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
	"crypto/ed25519"
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
	"go.elastic.co/apm/module/apmgin/v2"

	"gitlab.com/nunet/device-management-service/api"
	clover_db "gitlab.com/nunet/device-management-service/db/clover"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/gateway/provider"
	"gitlab.com/nunet/device-management-service/gateway/provider/local"
	gatewastore "gitlab.com/nunet/device-management-service/gateway/store"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs/controller"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/payment"
	payment_quote "gitlab.com/nunet/device-management-service/tokenomics/store/payment_quote"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/sys"
)

//go:embed node/data/GeoLite2-Country.mmdb
var geoLite2Country []byte

type DMS struct {
	P2P        *libp2p.Libp2p
	Node       *node.Node
	RestServer *api.Server
}

type dmsStores struct {
	contractStore            *store.Store
	paymentsStore            *payment.Store
	usageStore               *usage.Store
	txStore                  *transaction.Store
	paymentQuoteStore        *payment_quote.Store
	provisionedResourceStore *gatewastore.Store
}

type nodeParams struct {
	cfg              *config.Config
	fs               afero.Fs
	onboardingMan    *onboarding.Onboarding
	resourceMan      *resources.DefaultManager
	p2pNet           *libp2p.Libp2p
	trustCtx         did.TrustContext
	capCtx           ucan.CapabilityContext
	geoIP2DB         *geoip2.Reader
	volumeController *controller.GlusterController
	stores           *dmsStores
	deploymentStore  orchestrator.DeploymentStore
	providerRegistry *provider.Registry
}

type serverParams struct {
	config        *config.Config
	onboardingMan *onboarding.Onboarding
	resourceMan   *resources.DefaultManager
	p2pNet        *libp2p.Libp2p
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
	maskedForLog := gcfg
	if maskedForLog.APM.APIKey != "" || maskedForLog.Observability.Elastic.APIKey != "" {
		maskedForLog.APM.APIKey = "****"
		maskedForLog.Observability.Elastic.APIKey = "****"
	}
	log.Debugf("starting dms with config: %+v", maskedForLog)

	if contextName == "" {
		contextName = node.DefaultContextName
	}

	gcfg = prepareConfig(gcfg, env)
	initialize(fs, gcfg, env)

	volumeController, err := initStorage(gcfg)
	if err != nil {
		return nil, err
	}

	geoip2db, err := geoip2.FromBytes(geoLite2Country)
	if err != nil {
		return nil, fmt.Errorf("unable to load geoip2 database: %w", err)
	}
	log.Debugf("loaded geoip2 database: %v", geoip2db)

	privK, err := initCrypto(fs, gcfg, ksPassphrase, contextName)
	if err != nil {
		return nil, err
	}

	pubKey := privK.GetPublic()

	db, err := NewDMSDB(gcfg.General.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	stores, err := initStores(db)
	if err != nil {
		return nil, err
	}

	resourceManager, onboardingManager, deploymentStore, err := initManagers(db)
	if err != nil {
		return nil, err
	}

	p2pNet, trustCtx, capCtx, err := initNetwork(fs, gcfg, privK, pubKey, contextName)
	if err != nil {
		return nil, err
	}

	factories := provider.NewProviderFactoryRegistry(capCtx.DID().URI)
	// add local incus to the factory
	local.RegisterFactory(factories)
	provRegistry, err := buildProviderRegistry(gcfg, factories)
	if err != nil {
		return nil, fmt.Errorf("failed to build provider registry: %w", err)
	}

	node, err := initNode(&nodeParams{
		cfg:              gcfg,
		fs:               fs,
		onboardingMan:    onboardingManager,
		resourceMan:      resourceManager,
		p2pNet:           p2pNet,
		trustCtx:         trustCtx,
		capCtx:           capCtx,
		geoIP2DB:         geoip2db,
		volumeController: volumeController,
		stores:           stores,
		deploymentStore:  deploymentStore,
		providerRegistry: provRegistry,
	})
	if err != nil {
		return nil, err
	}

	restServer := initHTTPServer(&serverParams{
		config:        gcfg,
		onboardingMan: onboardingManager,
		resourceMan:   resourceManager,
		p2pNet:        p2pNet,
	})

	return &DMS{
		P2P:        p2pNet,
		Node:       node,
		RestServer: restServer,
	}, nil
}

func prepareConfig(gcfg *config.Config, env env.EnvironmentProvider) *config.Config {
	// Clone the config to avoid modifying the original
	cfg := *gcfg

	// if bootstrap peers were passed by env var then override them
	btPeers := env.Getenv("BOOTSTRAP_PEERS")
	if btPeers != "" {
		peers := strings.Split(btPeers, ",")
		cfg.P2P.BootstrapPeers = peers
	}

	return &cfg
}

func initStorage(gcfg *config.Config) (*controller.GlusterController, error) {
	if !gcfg.StorageMode {
		return nil, nil
	}

	volumeController, err := controller.NewGlusterController(gcfg.StorageGlusterfsHostname, gcfg.StorageBricksDir, gcfg.StorageCADirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create glusterfs controller: %w", err)
	}

	if !volumeController.IsServerWorking() {
		return nil, errors.New("failed to start in storage mode")
	}

	return volumeController, nil
}

func initCrypto(fs afero.Fs, gcfg *config.Config, ksPassphrase, contextName string) (crypto.PrivKey, error) {
	keyStoreDir := filepath.Join(gcfg.UserDir, node.KeystoreDir)
	keyStore, err := keystore.New(fs, keyStoreDir, false)
	if err != nil {
		return nil, fmt.Errorf("unable to create keystore: %w", err)
	}

	privK, err := GetPrivKeyFromKS(keyStore, ksPassphrase, contextName)
	if err != nil {
		return nil, fmt.Errorf("private key from keystore: %w", err)
	}

	return privK, nil
}

func initStores(db *clover.DB) (*dmsStores, error) {
	contractStore, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create contract store: %w", err)
	}

	paymentsStore, err := payment.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create payment store: %w", err)
	}

	usageStore, err := usage.New(db)
	if err != nil {
		return nil, fmt.Errorf("unable to create usage store: %w", err)
	}

	txStore, err := transaction.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction store: %w", err)
	}

	paymentQuoteStore, err := payment_quote.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment quote store: %w", err)
	}

	provisionedResourceStore, err := gatewastore.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare gateway store: %w", err)
	}

	stores := &dmsStores{
		contractStore:            contractStore,
		paymentsStore:            paymentsStore,
		usageStore:               usageStore,
		txStore:                  txStore,
		paymentQuoteStore:        paymentQuoteStore,
		provisionedResourceStore: provisionedResourceStore,
	}

	return stores, nil
}

func initManagers(db *clover.DB) (*resources.DefaultManager, *onboarding.Onboarding, orchestrator.DeploymentStore, error) {
	hardwareManager := hardware.NewHardwareManager()
	repos := resources.ManagerRepos{
		OnboardedResources: clover_db.NewGenericEntityRepository[types.OnboardedResources](db),
		ResourceAllocation: clover_db.NewGenericRepository[types.ResourceAllocation](db),
	}
	resourceManager, err := resources.NewResourceManager(repos, hardwareManager)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to create resource manager: %w", err)
	}

	onboardRepo := clover_db.NewGenericEntityRepository[types.OnboardingConfig](db)

	deploymentStore, err := orchestrator.NewCloverDeploymentStore(db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to create deployment store: %w", err)
	}

	onboardingManager, err := onboarding.New(context.Background(), resourceManager, hardwareManager, onboardRepo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to create onboarding manager: %w", err)
	}

	return resourceManager, onboardingManager, deploymentStore, nil
}

func initNetwork(fs afero.Fs, gcfg *config.Config, privK crypto.PrivKey, pubKey crypto.PubKey, contextName string) (*libp2p.Libp2p, did.TrustContext, ucan.CapabilityContext, error) {
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
		return nil, nil, nil, fmt.Errorf("unable to create libp2p instance: %v", err)
	}

	if err = p2pNet.Init(gcfg); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to initialize libp2p: %v", err)
	}

	trustCtx, err := did.NewTrustContextWithPrivateKey(privK)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	capStoreDir := filepath.Join(gcfg.UserDir, node.CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", contextName))

	// Check if capability context exists and if it uses a PRISM DID
	// If so, add PRISM provider to trust context before loading
	if _, err := fs.Stat(capStoreFile); err == nil {
		// File exists, check if it's a PRISM DID
		prismDIDStr, err := node.GetPrismDID(fs, gcfg.UserDir, contextName)
		if err == nil && prismDIDStr != "" {
			// This context has a PRISM DID association
			prismDID, err := did.FromString(prismDIDStr)
			if err == nil {
				// Create PRISM provider and add to trust context
				prismProvider, err := did.ProviderFromPRISMPrivateKey(prismDID, privK)
				if err == nil {
					trustCtx.AddProvider(prismProvider)
				}
			}
		}
	}

	capCtx, err := LoadOrCreateCapCtx(fs, capStoreFile, trustCtx, contextName, pubKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to load or create capability context: %w", err)
	}

	trustCtx.Start(time.Hour)
	capCtx.Start(5 * time.Minute)

	return p2pNet, trustCtx, capCtx, nil
}

func initNode(params *nodeParams) (*node.Node, error) {
	hostLocation := geolocation.Geolocation{
		Continent: params.cfg.HostContinent,
		Country:   params.cfg.HostCountry,
		City:      params.cfg.HostCity,
	}

	portConfig := node.PortConfig{
		AvailableRangeFrom: params.cfg.PortAvailableRangeFrom,
		AvailableRangeTo:   params.cfg.PortAvailableRangeTo,
	}

	volumeTracker := storage.NewVolumeTracker()

	if params.cfg.General.ComputeGateway {
		if os.Getenv("DMS_BINARY_PATH") == "" {
			return nil, fmt.Errorf("DMS_BINARY_PATH env var not set: compute gateway needs absolute path to dms binary")
		}
	}

	hostID := params.p2pNet.Host.ID().String()
	hardwareManager := hardware.NewHardwareManager()
	node, err := node.New(*params.cfg, afero.Afero{Fs: params.fs}, params.onboardingMan,
		params.capCtx, hostID, params.p2pNet, params.resourceMan, backgroundtasks.NewScheduler(10, 1*time.Second), hardwareManager,
		params.geoIP2DB, hostLocation, portConfig, volumeTracker,
		params.volumeController,
		params.stores.contractStore,
		params.stores.paymentsStore,
		params.stores.usageStore,
		params.stores.txStore,
		params.deploymentStore,
		params.providerRegistry,
		params.stores.provisionedResourceStore,
		params.stores.paymentQuoteStore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %s", err)
	}

	return node, nil
}

func initHTTPServer(params *serverParams) *api.Server {
	// initialize rest api server
	restConfig := api.ServerConfig{
		P2P:         params.p2pNet,
		Onboarding:  params.onboardingMan,
		Resource:    params.resourceMan,
		Middlewares: nil,
		Port:        params.config.Rest.Port,
		Addr:        params.config.Rest.Addr,
	}

	// Add APM middleware by appending to restConfig.MidW
	restConfig.Middlewares = append(restConfig.Middlewares, apmgin.Middleware(gin.Default()))

	rServer := api.NewServer(&restConfig, params.config)
	rServer.SetupRoutes()

	return rServer
}

func buildProviderRegistry(gcfg *config.Config, factories *provider.FactoryRegistry) (*provider.Registry, error) {
	reg := provider.NewProviderRegistry()

	for _, pc := range gcfg.General.Providers {
		p, err := factories.Create(pc.Type, pc.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %q: %w", pc.Type, err)
		}

		reg.Register(p)
	}

	return reg, nil
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

	// Listen for SIGUSR1 to reload capability contexts
	go func() {
		err := d.Node.ListenForCapabilityContextsUpdates()
		if err != nil {
			log.Errorf("failed to listen for capability contexts updates: %v", err)
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

// ImportAndStorePrivKey validates the provided private key and stores it into the user's keystore.
func ImportAndStorePrivKey(ks keystore.KeyStore, rawPriv []byte, passphrase string, keyID string) (crypto.PrivKey, error) {
	privK, err := crypto.UnmarshalPrivateKey(rawPriv)
	if err != nil {
		// try to interpret as raw Ed25519 key
		if len(rawPriv) == 32 {
			// assume it's a seed
			stdPriv := ed25519.NewKeyFromSeed(rawPriv)
			privK, err = crypto.UnmarshalEd25519PrivateKey(stdPriv)
		} else if len(rawPriv) == 64 {
			// assume it's a full private key
			privK, err = crypto.UnmarshalEd25519PrivateKey(rawPriv)
		}

		if err != nil {
			return nil, fmt.Errorf("invalid private key format: %w", err)
		}
	}

	// ensure we store the key in Protobuf format, regardless of input
	marshaledPriv, err := crypto.MarshalPrivateKey(privK)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %w", err)
	}

	_, err = ks.Save(
		keyID,
		marshaledPriv,
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
			"provisioned_resources",
			"contracts_payments",
			"service_provider_transactions",
			"contracts_usage",
			"usage_metadata",
			"payment_quotes",
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
