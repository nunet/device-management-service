package onboarding

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	repositories_gorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/types"
)

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

	service := NewTestService(db, fs, t)

	return &TestSuite{
		service: service,
		db:      db,
		fs:      fs,
	}
}

func NewTestService(db *gorm.DB, fs afero.Fs, t *testing.T) *Onboarding {
	t.Helper()

	ctrl := gomock.NewController(t)
	oConfig := Config{
		Fs:              afero.Afero{Fs: fs},
		P2PRepo:         repositories_gorm.NewLibp2pInfo(db),
		UUIDRepo:        repositories_gorm.NewMachineUUID(db),
		ConfigRepo:      repositories_gorm.NewOnboardingConfig(db),
		ResourceManager: NewMockResourceManager(ctrl),
		Hardware:        NewMockHardwareManager(ctrl),
	}
	workDir, err := oConfig.Fs.TempDir("test", "")
	require.NoError(t, err)
	oConfig.WorkDir = workDir

	dbDir, err := oConfig.Fs.TempDir("testdb", "")
	require.NoError(t, err)
	oConfig.DatabasePath = dbDir + "/test.db"

	o := New(&oConfig)
	return o
}

func (ts *TestSuite) setupDB() {
	_ = ts.db.AutoMigrate(&types.OnboardingConfig{})
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
		assert.Error(t, err)
		assert.False(t, onboarded)
	})
}

func TestOnboard(t *testing.T) {
	t.Parallel()

	t.Run("unhappy case - config dir doesn't exist", func(t *testing.T) {
		t.Parallel()
		ts := NewTestSuite(t)
		ts.setupDB()
		ctx := context.Background()

		config := types.OnboardingConfig{}
		ts.service.WorkDir = "/non/existent/dir"
		err := ts.service.Onboard(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "working directory does not exist")
	})

	t.Run("unhappy case - insufficient memory", func(t *testing.T) {
		t.Parallel()
		ts := NewTestSuite(t)
		ts.setupDB()
		ctx := context.Background()

		config := types.OnboardingConfig{
			OnboardedResources: types.Resources{
				CPU: types.CPU{Cores: 2, ClockSpeed: 1000},
				RAM: types.RAM{
					Size: 0.00001, // 0.00001 GB
				},
			},
		}
		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 1000,
				},
				RAM: types.RAM{Size: 100000},
			},
		}
		ts.service.Hardware.(*MockHardwareManager).EXPECT().GetMachineResources().Return(machineResources, nil)
		err := ts.service.Onboard(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory should be between")
	})

	t.Run("unhappy case - insufficient cpu", func(t *testing.T) {
		t.Parallel()
		ts := NewTestSuite(t)
		ts.setupDB()
		ctx := context.Background()

		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 1000,
				},
			},
		}
		ts.service.Hardware.(*MockHardwareManager).EXPECT().GetMachineResources().Return(machineResources, nil)

		config := types.OnboardingConfig{
			OnboardedResources: types.Resources{
				CPU: types.CPU{Cores: 0.00001},
			},
		}
		err := ts.service.Onboard(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cores must be between")
	})

	t.Run("happy case", func(t *testing.T) {
		t.Parallel()
		ts := NewTestSuite(t)
		ts.setupDB()
		ctx := context.Background()

		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      5,
					ClockSpeed: 1000,
				},
				RAM: types.RAM{
					Size: 100000,
				},
				Disk: types.Disk{
					Size: 100000,
				},
			},
		}
		ts.service.Hardware.(*MockHardwareManager).EXPECT().GetMachineResources().Return(machineResources, nil)

		config := types.OnboardingConfig{
			OnboardedResources: types.Resources{
				CPU:  types.CPU{Cores: 2, ClockSpeed: 1000},
				RAM:  types.RAM{Size: 10000},
				Disk: types.Disk{Size: 10000},
			},
		}
		ts.service.ResourceManager.(*MockResourceManager).EXPECT().
			UpdateOnboardedResources(ctx, config.OnboardedResources).Return(nil)
		err := ts.service.Onboard(ctx, config)
		assert.NoError(t, err)
	})
}
