// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package clover

import (
	"context"
	"testing"

	clover "github.com/ostafen/clover/v2"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// ComputerSpecs is a struct for testing purposes
type ComputerSpecs struct {
	types.BaseDBModel
	CPU     int
	RAM     int
	Network NetworkSpecs
}

type NetworkSpecs struct {
	Interfaces map[string]string
}

// ComputerSpecsRepository is an interface for CRUD operations on ComputerSpecs entities
type ComputerSpecsRepository interface {
	repositories.GenericEntityRepository[ComputerSpecs]
}

// ComputerSpecsClover is a Clover implementation of the ComputerSpecsRepository interface
type ComputerSpecsClover struct {
	repositories.GenericEntityRepository[ComputerSpecs]
}

// NewComputerSpecsRepository creates a new instance of ComputerSpecsClover
func NewComputerSpecsRepository(db *clover.DB) ComputerSpecsRepository {
	return &ComputerSpecsClover{
		NewGenericEntityRepository[ComputerSpecs](db),
	}
}

// TestGenericEntityRepository is a test suite for the ComputerSpecsRepository
func TestGenericEntityRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)
	computerSpecsRepo := NewComputerSpecsRepository(db)

	// Test Save method
	specsToSave := ComputerSpecs{
		CPU: 4,
		RAM: 16,
		Network: NetworkSpecs{
			Interfaces: map[string]string{"eth0": "1Gbps", "wlan0": "802.11ac"},
		},
	}
	savedSpecs, err := computerSpecsRepo.Save(context.Background(), specsToSave)
	assert.NoError(t, err)

	// Test Get method
	retrievedSpecs, err := computerSpecsRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, savedSpecs.CPU, retrievedSpecs.CPU)
	assert.Equal(t, savedSpecs.RAM, retrievedSpecs.RAM)
	assert.Equal(t, savedSpecs.Network, retrievedSpecs.Network)

	// Test Update using Save method
	updatedSpecs := retrievedSpecs
	updatedSpecs.CPU = 8
	updatedSpecs.RAM = 32
	updatedSpecs.Network.Interfaces["eth0"] = "10Gbps"

	_, err = computerSpecsRepo.Save(context.Background(), updatedSpecs)
	assert.NoError(t, err)

	retrievedUpdatedSpecs, err := computerSpecsRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedSpecs.CPU, retrievedUpdatedSpecs.CPU)
	assert.Equal(t, updatedSpecs.RAM, retrievedUpdatedSpecs.RAM)
	assert.Equal(t, "10Gbps", retrievedUpdatedSpecs.Network.Interfaces["eth0"])

	// Test Clear method
	err = computerSpecsRepo.Clear(context.Background())
	assert.NoError(t, err)

	// Try to get the cleared document
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
