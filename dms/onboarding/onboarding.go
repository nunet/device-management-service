package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"

	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/spf13/afero"
)

type OnboardingConfig struct {
	Filesystem     afero.Afero
	MetadataPath   string
	DatabasePath   string
	P2PRepo        repositories.Libp2pInfoRepository
	AvResourceRepo repositories.AvailableResourcesRepository
	UUIDRepo       repositories.MachineUUIDRepository
	Channels       []string // supported channels such as nunet-test and nunet-team
}

type Onboarding struct {
	config OnboardingConfig
}

func New(config OnboardingConfig) *Onboarding {
	return &Onboarding{config: config}
}

// Metadata reads metadata file
// It returns a models.Metadata struct and any error if encountered
func (o *Onboarding) Metadata() (*models.Metadata, error) {
	content, err := o.config.Filesystem.ReadFile(o.config.MetadataPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read metadata file: %w", err)
	}
	var metadata models.Metadata
	err = json.Unmarshal(content, &metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal metadata: %w", err)
	}
	return &metadata, nil
}

// IsOnboarded checks libp2p and metadata related information
// It returns whether the machine is onboarded or not and any error if encountered
func (o *Onboarding) IsOnboarded(ctx context.Context) (bool, error) {
	p2p, err := o.config.P2PRepo.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("could not get libp2p info: %w", err)
	}
	if p2p.PrivateKey == nil {
		return false, fmt.Errorf("private key is not set")
	}
	_, err = o.Metadata()
	if err != nil {
		return false, fmt.Errorf("could not read metadata: %w", err)
	}
	return true, nil
}

// Status return additional information about onboarding status
// It returns a *models.OnboardingStatus and any error if encountered
func (o *Onboarding) Status(ctx context.Context) (*models.OnboardingStatus, error) {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not check onboard status: %w", err)
	}
	machine, err := o.config.UUIDRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get machine UUID: %w", err)
	}
	resp := models.OnboardingStatus{
		Onboarded:    onboarded,
		Error:        err,
		MachineUUID:  machine.UUID,
		MetadataPath: o.config.DatabasePath,
		DatabasePath: o.config.MetadataPath,
	}
	return &resp, nil
}

// Onboard validates host information, creates metadata file and update reserved resources for the network
// It returns a *models.Metadata and any error if encountered
func (o *Onboarding) Onboard(ctx context.Context, capacity models.CapacityForNunet) (*models.Metadata, error) {
	if err := o.validateOnboardingPrerequisites(capacity); err != nil {
		return nil, err
	}

	metadata, err := o.createMetadata(capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata: %w", err)
	}

	if err := o.saveMetadata(metadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := o.updateAvailableResources(ctx, capacity); err != nil {
		return nil, fmt.Errorf("failed to update available resources: %w", err)
	}

	// TODO: START NETWORKING AND OTHER WORKERS FOR THE NODE
	return nil, errors.New("NOT YET IMPLEMENTED")
}

// ResourceConfig checks if user is onboarded and updates onboarding information
// It returns a *models.Metadata and any error if encountered
func (o *Onboarding) ResourceConfig(ctx context.Context, capacity models.CapacityForNunet) (*models.Metadata, error) {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not check onboard status: %w", err)
	}
	if !onboarded {
		return nil, fmt.Errorf("machine is not onboarded")
	}

	if err := validateCapacityForNunet(capacity); err != nil {
		return nil, fmt.Errorf("could not validate capacity data: %w", err)
	}

	metadata, err := o.Metadata()
	if err != nil {
		return nil, fmt.Errorf("could not read metadata file: %w", err)
	}

	metadata.Reserved.CPU = capacity.CPU
	metadata.Reserved.Memory = capacity.Memory
	metadata.NTXPricePerMinute = capacity.NTXPricePerMinute

	available, err := o.config.AvResourceRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get available resources info: %w", err)
	}

	available.TotCpuHz = int(capacity.CPU)
	available.Ram = int(capacity.Memory)
	available.NTXPricePerMinute = capacity.NTXPricePerMinute

	if _, err := o.config.AvResourceRepo.Save(ctx, available); err != nil {
		return nil, fmt.Errorf("could not save available resources info: %w", err)
	}

	if err := o.saveMetadata(metadata); err != nil {
		return nil, fmt.Errorf("could not save metadata: %w", err)
	}

	// TODO: Replace with ResourceManager and Repository
	err = resources.CalcFreeResAndUpdateDB()
	if err != nil {
		return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	}

	return metadata, nil
}

