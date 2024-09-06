package onboarding

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/resources"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/afero"
)

var ErrMachineNotOnboarded = errors.New("machine is not onboarded")

type Config struct {
	Fs              afero.Afero
	WorkDir         string
	DatabasePath    string
	ParamsRepo      repositories.OnboardingParams
	P2PRepo         repositories.Libp2pInfo
	ResourceManager resources.Manager
	AvResourceRepo  repositories.AvailableResources
	UUIDRepo        repositories.MachineUUID
	Channels        []string // supported channels such as nunet-test and nunet-team
}

// NewConfig is a constructor for Config
func NewConfig(
	fs afero.Afero,
	workDir, dbPath string,
	onboardingRepo repositories.OnboardingParams,
	p2pRepo repositories.Libp2pInfo,
	avResourceRepo repositories.AvailableResources,
	uuidRepo repositories.MachineUUID,
	channels []string,
) *Config {
	return &Config{
		Fs:             fs,
		WorkDir:        workDir,
		DatabasePath:   dbPath,
		ParamsRepo:     onboardingRepo,
		P2PRepo:        p2pRepo,
		AvResourceRepo: avResourceRepo,
		UUIDRepo:       uuidRepo,
		Channels:       channels,
	}
}

// Onboarding acts a receiver for methods related to onboarding
type Onboarding struct {
	Config
}

// New is a constructor for Onboarding
func New(config *Config) *Onboarding {
	return &Onboarding{Config: *config}
}

// IsOnboarded checks whether the machine is onboarded or not
func (o *Onboarding) IsOnboarded(ctx context.Context) (bool, error) {
	_, err := o.ParamsRepo.Get(ctx)
	if err != nil {
		return false, nil
	}
	// TODO: validate onboarding params
	return true, nil
}

// Info returns additional info from onboarding
func (o *Onboarding) Info(ctx context.Context) (*types.OnboardingConfig, error) {
	info, err := o.ParamsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &info, err
}

// Onboard validates the onboarding params and onboards the machine to the network
// It returns a *types.OnboardingConfig and any error if encountered
func (o *Onboarding) Onboard(ctx context.Context, capacity types.CapacityForNunet) (*types.OnboardingConfig, *libp2p.Libp2p, error) {
	if err := o.validateOnboardingPrerequisites(capacity); err != nil {
		return nil, nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get hostname: %v", err)
	}

	provisionedResources, err := o.ResourceManager.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get provisioned resources: %w", err)
	}

	var oConf types.OnboardingConfig
	oConf.Name = hostname
	oConf.UpdateTimestamp = time.Now().Unix()
	// TODO: 553
	oConf.TotalResources.RAM = provisionedResources.RAM
	oConf.TotalResources.NumCores = provisionedResources.NumCores
	oConf.TotalResources.CPU = provisionedResources.CPU

	oConf.AllowCardano = false
	if capacity.Cardano {
		if capacity.Memory < 10000 || capacity.CPU < 6000 {
			return nil, nil, fmt.Errorf("cardano node requires 10000MB of RAM and 6000MHz CPU")
		}
		oConf.AllowCardano = true
	}

	gpuInfo, err := o.ResourceManager.SystemSpecs().GetGPUs()
	if err != nil {
		zlog.Sugar().Errorf("unable to detect GPU: %v ", err.Error())
	}
	oConf.GpuInfo = gpuInfo

	oConf.OnboardedResources.RAM = capacity.Memory
	oConf.OnboardedResources.CPU = float64(capacity.CPU)
	// TODO: 553
	oConf.Network = capacity.Channel
	oConf.PublicKey = capacity.PaymentAddress
	oConf.NTXPricePerMinute = capacity.NTXPricePerMinute

	savedConfig, err := o.ParamsRepo.Save(context.Background(), oConf)
	if err != nil {
		return nil, nil, fmt.Errorf("could not save onboarding params: %w", err)
	}

	if err := o.updateAvailableResources(ctx, capacity); err != nil {
		return nil, nil, fmt.Errorf("failed to update available resources: %w", err)
	}

	// initialize libp2p
	priv, pub, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate key pair: %v", err)
	}

	rawPriv, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal private key: %w", err)
	}

	rawPub, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal public key: %w", err)
	}

	_, err = o.P2PRepo.Save(ctx, types.Libp2pInfo{
		PrivateKey: rawPriv,
		PublicKey:  rawPub,
		ServerMode: capacity.ServerMode,
		Available:  capacity.IsAvailable,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to save libp2pInfo: %w", err)
	}

	bootstrapPeers := make([]multiaddr.Multiaddr, len(config.GetConfig().P2P.BootstrapPeers))
	for i, addr := range config.GetConfig().P2P.BootstrapPeers {
		bootstrapPeers[i], _ = multiaddr.NewMultiaddr(addr)
	}

	cfg := &types.Libp2pConfig{
		PrivateKey:              priv,
		BootstrapPeers:          bootstrapPeers,
		Rendezvous:              "nunet-test",
		Server:                  false,
		Scheduler:               backgroundtasks.NewScheduler(10),
		CustomNamespace:         "/nunet-dht-1/",
		ListenAddress:           config.GetConfig().ListenAddress,
		PeerCountDiscoveryLimit: 40,
	}

	p2p, err := libp2p.New(cfg, afero.NewMemMapFs())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create libp2p instance: %v", err)
	}

	if err = p2p.Init(ctx); err != nil {
		return nil, nil, fmt.Errorf("unable to initialize libp2p: %v", err)
	}

	if err = p2p.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("unable to start libp2p: %v", err)
	}

	return &savedConfig, p2p, nil
}

