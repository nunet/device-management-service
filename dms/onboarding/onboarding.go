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
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/spf13/afero"
)

var ErrMachineNotOnboarded = errors.New("machine is not onboarded")

type Config struct {
	Fs             afero.Afero
	WorkDir        string
	DatabasePath   string
	OnboardingRepo repositories.OnboardingParams
	P2PRepo        repositories.Libp2pInfo
	AvResourceRepo repositories.AvailableResources
	UUIDRepo       repositories.MachineUUID
	Channels       []string // supported channels such as nunet-test and nunet-team
}

type Onboarding struct {
	config Config
}

func New(config Config) *Onboarding {
	return &Onboarding{config: config}
}

// IsOnboarded checks whether the machine is onboarded or not
func (o *Onboarding) IsOnboarded(ctx context.Context) (bool, error) {
	_, err := o.config.OnboardingRepo.Get(ctx)
	if err != nil {
		return false, nil
	}
	// TODO: validate onbaording params
	return true, nil
}

// Status returns onbaording status and any error if encountered
func (o *Onboarding) Status(ctx context.Context) (*types.OnboardingStatus, error) {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not check onboard status: %w", err)
	}
	machine, err := o.config.UUIDRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get machine UUID: %w", err)
	}
	resp := types.OnboardingStatus{
		Onboarded:    onboarded,
		Error:        err,
		MachineUUID:  machine.UUID,
		DatabasePath: o.config.WorkDir,
	}
	return &resp, nil
}

// Onboard validates the onboarding params and onboards the machine to the network
// It returns a *types.OnboardingConfig and any error if encountered
func (o *Onboarding) Onboard(ctx context.Context, capacity types.CapacityForNunet) (*types.OnboardingConfig, error) {
	if err := o.validateOnboardingPrerequisites(capacity); err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}

	provisionedResources, err := resources.ManagerInstance.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return nil, fmt.Errorf("cannot get provisioned resources: %w", err)
	}

	var oConf types.OnboardingConfig
	oConf.Name = hostname
	oConf.UpdateTimestamp = time.Now().Unix()
	// nolint TODO: 553
	oConf.TotalResources.RAM = provisionedResources.RAM
	oConf.TotalResources.NumCores = provisionedResources.NumCores
	oConf.TotalResources.CPU = provisionedResources.CPU

	oConf.AllowCardano = false
	if capacity.Cardano {
		if capacity.Memory < 10000 || capacity.CPU < 6000 {
			return nil, fmt.Errorf("cardano node requires 10000MB of RAM and 6000MHz CPU")
		}
		oConf.AllowCardano = true
	}

	gpuInfo, err := resources.ManagerInstance.SystemSpecs().GetGPUs()
	if err != nil {
		zlog.Sugar().Errorf("unable to detect GPU: %v ", err.Error())
	}
	oConf.GpuInfo = gpuInfo

	oConf.OnboardedResources.RAM = capacity.Memory
	oConf.OnboardedResources.CPU = float64(capacity.CPU)
	// nolint TODO: 553
	oConf.Network = capacity.Channel
	oConf.PublicKey = capacity.PaymentAddress
	oConf.NTXPricePerMinute = capacity.NTXPricePerMinute

	savedOConf, err := o.config.OnboardingRepo.Save(context.Background(), oConf)
	if err != nil {
		return nil, fmt.Errorf("could not save onboarding params: %w", err)
	}

	if err := o.updateAvailableResources(ctx, capacity); err != nil {
		return nil, fmt.Errorf("failed to update available resources: %w", err)
	}

	// TODO: START NETWORKING AND OTHER WORKERS FOR THE NODE
	return &savedOConf, errors.New("NOT YET IMPLEMENTED")
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

	if err := validateCapacityForNunet(capacity); err != nil {
		return nil, fmt.Errorf("could not validate capacity data: %w", err)
	}

	oParams, err := o.config.OnboardingRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read onboarding params from db: %w", err)
	}

	oParams.OnboardedResources.CPU = float64(capacity.CPU)
	oParams.OnboardedResources.RAM = capacity.Memory
	oParams.NTXPricePerMinute = capacity.NTXPricePerMinute

	available, err := o.config.AvResourceRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get available resources info: %w", err)
	}

	available.TotCPUHz = capacity.CPU
	available.RAM = capacity.Memory
	available.NTXPricePerMinute = capacity.NTXPricePerMinute

	if _, err := o.config.AvResourceRepo.Save(ctx, available); err != nil {
		return nil, fmt.Errorf("could not save available resources info: %w", err)
	}

	if _, err := o.config.OnboardingRepo.Save(ctx, oParams); err != nil {
		return nil, fmt.Errorf("could not save onboarding params in db: %w", err)
	}

	_, err = resources.ManagerInstance.UpdateFreeResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	}

	return &oParams, nil
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

	err = o.config.OnboardingRepo.Clear(ctx)
	if err != nil && !force {
		return fmt.Errorf("failed to remove onboarding params from db: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("failed to delete onboarding params from db - problem with onboarding state: %w", err)
		zlog.Info("continuing with offboarding because forced")
	}

	// delete the available resources from database
	err = o.config.AvResourceRepo.Clear(ctx)
	if err != nil && !force {
		return fmt.Errorf("failed to remove reserved resource from db: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("failed to delete reserved resource from db - problem with onboarding state: %w", err)
		zlog.Info("continuing with offboarding because forced")
	}

	return nil
}

func validateCapacityForNunet(capacity types.CapacityForNunet) error {
	provisionedResources, err := resources.ManagerInstance.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	if capacity.CPU > int64(provisionedResources.CPU*9/10) || capacity.CPU < int64(provisionedResources.CPU/10) {
		return fmt.Errorf("CPU should be between 10%% and 90%% of the available CPU (%d and %d)", int64(provisionedResources.CPU/10), int64(provisionedResources.CPU*9/10))
	}

	// nolint TODO: 553
	if capacity.Memory > provisionedResources.RAM*9/10 || capacity.Memory < provisionedResources.RAM/10 {
		return fmt.Errorf("memory should be between 10%% and 90%% of the available memory (%d and %d)", int64(provisionedResources.RAM/10), int64(provisionedResources.RAM*9/10))
	}

	return nil
}

func (o *Onboarding) validateOnboardingPrerequisites(capacity types.CapacityForNunet) error {
	ok, err := o.config.Fs.DirExists(o.config.WorkDir)
	if err != nil {
		return fmt.Errorf("could not check if config directory exists: %w", err)
	}
	if !ok {
		return fmt.Errorf("config directory does not exist")
	}

	if err := utils.ValidateAddress(capacity.PaymentAddress); err != nil {
		return fmt.Errorf("could not validate payment address: %w", err)
	}

	if err := validateCapacityForNunet(capacity); err != nil {
		return fmt.Errorf("could not validate capacity data: %w", err)
	}

	if !slices.Contains(o.config.Channels, capacity.Channel) {
		return fmt.Errorf("invalid channel data: '%s' channel does not exist", capacity.Channel)
	}

	return nil
}

func (o *Onboarding) updateAvailableResources(ctx context.Context, capacity types.CapacityForNunet) error {
	totalProvisioned, err := resources.ManagerInstance.SystemSpecs().GetProvisionedResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	cpuInfo, err := resources.ManagerInstance.SystemSpecs().GetCPUInfo()
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

	_, err = o.config.AvResourceRepo.Save(ctx, avalRes)
	if err != nil {
		return fmt.Errorf("failed to save available resources: %w", err)
	}

	if _, err := resources.ManagerInstance.UpdateFreeResources(ctx); err != nil {
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
