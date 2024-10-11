package clover

import (
	"context"
	"testing"

	clover "github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// Car is a struct for testing purposes
type Car struct {
	types.BaseDBModel
	Brand    string
	Model    string
	Price    float64
	Engine   Engine
	Features map[string]int
}

type Engine struct {
	Cylinders int
	Power     float64
}

// CarRepository is an interface for CRUD operations on Car entities
type CarRepository interface {
	repositories.GenericRepository[Car]
}

// CarClover is a Clover implementation of the CarRepository interface
type CarClover struct {
	repositories.GenericRepository[Car]
}

// NewCarRepository creates a new instance of CarClover
func NewCarRepository(db *clover.DB) CarRepository {
	return &CarClover{
		NewGenericRepository[Car](db),
	}
}

// TestGenericRepository is a test suite for the GenericRepository
func TestGenericRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)
	carRepo := NewCarRepository(db)

	// Test Create method
	createdCar, err := carRepo.Create(
		context.Background(),
		Car{
			Brand: "Toyota",
			Model: "Corolla",
			Price: 25000.00,
			Engine: Engine{
				Cylinders: 4,
				Power:     132.5,
			},
			Features: map[string]int{
				"Airbags": 6,
				"Doors":   4,
			},
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdCar.ID)

	// Test Get method
	retrievedCar, err := carRepo.Get(context.Background(), createdCar.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdCar.Brand, retrievedCar.Brand)
	assert.Equal(t, createdCar.Engine.Cylinders, retrievedCar.Engine.Cylinders)
	assert.Equal(t, createdCar.Engine.Power, retrievedCar.Engine.Power)
	assert.Equal(t, createdCar.Features, retrievedCar.Features)

	// Test Update method
	updatedCar := retrievedCar
	updatedCar.Price = 26000.00
	updatedCar.Features["Airbags"] = 8

	_, err = carRepo.Update(context.Background(), updatedCar.ID, updatedCar)
	assert.NoError(t, err)

	retrievedCar, err = carRepo.Get(context.Background(), createdCar.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedCar.Price, retrievedCar.Price)
	assert.Equal(t, 8, retrievedCar.Features["Airbags"])

	// Test Delete method
	err = carRepo.Delete(context.Background(), updatedCar.ID)
	assert.NoError(t, err)

	// Try to get the deleted document
	_, err = carRepo.Get(context.Background(), updatedCar.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repositories.ErrNotFound)

	// Try to find the deleted document
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

func TestGenericRepoQueries(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)
	carRepo := NewCarRepository(db)

	cars := []Car{
		{Brand: "Toyota", Model: "Corolla", Price: 25000.00, Engine: Engine{Cylinders: 4, Power: 132.5}, Features: map[string]int{"Airbags": 6, "Doors": 4}},
		{Brand: "Honda", Model: "Civic", Price: 23000.00, Engine: Engine{Cylinders: 4, Power: 158.0}, Features: map[string]int{"Airbags": 8, "Doors": 4}},
		{Brand: "Ford", Model: "Mustang", Price: 35000.00, Engine: Engine{Cylinders: 8, Power: 460.0}, Features: map[string]int{"Airbags": 6, "Doors": 2}},
		{Brand: "Tesla", Model: "Model 3", Price: 45000.00, Engine: Engine{Cylinders: 0, Power: 283.0}, Features: map[string]int{"Airbags": 8, "Doors": 4}},
		{Brand: "BMW", Model: "X5", Price: 60000.00, Engine: Engine{Cylinders: 6, Power: 335.0}, Features: map[string]int{"Airbags": 10, "Doors": 5}},
	}

	for _, car := range cars {
		_, err := carRepo.Create(context.Background(), car)
		assert.NoError(t, err)
	}

	// Test EQ operator
	query := carRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Brand", "Toyota"))
	result, err := carRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, cars[0].Model, result.Model)

	// Test GT operator
	query = carRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.GT("Price", 50000.00))
	results, err := carRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "BMW", results[0].Brand)

	// Test LTE operator
	query = carRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.LTE("Price", 25000.00))
	results, err = carRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))

	// Test IN operator
	query = carRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.IN("Brand", []interface{}{"Honda", "Ford", "Tesla"}))
	results, err = carRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))

	// Test LIKE operator: TODO

	// Test sorting
	query = carRepo.GetQuery()
	query.SortBy = "-Price" // Sort by Price in descending order
	results, err = carRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(results))
	assert.Equal(t, "BMW", results[0].Brand)
	assert.Equal(t, "Honda", results[4].Brand)

	// Test limit
	query = carRepo.GetQuery()
	query.SortBy = "Price" // Sort by Price in ascending order
	query.Limit = 2
	results, err = carRepo.FindAll(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "Honda", results[0].Brand)
	assert.Equal(t, "Toyota", results[1].Brand)

	// Clean up created records
	for _, car := range cars {
		err := carRepo.Delete(context.Background(), car.ID)
		assert.NoError(t, err)
	}
}