// ResourceConfig allows changing onboarding parameters
func (o *Onboarding) ResourceConfig(ctx context.Context, capacity types.CapacityForNunet) (*types.OnboardingConfig, error) {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not check onboard status: %w", err)
	}
	if !onboarded {
		return nil, ErrMachineNotOnboarded
	}

	if err := o.validateCapacityForNunet(capacity); err != nil {
		return nil, fmt.Errorf("could not validate capacity data: %w", err)
	}

	params, err := o.ParamsRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read onboarding params from db: %w", err)
	}

	params.OnboardedResources.CPU = float64(capacity.CPU)
	params.OnboardedResources.RAM = capacity.Memory
	params.NTXPricePerMinute = capacity.NTXPricePerMinute

	available, err := o.AvResourceRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get available resources info: %w", err)
	}

	available.TotCPUHz = capacity.CPU
	available.RAM = capacity.Memory
	available.NTXPricePerMinute = capacity.NTXPricePerMinute

	if _, err := o.AvResourceRepo.Save(ctx, available); err != nil {
		return nil, fmt.Errorf("could not save available resources info: %w", err)
	}

	if _, err := o.ParamsRepo.Save(ctx, params); err != nil {
		return nil, fmt.Errorf("could not save onboarding params in db: %w", err)
	}

	_, err = o.ResourceManager.UpdateFreeResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	}

	return &params, nil
}