// Offboard deletes all onboarding information if already set
// It returns an error
func (o *Onboarding) Offboard(ctx context.Context, force bool) error {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil && !force {
		return fmt.Errorf("could not retrieve onboard status: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("problem with onboarding state: %v", err)
		zlog.Info("continuing with offboarding because forced")
	}

	if !onboarded {
		return fmt.Errorf("machine is not onboarded")
	}

	// err = libp2p.ShutdownNode()
	// if err != nil {
	// 	return fmt.Errorf("unable to shutdown node: %w", err)
	// }
	return errors.New("ShutdownNode is not implemented")

	metadataPath := utils.GetMetadataFilePath()
	err = os.Remove(metadataPath)
	if err != nil && !force {
		return fmt.Errorf("failed to remove metadata file: %w", err)
	} else if err != nil && force {
		zlog.Sugar().Errorf("failed to delete metadata file - problem with onboarding state: %v", err)
		zlog.Info("continuing with offboarding because forced")
	}

	// delete the available resources from database
	var aval models.AvailableResources
	res := db.DB.WithContext(ctx).Where("id = ?", 1).Delete(&aval)
	if res.Error != nil {
		zlog.Error(res.Error.Error())
	} else if res.RowsAffected == 0 && !force {
		zlog.Error("no rows were affected while deleting available resources")
		return fmt.Errorf("unable to delete available resources on database: %w", err)
	}

	return nil
}

func validateCapacityForNunet(capacity models.CapacityForNunet) error {
	// TODO: Replace with ResourceManager
	totalCPU := resources.GetTotalProvisioned().CPU
	totalMem := resources.GetTotalProvisioned().Memory

	if capacity.CPU > int64(totalCPU*9/10) || capacity.CPU < int64(totalCPU/10) {
		return fmt.Errorf("CPU should be between 10%% and 90%% of the available CPU (%d and %d)", int64(totalCPU/10), int64(totalCPU*9/10))
	}

	if capacity.Memory > int64(totalMem*9/10) || capacity.Memory < int64(totalMem/10) {
		return fmt.Errorf("memory should be between 10%% and 90%% of the available memory (%d and %d)", int64(totalMem/10), int64(totalMem*9/10))
	}

	return nil
}

func (o *Onboarding) validateOnboardingPrerequisites(capacity models.CapacityForNunet) error {
	ok, err := o.config.Filesystem.DirExists(o.config.MetadataPath)
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

func (o *Onboarding) createMetadata(capacity models.CapacityForNunet) (*models.Metadata, error) {
	// TODO: Replace with ResourceManager
	totalProvisioned := resources.GetTotalProvisioned()

	metadata := &models.Metadata{
		Resource: struct {
			MemoryMax int64 `json:"memory_max,omitempty"`
			TotalCore int64 `json:"total_core,omitempty"`
			CPUMax    int64 `json:"cpu_max,omitempty"`
		}{
			MemoryMax: int64(totalProvisioned.Memory),
			TotalCore: int64(totalProvisioned.NumCores),
			CPUMax:    int64(totalProvisioned.CPU),
		},
		Reserved: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			Memory: int64(capacity.Memory),
			CPU:    int64(capacity.CPU),
		},
		Available: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			Memory: int64(totalProvisioned.Memory) - capacity.Memory,
			CPU:    int64(totalProvisioned.CPU) - capacity.CPU,
		},
		Network:           capacity.Channel,
		PublicKey:         capacity.PaymentAddress,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
		AllowCardano:      capacity.Cardano,
	}

	if capacity.Cardano && capacity.Memory < 10000 || capacity.CPU < 6000 {
		return nil, fmt.Errorf("cardano node requires 10000MB of RAM and 6000MHz CPU")
	}

	gpuInfo, err := resources.CheckGPU()
	if err != nil {
		zlog.Sugar().Errorf("unable to detect GPU: %v", err)
	}
	metadata.GpuInfo = gpuInfo

	return metadata, nil
}

func (o *Onboarding) saveMetadata(metadata *models.Metadata) error {
	file, err := json.MarshalIndent(metadata, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := o.config.Filesystem.WriteFile(o.config.MetadataPath, file, 0644); err != nil {
		return fmt.Errorf("could not write to metadata file: %w", err)
	}
	return nil
}

func (o *Onboarding) updateAvailableResources(ctx context.Context, capacity models.CapacityForNunet) error {
	// TODO: Replace with ResourceManager
	totalProvisioned := resources.GetTotalProvisioned()

	avalRes := models.AvailableResources{
		TotCpuHz:          int(capacity.CPU),
		CpuNo:             int(totalProvisioned.NumCores),
		CpuHz:             resources.Hz_per_cpu(),
		PriceCpu:          0, // TODO: Get price of CPU
		Ram:               int(capacity.Memory),
		PriceRam:          0, // TODO: Get price of RAM
		Vcpu:              int(math.Floor((float64(capacity.CPU)) / resources.Hz_per_cpu())),
		Disk:              0,
		PriceDisk:         0,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
	}

	_, err := o.config.AvResourceRepo.Save(ctx, avalRes)
	if err != nil {
		return fmt.Errorf("failed to save available resources: %w", err)
	}

	// TODO: Replace with ResourceManager and Repository
	//       broken on gpu detection
	if err := resources.CalcFreeResAndUpdateDB(); err != nil {
		zlog.Sugar().Errorf("could not calculate free resources and update database: %w", err)
	}
	return nil
}

// CreatePaymentAddress generates a keypair based on the wallet type. Currently supported types: ethereum, cardano.
// TODO: This should be moved to utils-related package. It's a utility function independent of onboarding
func CreatePaymentAddress(wallet string) (*models.BlockchainAddressPrivKey, error) {
	var (
		pair *models.BlockchainAddressPrivKey
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
