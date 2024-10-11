package gorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// Note: our GORM implementation does not support:
//
// - Structs with maps
//
// - Structs with nested NAMED structs, e.g.:
//     type ComputerSpecs struct {
//      types.BaseDBModel
//      CPU     int
//      Another AnotherStruct
//     }

// ComputerSpecs is a struct for testing purposes
type ComputerSpecs struct {
	types.BaseDBModel
	CPU int
	RAM int
}

// ComputerSpecsRepository is an interface for CRUD operations on ComputerSpecs entities
type ComputerSpecsRepository interface {
	repositories.GenericEntityRepository[ComputerSpecs]
}

// ComputerSpecsGorm is a Gorm implementation of the ComputerSpecsRepository interface
type ComputerSpecsGorm struct {
	repositories.GenericEntityRepository[ComputerSpecs]
}

// NewComputerSpecsRepository creates a new instance of ComputerSpecsGorm
func NewComputerSpecsRepository(db *gorm.DB) ComputerSpecsRepository {
	return &ComputerSpecsGorm{
		NewGenericEntityRepository[ComputerSpecs](db),
	}
}

// TestGenericEntityRepository is a test suite for the ComputerSpecsRepository
func TestGenericEntityRepository(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&ComputerSpecs{})
	assert.NoError(t, err)
	computerSpecsRepo := NewComputerSpecsRepository(db)

	// Test Save method
	specsToSave := ComputerSpecs{
		CPU: 4,
		RAM: 16,
	}
	savedSpecs, err := computerSpecsRepo.Save(context.Background(), specsToSave)
	assert.NoError(t, err)

	// Test Get method
	retrievedSpecs, err := computerSpecsRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, savedSpecs.CPU, retrievedSpecs.CPU)
	assert.Equal(t, savedSpecs.RAM, retrievedSpecs.RAM)

	// Test Update using Save method
	updatedSpecs := retrievedSpecs
	updatedSpecs.CPU = 8
	updatedSpecs.RAM = 32

	_, err = computerSpecsRepo.Save(context.Background(), updatedSpecs)
	assert.NoError(t, err)

	retrievedUpdatedSpecs, err := computerSpecsRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedSpecs.CPU, retrievedUpdatedSpecs.CPU)
	assert.Equal(t, updatedSpecs.RAM, retrievedUpdatedSpecs.RAM)

	// Test Clear method
	err = computerSpecsRepo.Clear(context.Background())
	assert.NoError(t, err)

	// Try to get the cleared record
	_, err = computerSpecsRepo.Get(context.Background())
	assert.Error(t, err)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// Test History method
	specs1 := ComputerSpecs{CPU: 4, RAM: 16}
	_, err = computerSpecsRepo.Save(context.Background(), specs1)
	assert.NoError(t, err)

	specs2 := ComputerSpecs{CPU: 8, RAM: 32}
	_, err = computerSpecsRepo.Save(context.Background(), specs2)
	assert.NoError(t, err)

	query := computerSpecsRepo.GetQuery()
	query.SortBy = "-CreatedAt" // Sort by CreatedAt in descending order
	history, err := computerSpecsRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, specs2.CPU, history[0].CPU)
	assert.Equal(t, specs1.CPU, history[1].CPU)

	// Clean up
	_ = computerSpecsRepo.Clear(context.Background())
}
