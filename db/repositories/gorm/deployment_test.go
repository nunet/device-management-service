package gorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// TestDeploymentRequestFlat is a test suite for the DeploymentRequestFlat.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the DeploymentRequestFlat model behave as expected.
func TestDeploymentRequestFlat(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	deploymentRequestFlatRepo := NewDeploymentRequestFlat(db)

	// Test Create method
	createdDeploymentRequestFlat, err := deploymentRequestFlatRepo.Create(
		context.Background(),
		types.DeploymentRequestFlat{},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdDeploymentRequestFlat.ID)

	// Test Get method
	retrievedDeploymentRequestFlat, err := deploymentRequestFlatRepo.Get(
		context.Background(),
		createdDeploymentRequestFlat.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdDeploymentRequestFlat.ID, retrievedDeploymentRequestFlat.ID)

	// Test Update method
	updatedDeploymentRequestFlat := retrievedDeploymentRequestFlat
	updatedDeploymentRequestFlat.JobStatus = "finished without errors"

	_, err = deploymentRequestFlatRepo.Update(
		context.Background(),
		updatedDeploymentRequestFlat.ID,
		updatedDeploymentRequestFlat,
	)
	assert.NoError(t, err)
	retrievedDeploymentRequestFlat, err = deploymentRequestFlatRepo.Get(
		context.Background(),
		createdDeploymentRequestFlat.ID,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		updatedDeploymentRequestFlat.JobStatus,
		retrievedDeploymentRequestFlat.JobStatus,
	)

	// Test Delete method
	err = deploymentRequestFlatRepo.Delete(context.Background(), updatedDeploymentRequestFlat.ID)
	assert.NoError(t, err)

	// Test Find method
	deploymentRequestFlat1, err := deploymentRequestFlatRepo.Create(
		context.Background(),
		types.DeploymentRequestFlat{JobStatus: "finished without errors"},
	)
	assert.NoError(t, err)

	query := deploymentRequestFlatRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.EQ("JobStatus", deploymentRequestFlat1.JobStatus),
	)
	foundDeploymentRequestFlat, err := deploymentRequestFlatRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, deploymentRequestFlat1.JobStatus, foundDeploymentRequestFlat.JobStatus)

	// Test FindAll method
	deploymentRequestFlat2, err := deploymentRequestFlatRepo.Create(
		context.Background(),
		types.DeploymentRequestFlat{JobStatus: "finished with errors"},
	)
	assert.NoError(t, err)

	allDeploymentRequestFlat, err := deploymentRequestFlatRepo.FindAll(
		context.Background(),
		deploymentRequestFlatRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allDeploymentRequestFlat, 2)

	// Clean up created records
	_ = deploymentRequestFlatRepo.Delete(context.Background(), deploymentRequestFlat1.ID)
	_ = deploymentRequestFlatRepo.Delete(context.Background(), deploymentRequestFlat2.ID)
}