// Offboard deletes all onboarding information if already set
// It returns an error
func (o *Onboarding) Offboard(ctx context.Context, force bool) error {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil && !force {
		return fmt.Errorf("could not retrieve onboard status: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("problem with onboarding state: %w", err)
		zlog.Info("continuing with offboarding because forced")
	}

	if !onboarded {
		return fmt.Errorf("machine is not onboarded")
	}

	// TODO: shutdown routine to stop networking etc... here

	err = o.ParamsRepo.Clear(ctx)
	if err != nil && !force {
		return fmt.Errorf("failed to remove onboarding params from db: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("failed to delete onboarding params from db - problem with onboarding state: %w", err)
		zlog.Info("continuing with offboarding because forced")
	}

	// delete the available resources from database
	err = o.AvResourceRepo.Clear(ctx)
	if err != nil && !force {
		return fmt.Errorf("failed to remove reserved resource from db: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("failed to delete reserved resource from db - problem with onboarding state: %w", err)
		zlog.Info("continuing with offboarding because forced")
	}

	return nil
}

func (o *Onboarding) validateCapacityForNunet(capacity types.CapacityForNunet) error {
	provisionedResources, err := o.ResourceManager.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	if capacity.CPU > int64(provisionedResources.CPU*9/10) || capacity.CPU < int64(provisionedResources.CPU/10) {
		return fmt.Errorf("CPU should be between 10%% and 90%% of the available CPU (%d and %d)", int64(provisionedResources.CPU/10), int64(provisionedResources.CPU*9/10))
	}

	//nolint:gosec // to be fixed in TODO: 553
	if capacity.Memory > provisionedResources.RAM*9/10 || capacity.Memory < provisionedResources.RAM/10 {
		return fmt.Errorf("memory should be between 10%% and 90%% of the available memory (%d and %d)", int64(provisionedResources.RAM/10), int64(provisionedResources.RAM*9/10))
	}

	return nil
}

func (o *Onboarding) validateOnboardingPrerequisites(capacity types.CapacityForNunet) error {
	ok, err := o.Fs.DirExists(o.WorkDir)
	if err != nil {
		return fmt.Errorf("could not check if config directory exists: %w", err)
	}
	if !ok {
		return fmt.Errorf("config directory does not exist")
	}

	if err := utils.ValidateAddress(capacity.PaymentAddress); err != nil {
		return fmt.Errorf("could not validate payment address: %w", err)
	}

	if err := o.validateCapacityForNunet(capacity); err != nil {
		return fmt.Errorf("could not validate capacity data: %w", err)
	}

	if !slices.Contains(o.Channels, capacity.Channel) {
		return fmt.Errorf("invalid channel data: '%s' channel does not exist", capacity.Channel)
	}

	return nil
}

func (o *Onboarding) updateAvailableResources(ctx context.Context, capacity types.CapacityForNunet) error {
	totalProvisioned, err := o.ResourceManager.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	cpuInfo, err := o.ResourceManager.SystemSpecs().GetCPUInfo()
	if err != nil {
		return fmt.Errorf("could not get CPU info: %w", err)
	}

	avalRes := types.AvailableResources{
		TotCPUHz:          capacity.CPU,
		CPUNo:             totalProvisioned.NumCores,
		CPUHz:             cpuInfo.MHzPerCore,
		PriceCPU:          0, // TODO: Get price of CPU
		RAM:               capacity.Memory,
		PriceRAM:          0, // TODO: Get price of RAM
		Vcpu:              int(math.Floor((float64(capacity.CPU)) / cpuInfo.MHzPerCore)),
		Disk:              0,
		PriceDisk:         0,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
	}

	_, err = o.AvResourceRepo.Save(ctx, avalRes)
	if err != nil {
		return fmt.Errorf("failed to save available resources: %w", err)
	}

	if _, err := o.ResourceManager.UpdateFreeResources(ctx); err != nil {
		zlog.Sugar().Errorf("could not calculate free resources and update database: %w", err)
	}
	return nil
}

// CreatePaymentAddress generates a keypair based on the wallet type. Currently supported types: ethereum, cardano.
// TODO: This should be moved to utils-related package. It's a utility function independent of onboarding
func CreatePaymentAddress(wallet string) (*types.BlockchainAddressPrivKey, error) {
	var (
		pair *types.BlockchainAddressPrivKey
		err  error
	)
	switch wallet {
	case "ethereum":
		pair, err = GetEthereumAddressAndPrivateKey()
	case "cardano":
		pair, err = GetCardanoAddressAndMnemonic()
	default:
		return nil, fmt.Errorf("invalid wallet")
	}
	if err != nil {
		return nil, fmt.Errorf("could not generate %s address: %w", wallet, err)
	}
	return pair, nil
}
