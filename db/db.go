package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/types"
)

func ConnectDatabase(dbPath string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", dbPath)), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	_ = database.AutoMigrate(&types.FreeResources{})
	_ = database.AutoMigrate(&types.RequestTracker{})
	_ = database.AutoMigrate(&types.OnboardedResources{})
	_ = database.AutoMigrate(&types.MachineResources{})
	_ = database.AutoMigrate(&types.OnboardingConfig{})
	_ = database.AutoMigrate(&types.ResourceAllocation{})
	_ = database.AutoMigrate(&types.GPU{})

	return database, nil
}
