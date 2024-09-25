package onboarding

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	repositories_gorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
)

const testWalletAddress = "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk"

type TestSuite struct {
	service *Onboarding
	db      *gorm.DB
	fs      afero.Fs
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	fs := afero.NewMemMapFs()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	service := NewTestService(db, fs)

	return &TestSuite{
		service: service,
		db:      db,
		fs:      fs,
	}
}

func NewTestService(db *gorm.DB, fs afero.Fs) *Onboarding {
	oConfig := Config{
		Fs:             afero.Afero{Fs: fs},
		P2PRepo:        repositories_gorm.NewLibp2pInfo(db),
		UUIDRepo:       repositories_gorm.NewMachineUUID(db),
		AvResourceRepo: repositories_gorm.NewAvailableResources(db),
		ParamsRepo:     repositories_gorm.NewOnboardingParams(db),
		ResourceManager: resources.NewResourceManager(resources.ManagerRepos{
			FreeResources:      repositories_gorm.NewFreeResources(db),
			OnboardedResources: repositories_gorm.NewOnboardedResources(db),
			ResourceAllocation: repositories_gorm.NewResourceAllocation(db),
		}),
		WorkDir:      "/test",
		DatabasePath: "/test/db.sqlite",
	}
	o := New(&oConfig)
	return o
}

func (ts *TestSuite) setupDB() {
	_ = ts.db.AutoMigrate(&types.AvailableResources{})
	_ = ts.db.AutoMigrate(&types.Libp2pInfo{})
	_ = ts.db.AutoMigrate(&types.MachineUUID{})
}

func (ts *TestSuite) savePrivateKey(ctx context.Context) error {
	p2p := types.Libp2pInfo{
		PrivateKey: []byte("1234"),
	}
	_, err := ts.service.P2PRepo.Save(ctx, p2p)
	return err
}

func TestIsOnboarded(t *testing.T) {
	ts := NewTestSuite(t)

	ts.setupDB()

	ctx := context.Background()
	if err := ts.savePrivateKey(ctx); err != nil {
		t.Errorf("unable to save private key: %v", err)
	}

	t.Run("happy case", func(t *testing.T) {
		onboarded, err := ts.service.IsOnboarded(ctx)
		assert.NoError(t, err)
		assert.False(t, onboarded)
	})
}

func TestOnboard(t *testing.T) {
	ts := NewTestSuite(t)

	ts.setupDB()

	ctx := context.Background()

	total, err := ts.service.ResourceManager.SystemSpecs().GetMachineResources()
	require.NoError(t, err)

	capacity := types.CapacityForNunet{
		CPU:               3,
		Memory:            4,
		PaymentAddress:    "0x1234567890abcdef",
		NTXPricePerMinute: 10,
		ServerMode:        true,
		IsAvailable:       true,
	}

	t.Run("unhappy case - config dir doesn't exist", func(t *testing.T) {
		config, err := ts.service.Onboard(ctx, capacity)
		assert.Nil(t, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "working directory does not exist")
	})

	t.Run("unhappy case - invalid payment address", func(t *testing.T) {
		err = ts.fs.Mkdir(ts.service.WorkDir, 0o755)
		require.NoError(t, err)

		capacity.PaymentAddress = "invalid"

		config, err := ts.service.Onboard(ctx, capacity)
		assert.Nil(t, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not validate payment address")
	})

	t.Run("unhappy case - insufficient memory", func(t *testing.T) {
		capacity.PaymentAddress = testWalletAddress
		capacity.CPU = int64(total.CPU.Cores) / 2                      // 50% of total CPU cores
		capacity.Memory = (total.RAM.Size / (1024 * 1024 * 1024)) / 20 // 5% of total RAM

		config, err := ts.service.Onboard(ctx, capacity)
		assert.Nil(t, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory should be between")
	})

	t.Run("unhappy case - insufficient cpu", func(t *testing.T) {
		capacity.Memory = (total.RAM.Size / (1024 * 1024 * 1024)) / 2 // 50% of total RAM
		capacity.CPU = int64(total.CPU.Cores) / 20                    // 5% of total CPU cores

		config, err := ts.service.Onboard(ctx, capacity)
		assert.Nil(t, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CPU should be between")
	})

	t.Run("unhappy case - unmigrated schema", func(t *testing.T) {
		capacity.Memory = (total.RAM.Size / (1024 * 1024 * 1024)) / 2 // 50% of total RAM
		capacity.CPU = int64(total.CPU.Cores) / 2                     // 50% of total CPU cores
		config, err := ts.service.Onboard(ctx, capacity)
		assert.Nil(t, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such table")
	})

	t.Run("happy case", func(t *testing.T) {
		capacity.Memory = (total.RAM.Size / (1024 * 1024 * 1024)) / 2 // 50% of total RAM
		capacity.CPU = int64(total.CPU.Cores) / 2                     // 50% of total CPU

		// migrate tables
		err = ts.db.AutoMigrate(&types.OnboardingConfig{})
		require.NoError(t, err)
		err = ts.db.AutoMigrate(&types.MachineResources{})
		require.NoError(t, err)
		err = ts.db.AutoMigrate(&types.OnboardedResources{})
		require.NoError(t, err)

		config, err := ts.service.Onboard(ctx, capacity)
		assert.NotNil(t, config)
		assert.NoError(t, err)
		assert.Equal(t, capacity.PaymentAddress, config.PublicKey)
	})
}
