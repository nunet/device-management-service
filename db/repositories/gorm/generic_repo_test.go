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

// Car is a struct for testing purposes
type Car struct {
	types.BaseDBModel
	Brand string
	Model string
	Price float64
}

// CarRepository is an interface for CRUD operations on Car entities
type CarRepository interface {
	repositories.GenericRepository[Car]
}

// CarGorm is a Gorm implementation of the CarRepository interface
type CarGorm struct {
	repositories.GenericRepository[Car]
}

// NewCarRepository creates a new instance of CarGorm
func NewCarRepository(db *gorm.DB) CarRepository {
	return &CarGorm{
		NewGenericRepository[Car](db),
	}
}

// TestGenericRepository is a test suite for the GenericRepository
func TestGenericRepository(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&Car{})
	assert.NoError(t, err)

	carRepo := NewCarRepository(db)

	// Test Create method
	createdCar, err := carRepo.Create(
		context.Background(),
		Car{
			Brand: "Toyota",
			Model: "Corolla",
			Price: 25000.00,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdCar.ID)

	// Test Get method
	retrievedCar, err := carRepo.Get(context.Background(), createdCar.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdCar.Brand, retrievedCar.Brand)
	assert.Equal(t, createdCar.Model, retrievedCar.Model)
	assert.Equal(t, createdCar.Price, retrievedCar.Price)

	// Test Update method
	updatedCar := retrievedCar
	updatedCar.Price = 26000.00

	_, err = carRepo.Update(context.Background(), updatedCar.ID, updatedCar)
	assert.NoError(t, err)

	retrievedCar, err = carRepo.Get(context.Background(), createdCar.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedCar.Price, retrievedCar.Price)

	// Test Delete method
	err = carRepo.Delete(context.Background(), updatedCar.ID)
	assert.NoError(t, err)

	// Try to get the deleted record
	_, err = carRepo.Get(context.Background(), updatedCar.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// Try to find the deleted record
	queryFindDeleted := carRepo.GetQuery()
	queryFindDeleted.Conditions = append(
		queryFindDeleted.Conditions,
		repositories.EQ("Brand", updatedCar.Brand),
	)
	_, err = carRepo.Find(context.Background(), queryFindDeleted)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// Test Find method
	car2, err := carRepo.Create(
		context.Background(),
		Car{
			Brand: "Honda",
			Model: "Civic",
			Price: 23000.00,
		},
	)
	assert.NoError(t, err)

	query := carRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.EQ("Brand", car2.Brand),
	)
	foundCar, err := carRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, car2.Brand, foundCar.Brand)

	// Test FindAll method
	car3, err := carRepo.Create(
		context.Background(),
		Car{
			Brand: "Ford",
			Model: "Mustang",
			Price: 35000.00,
		},
	)
	assert.NoError(t, err)

	allCars, err := carRepo.FindAll(context.Background(), carRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allCars, 2)

	// Clean up created records
	_ = carRepo.Delete(context.Background(), car2.ID)
	_ = carRepo.Delete(context.Background(), car3.ID)
}
