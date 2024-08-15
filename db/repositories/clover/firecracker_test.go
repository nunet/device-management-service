package repositories_clover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// TestVirtualMachine is a test suite for the VirtualMachine.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the VirtualMachine model behave as expected.
func TestVirtualMachine(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	virtualMachineRepo := NewVirtualMachine(db)

	// Test Create method
	createdVirtualMachine, err := virtualMachineRepo.Create(
		context.Background(),
		types.VirtualMachine{},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdVirtualMachine.ID)

	// Test Get method
	retrievedVirtualMachine, err := virtualMachineRepo.Get(
		context.Background(),
		createdVirtualMachine.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdVirtualMachine.ID, retrievedVirtualMachine.ID)

	// Test Update method
	updatedVirtualMachine := retrievedVirtualMachine
	updatedVirtualMachine.State = "awaiting"

	_, err = virtualMachineRepo.Update(
		context.Background(),
		updatedVirtualMachine.ID,
		updatedVirtualMachine,
	)
	assert.NoError(t, err)
	retrievedVirtualMachine, err = virtualMachineRepo.Get(
		context.Background(),
		createdVirtualMachine.ID,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		updatedVirtualMachine.State,
		retrievedVirtualMachine.State,
	)

	// Test Delete method
	err = virtualMachineRepo.Delete(context.Background(), updatedVirtualMachine.ID)
	assert.NoError(t, err)

	// Test Find method
	virtualMachine1, err := virtualMachineRepo.Create(
		context.Background(),
		types.VirtualMachine{State: "running"},
	)
	assert.NoError(t, err)

	query := virtualMachineRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("State", virtualMachine1.State))
	foundVirtualMachine, err := virtualMachineRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, virtualMachine1.State, foundVirtualMachine.State)

	// Test FindAll method
	virtualMachine2, err := virtualMachineRepo.Create(
		context.Background(),
		types.VirtualMachine{State: "stopped"},
	)
	assert.NoError(t, err)

	allVirtualMachine, err := virtualMachineRepo.FindAll(
		context.Background(),
		virtualMachineRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allVirtualMachine, 2)

	// Clean up created records
	err = virtualMachineRepo.Delete(context.Background(), virtualMachine1.ID)
	err = virtualMachineRepo.Delete(context.Background(), virtualMachine2.ID)
}
