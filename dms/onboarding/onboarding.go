package onboarding

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

var ErrMachineNotOnboarded = errors.New("machine is not onboarded")

type Config struct {
	Fs              afero.Afero
	WorkDir         string
	DatabasePath    string
	ParamsRepo      repositories.OnboardingParams
	P2PRepo         repositories.Libp2pInfo
	ResourceManager types.ResourceManager
	AvResourceRepo  repositories.AvailableResources
	UUIDRepo        repositories.MachineUUID
}

// NewConfig is a constructor for Config
func NewConfig(
	fs afero.Afero,
	workDir, dbPath string,
	onboardingRepo repositories.OnboardingParams,
	p2pRepo repositories.Libp2pInfo,
	avResourceRepo repositories.AvailableResources,
	uuidRepo repositories.MachineUUID,
) *Config {
	return &Config{
		Fs:             fs,
		WorkDir:        workDir,
		DatabasePath:   dbPath,
		ParamsRepo:     onboardingRepo,
		P2PRepo:        p2pRepo,
		AvResourceRepo: avResourceRepo,
		UUIDRepo:       uuidRepo,
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
func (o *Onboarding) Onboard(ctx context.Context, capacity types.CapacityForNunet) (*types.OnboardingConfig, error) {
	if err := o.validateOnboardingPrerequisites(capacity); err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %v", err)
	}

	machineResources, err := o.ResourceManager.SystemSpecs().GetMachineResources()
	if err != nil {
		return nil, fmt.Errorf("cannot get provisioned resources: %w", err)
	}

	var oConf types.OnboardingConfig
	oConf.Name = hostname
	oConf.UpdateTimestamp = time.Now().Unix()
	oConf.TotalResources.RAM = machineResources.RAM
	oConf.TotalResources.CPU = machineResources.CPU

	// TODO: refactor on !531 and !563 pending merge
	//       set the other fields in RAM and CPU
	oConf.OnboardedResources.RAM = types.RAM{Size: capacity.Memory * 1024 * 1024 * 1024} // convert memory to bytes
	oConf.OnboardedResources.CPU = types.CPU{Cores: float32(capacity.CPU)}

	oConf.PublicKey = capacity.PaymentAddress
	oConf.NTXPricePerMinute = capacity.NTXPricePerMinute

	savedConfig, err := o.ParamsRepo.Save(context.Background(), oConf)
	if err != nil {
		return nil, fmt.Errorf("could not save onboarding params: %w", err)
	}

	// TODO: call the resource manager directly instead
	if err := o.updateAvailableResources(ctx, capacity); err != nil {
		return nil, fmt.Errorf("failed to update available resources: %w", err)
	}

	_, err = o.P2PRepo.Save(ctx, types.Libp2pInfo{
		ServerMode: capacity.ServerMode,
		Available:  capacity.IsAvailable,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to save libp2pInfo: %w", err)
	}

	return &savedConfig, nil
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

	// TODO: refactor on !531 and !563 pending merge
	//       set the other fields in RAM and CPU
	params.OnboardedResources.RAM = types.RAM{Size: capacity.Memory * 1024 * 1024 * 1024} // convert memory to bytes
	params.OnboardedResources.CPU = types.CPU{Cores: float32(capacity.CPU)}

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

	// TODO: change the way the resources are being onboarded
	// _, err = o.ResourceManager.UpdateFreeResources(ctx)
	// if err != nil {
	//	 return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	// }

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
	machineResources, err := o.ResourceManager.SystemSpecs().GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	if capacity.CPU < 1 || capacity.CPU >= int64(machineResources.CPU.Cores) {
		return fmt.Errorf("CPU should be between 1 and %d cores", int64(machineResources.CPU.Cores-1))
	}

	memInGB := machineResources.RAM.Size / (1024 * 1024 * 1024)

	if capacity.Memory < memInGB/10 || capacity.Memory > memInGB*9/10 {
		return fmt.Errorf("memory should be between 10%% and 90%% of the available memory in GigaBytes (%dGB and %dGB)", memInGB/10, memInGB*9/10)
	}

	return nil
}

func (o *Onboarding) validateOnboardingPrerequisites(capacity types.CapacityForNunet) error {
	ok, err := o.Fs.DirExists(o.WorkDir)
	if err != nil {
		return fmt.Errorf("could not check if config directory exists: %w", err)
	}
	if !ok {
		return fmt.Errorf("working directory does not exist")
	}

	if err := utils.ValidateAddress(capacity.PaymentAddress); capacity.PaymentAddress != "" && err != nil {
		return fmt.Errorf("could not validate payment address: %w", err)
	}

	if err := o.validateCapacityForNunet(capacity); err != nil {
		return fmt.Errorf("could not validate capacity data: %w", err)
	}

	return nil
}

func (o *Onboarding) updateAvailableResources(ctx context.Context, capacity types.CapacityForNunet) error {
	machineResources, err := o.ResourceManager.SystemSpecs().GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	avalRes := types.AvailableResources{
		TotCPUHz:          capacity.CPU,
		CPUNo:             int(machineResources.CPU.Cores),
		CPUHz:             machineResources.CPU.ClockSpeed,
		PriceCPU:          0, // TODO: Get price of CPU
		RAM:               capacity.Memory,
		PriceRAM:          0, // TODO: Get price of RAM
		Vcpu:              int(float64(capacity.CPU) / machineResources.CPU.ClockSpeed),
		Disk:              0,
		PriceDisk:         0,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
	}

	_, err = o.AvResourceRepo.Save(ctx, avalRes)
	if err != nil {
		return fmt.Errorf("failed to save available resources: %w", err)
	}

	// if _, err := o.ResourceManager.UpdateFreeResources(ctx); err != nil {
	// 	 zlog.Sugar().Errorf("could not calculate free resources and update database: %w", err)
	// }
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
