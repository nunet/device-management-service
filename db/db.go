package db

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(dbPath string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", dbPath)), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	_ = database.AutoMigrate(&types.ElasticToken{})
	_ = database.AutoMigrate(&types.VirtualMachine{})
	_ = database.AutoMigrate(&types.Machine{})
	_ = database.AutoMigrate(&types.AvailableResources{})
	_ = database.AutoMigrate(&types.FreeResources{})
	_ = database.AutoMigrate(&types.PeerInfo{})
	_ = database.AutoMigrate(&types.Services{})
	_ = database.AutoMigrate(&types.ServiceResourceRequirements{})
	_ = database.AutoMigrate(&types.ContainerImages{})
	_ = database.AutoMigrate(&types.RequestTracker{})
	_ = database.AutoMigrate(&types.Libp2pInfo{})
	_ = database.AutoMigrate(&types.DeploymentRequestFlat{})
	_ = database.AutoMigrate(&types.MachineUUID{})
	_ = database.AutoMigrate(&types.Connection{})
	_ = database.AutoMigrate(&types.OnboardedResources{})
	_ = database.AutoMigrate(&types.MachineResources{})
	_ = database.AutoMigrate(&types.OnboardingConfig{})
	_ = database.AutoMigrate(&types.RequiredResources{})

	// TODO remove once all DB usage is transitioned to the repos
	DB = database

	return database, nil
}
