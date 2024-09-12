package cmd

import (
	"fmt"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestSuite struct {
	client *utils.HTTPClient
	db     *gorm.DB
	afs    afero.Afero
}

func NewTestSuite() *TestSuite {
	return &TestSuite{
		client: utils.NewHTTPClient("//", "v1"),
		afs:    afero.Afero{Fs: afero.NewMemMapFs()},
	}
}

func (ts *TestSuite) setup() error {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to initialize db: %w", err)
	}
	_ = db.AutoMigrate(&types.ElasticToken{})
	_ = db.AutoMigrate(&types.VirtualMachine{})
	_ = db.AutoMigrate(&types.Machine{})
	_ = db.AutoMigrate(&types.AvailableResources{})
	_ = db.AutoMigrate(&types.FreeResources{})
	_ = db.AutoMigrate(&types.PeerInfo{})
	_ = db.AutoMigrate(&types.Services{})
	_ = db.AutoMigrate(&types.ServiceResourceRequirements{})
	_ = db.AutoMigrate(&types.ContainerImages{})
	_ = db.AutoMigrate(&types.RequestTracker{})
	_ = db.AutoMigrate(&types.Libp2pInfo{})
	_ = db.AutoMigrate(&types.DeploymentRequestFlat{})
	_ = db.AutoMigrate(&types.MachineUUID{})
	_ = db.AutoMigrate(&types.Connection{})
	_ = db.AutoMigrate(&types.OnboardedResources{})
	_ = db.AutoMigrate(&types.ResourceAllocation{})

	ts.db = db
	return nil
}

func (ts *TestSuite) teardown() {
	ts.client = nil
	if ts.db != nil {
		db, err := ts.db.DB()
		if err != nil {
			return
		}
		if err := db.Close(); err != nil {
			return
		}
		ts.db = nil
	}
	ts.afs = afero.Afero{}
}
