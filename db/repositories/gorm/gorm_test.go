package repositories_gorm

import (
	"gitlab.com/nunet/device-management-service/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// setup initializes and sets up the in-memory SQLite database connection for testing purposes.
// Additionally, it automatically migrates the necessary models to ensure the schema is up-to-date.
func setup() {
	// Set up the database connection for tests
	var err error
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	// Run Migrations if needed
	db.AutoMigrate(
		&models.PeerInfo{},
		&models.Machine{},
		&models.FreeResources{},
		&models.AvailableResources{},
		&models.Services{},
		&models.ServiceResourceRequirements{},
		&models.Libp2pInfo{},
		&models.MachineUUID{},
		&models.Connection{},
		&models.ElasticToken{},
		&models.DeploymentRequestFlat{},
		&models.RequestTracker{},
		&models.VirtualMachine{},
	)
}

// teardown resets the GORM database connection after tests.
// In the context of an in-memory SQLite database, it creates a new instance of the GORM DB,
// effectively closing the current connection. This is often sufficient cleanup for testing
// with in-memory databases as they don't persist data between tests.
func teardown() {
	// Reset the GORM database connection after tests
	db = db.Session(&gorm.Session{NewDB: true})
}
