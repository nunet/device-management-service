package onboarding

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/db"
	repositories_gorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const tmpDir = "/tmp/test"

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
		WorkDir:        "/test",
		DatabasePath:   "/test/db.sqlite",
		Channels:       []string{"test1", "test2", "test3"},
	}
	o := New(oConfig)
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
	_, err := ts.service.config.P2PRepo.Save(ctx, p2p)
	return err
}

func (ts *TestSuite) saveMachineUUID(ctx context.Context) error {
	uuid, err := utils.GenerateMachineUUID()
	if err != nil {
		return err
	}
	mUUID := types.MachineUUID{
		UUID: uuid,
	}
	_, err = ts.service.config.UUIDRepo.Save(ctx, mUUID)
	if err != nil {
		return err
	}

	return nil
}

func TestIsOnboarded(t *testing.T) {
	ts := NewTestSuite(t)

	ts.setupDB()

	ctx := context.Background()
	if err := ts.savePrivateKey(ctx); err != nil {
		t.Errorf("unable to save private key: %v", err)
	}

	// TODO: Add more test cases
	t.Run("happy case", func(t *testing.T) {
		onboarded, err := ts.service.IsOnboarded(ctx)
		assert.NoError(t, err)
		assert.True(t, onboarded)
	})
}

func TestStatus(t *testing.T) {
	ts := NewTestSuite(t)

	ts.setupDB()

	ctx := context.Background()
	if err := ts.savePrivateKey(ctx); err != nil {
		t.Errorf("unable to save private key: %v", err)
	}
	if err := ts.saveMachineUUID(ctx); err != nil {
		t.Errorf("unable to save machine UUID: %v", err)
	}

	// TODO: Add more test cases
	t.Run("happy case", func(t *testing.T) {
		status, err := ts.service.Status(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, status)
	})
}

// // TODO: It needs ResourceManager in order to fully test it
// func TestOnboard(t *testing.T) {}

// // TODO: It needs ResourceManager in order to fully test it
// func TestResourceConfig(t *testing.T) {}
func TestOnboard(t *testing.T) {
	ctx := context.Background()
	capacity := types.CapacityForNunet{
		CPU:               8000,
		Memory:            16000,
		Channel:           "test",
		PaymentAddress:    "0x1234567890abcdef",
		NTXPricePerMinute: 10,
		Cardano:           false,
	}

	testFS := afero.Afero{Fs: afero.NewMemMapFs()}

	// Create a temporary working directory

	// nolint:gofumpt
	err := testFS.MkdirAll(tmpDir, 0755)
	assert.NoError(t, err)

	// Create a new Onboarding instance with the test options
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// XXX: only after we get rid of the global db usage everywhere
	db.DB = mockDB

	_ = mockDB.AutoMigrate(&types.AvailableResources{})

	oConfig := Config{
		Fs:             testFS,
		P2PRepo:        repositories_gorm.NewLibp2pInfo(mockDB),
		UUIDRepo:       repositories_gorm.NewMachineUUID(mockDB),
		AvResourceRepo: repositories_gorm.NewAvailableResources(mockDB),
		WorkDir:        tmpDir,
		DatabasePath:   tmpDir,
		Channels:       []string{"test"},
	}
	service := New(oConfig)

	t.Run("unhappy case - invalid cardano wallet", func(t *testing.T) {
		// Call the Onboard method
		_, err := service.Onboard(ctx, capacity)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cardano wallet address")
	})

	t.Run("happy case", func(t *testing.T) {
		// correct the cardano wallet address
		capacity.PaymentAddress = "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk"

		// TODO: update after onbaording implementation
		_, err := service.Onboard(ctx, capacity)
		assert.Error(t, err)
		assert.Equal(t, "NOT YET IMPLEMENTED", err.Error())
	})

	// TODO: more test cases once resource manager is fixed
	//       currently there're problems with gpu detection during onboard
}

func TestResourceConfig(t *testing.T) {
	ctx := context.Background()
	capacity := types.CapacityForNunet{
		CPU:               8000,
		Memory:            16000,
		Channel:           "test",
		PaymentAddress:    "0x1234567890abcdef",
		NTXPricePerMinute: 10,
		Cardano:           false,
	}

	testFS := afero.Afero{Fs: afero.NewMemMapFs()}

	// Create a temporary working directory
	tmpDir := "/tmp/test"
	// nolint:gofumpt
	err := testFS.MkdirAll(tmpDir, 0755)
	assert.NoError(t, err)

	// Create a new Onboarding instance with the test options
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// XXX: only after we get rid of the global db usage everywhere
	db.DB = mockDB

	_ = mockDB.AutoMigrate(&types.AvailableResources{})

	oConfig := Config{
		Fs:             testFS,
		P2PRepo:        repositories_gorm.NewLibp2pInfo(mockDB),
		UUIDRepo:       repositories_gorm.NewMachineUUID(mockDB),
		AvResourceRepo: repositories_gorm.NewAvailableResources(mockDB),
		WorkDir:        tmpDir,
		DatabasePath:   tmpDir,
		Channels:       []string{"test"},
	}
	service := New(oConfig)

	t.Run("unhappy case - not onboarded", func(t *testing.T) {
		// Call the ResourceConfig method
		_, err := service.ResourceConfig(ctx, capacity)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not check onboard status")
	})
	// TODO: Add more test cases when onboarding is implemented after resource manager
}

func TestOffboard(t *testing.T) {
	ctx := context.Background()
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	tmpDir := "/tmp/test"
	// nolint:gofumpt
	err := fs.MkdirAll(tmpDir, 0755)
	assert.NoError(t, err)
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	oConfig := Config{
		Fs:             fs,
		P2PRepo:        repositories_gorm.NewLibp2pInfo(mockDB),
		UUIDRepo:       repositories_gorm.NewMachineUUID(mockDB),
		AvResourceRepo: repositories_gorm.NewAvailableResources(mockDB),
		WorkDir:        tmpDir,
		DatabasePath:   tmpDir,
		Channels:       []string{"test"},
	}
	service := New(oConfig)

	t.Run("unhappy case - not onboarded", func(t *testing.T) {
		err := service.Offboard(ctx, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not retrieve onboard status")
		// assert.Contains(t, err.Error(), "machine is not onboarded")
	})

	// TODO: Add more test cases when onboarding is implemented after resource manager
}
