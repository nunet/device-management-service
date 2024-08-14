package db

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir)), &gorm.Config{})

	if err != nil {
		panic("Failed to connect to database!")
	}

	database.AutoMigrate(&types.ElasticToken{})
	database.AutoMigrate(&types.VirtualMachine{})
	database.AutoMigrate(&types.Machine{})
	database.AutoMigrate(&types.AvailableResources{})
	database.AutoMigrate(&types.FreeResources{})
	database.AutoMigrate(&types.PeerInfo{})
	database.AutoMigrate(&types.Services{})
	database.AutoMigrate(&types.ServiceResourceRequirements{})
	database.AutoMigrate(&types.ContainerImages{})
	database.AutoMigrate(&types.RequestTracker{})
	database.AutoMigrate(&types.Libp2pInfo{})
	database.AutoMigrate(&types.DeploymentRequestFlat{})
	database.AutoMigrate(&types.MachineUUID{})
	database.AutoMigrate(&types.Connection{})

	DB = database
}
