package repositories_clover

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/models"
)

// TestPeerInfoRepository is a test suite for the PeerInfoRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the PeerInfo model behave as expected.
func TestPeerInfoRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	peerInfoRepo := NewPeerInfoRepository(db)

	// Test Create method
	createdPeerInfo, err := peerInfoRepo.Create(context.Background(), models.PeerInfo{})
	assert.NoError(t, err)
	assert.NotZero(t, createdPeerInfo.ID)

	// Test Get method
	retrievedPeerInfo, err := peerInfoRepo.Get(context.Background(), createdPeerInfo.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdPeerInfo.ID, retrievedPeerInfo.ID)

	// Test Update method
	updatedPeerInfo := retrievedPeerInfo
	updatedPeerInfo.Address = "127.0.0.1"

	_, err = peerInfoRepo.Update(
		context.Background(),
		updatedPeerInfo.ID,
		updatedPeerInfo,
	)
	assert.NoError(t, err)
	retrievedPeerInfo, err = peerInfoRepo.Get(context.Background(), createdPeerInfo.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedPeerInfo.Address, retrievedPeerInfo.Address)

	// Test Delete method
	err = peerInfoRepo.Delete(context.Background(), updatedPeerInfo.ID)
	assert.NoError(t, err)

	// Test Find method
	peerInfo1, err := peerInfoRepo.Create(
		context.Background(),
		models.PeerInfo{Address: "127.0.0.1"},
	)
	assert.NoError(t, err)

	query := peerInfoRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Address", peerInfo1.Address))
	foundPeerInfo, err := peerInfoRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, peerInfo1.Address, foundPeerInfo.Address)

	// Test FindAll method
	peerInfo2, err := peerInfoRepo.Create(
		context.Background(),
		models.PeerInfo{Address: "127.0.0.2"},
	)
	assert.NoError(t, err)

	allPeerInfos, err := peerInfoRepo.FindAll(context.Background(), peerInfoRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allPeerInfos, 2)

	// Clean up created records
	err = peerInfoRepo.Delete(context.Background(), peerInfo1.ID)
	err = peerInfoRepo.Delete(context.Background(), peerInfo2.ID)
}

// TestMachineRepository is a test suite for the MachineRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Machine model behave as expected.
func TestMachineRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	machineRepo := NewMachineRepository(db)

	// Test Create method
	createdMachine, err := machineRepo.Create(context.Background(), models.Machine{})
	assert.NoError(t, err)
	assert.NotZero(t, createdMachine.ID)

	// Test Get method
	retrievedMachine, err := machineRepo.Get(context.Background(), createdMachine.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdMachine.ID, retrievedMachine.ID)

	// Test Update method
	updatedMachine := retrievedMachine
	updatedMachine.IpAddr = "127.0.0.1"

	_, err = machineRepo.Update(
		context.Background(),
		updatedMachine.ID,
		updatedMachine,
	)
	assert.NoError(t, err)
	retrievedMachine, err = machineRepo.Get(context.Background(), createdMachine.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedMachine.IpAddr, retrievedMachine.IpAddr)

	// Test Delete method
	err = machineRepo.Delete(context.Background(), updatedMachine.ID)
	assert.NoError(t, err)

	// Test Find method
	machine1, err := machineRepo.Create(context.Background(), models.Machine{IpAddr: "127.0.0.1"})
	assert.NoError(t, err)

	query := machineRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("IpAddr", machine1.IpAddr))
	foundMachine, err := machineRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, machine1.IpAddr, foundMachine.IpAddr)

	// Test FindAll method
	machine2, err := machineRepo.Create(context.Background(), models.Machine{IpAddr: "127.0.0.2"})
	assert.NoError(t, err)

	allMachines, err := machineRepo.FindAll(context.Background(), machineRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allMachines, 2)

	// Clean up created records
	err = machineRepo.Delete(context.Background(), machine1.ID)
	err = machineRepo.Delete(context.Background(), machine2.ID)
}

// TestFreeResourcesRepository is a test suite for the FreeResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the FreeResources model behave as expected.
func TestFreeResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	freeResourcesRepo := NewFreeResourcesRepository(db)

	// Test Save method
	createdFreeResources, err := freeResourcesRepo.Save(
		context.Background(),
		models.FreeResources{
			Vcpu: 2,
		},
	)
	assert.NoError(t, err)

	// Test Get method
	retrievedFreeResources, err := freeResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdFreeResources.Vcpu, retrievedFreeResources.Vcpu)

	// Test Save (update) method
	updatedFreeResources := retrievedFreeResources
	updatedFreeResources.Vcpu = 4

	_, err = freeResourcesRepo.Save(context.Background(), updatedFreeResources)
	assert.NoError(t, err)
	retrievedFreeResources, err = freeResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedFreeResources.Vcpu, retrievedFreeResources.Vcpu)

	// Test History method
	query := freeResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := freeResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = freeResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = freeResourcesRepo.History(context.Background(), query)
	assert.Len(t, history, 0)
}

// TestAvailableResourcesRepository is a test suite for the AvailableResourcesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the AvailableResources model behave as expected.
func TestAvailableResourcesRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	availableResourcesRepo := NewAvailableResourcesRepository(db)

	// Test Save method
	createdAvailableResources, err := availableResourcesRepo.Save(
		context.Background(),
		models.AvailableResources{
			Vcpu: 2,
		},
	)
	assert.NoError(t, err)

	// Test Get method
	retrievedAvailableResources, err := availableResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdAvailableResources.Vcpu, retrievedAvailableResources.Vcpu)

	// Test Save (update) method
	updatedAvailableResources := retrievedAvailableResources
	updatedAvailableResources.Vcpu = 4

	_, err = availableResourcesRepo.Save(context.Background(), updatedAvailableResources)
	assert.NoError(t, err)
	retrievedAvailableResources, err = availableResourcesRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedAvailableResources.Vcpu, retrievedAvailableResources.Vcpu)

	// Test History method
	query := availableResourcesRepo.GetQuery()
	query.Limit = 3
	history, err := availableResourcesRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = availableResourcesRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = availableResourcesRepo.History(context.Background(), query)
	assert.Len(t, history, 0)
}

// TestServicesRepository is a test suite for the ServicesRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Services model behave as expected.
func TestServicesRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	servicesRepo := NewServicesRepository(db)

	// Test Create method
	createdServices, err := servicesRepo.Create(context.Background(), models.Services{})
	assert.NoError(t, err)
	assert.NotZero(t, createdServices.ID)

	// Test Get method
	retrievedServices, err := servicesRepo.Get(context.Background(), createdServices.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdServices.ID, retrievedServices.ID)

	// Test Update method
	updatedServices := retrievedServices
	updatedServices.JobStatus = "finished without errors"

	_, err = servicesRepo.Update(
		context.Background(),
		updatedServices.ID,
		updatedServices,
	)
	assert.NoError(t, err)
	retrievedServices, err = servicesRepo.Get(context.Background(), createdServices.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedServices.JobStatus, retrievedServices.JobStatus)

	// Test Delete method
	err = servicesRepo.Delete(context.Background(), updatedServices.ID)
	assert.NoError(t, err)

	// Test Find method
	service1, err := servicesRepo.Create(
		context.Background(),
		models.Services{JobStatus: "finished without errors"},
	)
	assert.NoError(t, err)

	query := servicesRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("JobStatus", service1.JobStatus))
	foundService, err := servicesRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, service1.JobStatus, foundService.JobStatus)

	// Test FindAll method
	service2, err := servicesRepo.Create(
		context.Background(),
		models.Services{JobStatus: "finished with errors"},
	)
	assert.NoError(t, err)

	allServices, err := servicesRepo.FindAll(context.Background(), servicesRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allServices, 2)

	// Clean up created records
	err = servicesRepo.Delete(context.Background(), service1.ID)
	err = servicesRepo.Delete(context.Background(), service2.ID)
}

// TestServiceResourceRequirementsRepository is a test suite for the ServiceResourceRequirementsRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the ServiceResourceRequirements model behave as expected.
func TestServiceResourceRequirementsRepository(t *testing.T) {
	// Setup database serviceResourceRequirements for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	serviceResourceRequirementsRepo := NewServiceResourceRequirementsRepository(
		db,
	)

	// Test Create method
	createdServiceResourceRequirements, err := serviceResourceRequirementsRepo.Create(
		context.Background(),
		models.ServiceResourceRequirements{},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdServiceResourceRequirements.ID)

	// Test Get method
	retrievedServiceResourceRequirements, err := serviceResourceRequirementsRepo.Get(
		context.Background(),
		createdServiceResourceRequirements.ID,
	)
	assert.NoError(t, err)
	assert.Equal(t, createdServiceResourceRequirements.ID, retrievedServiceResourceRequirements.ID)

	// Test Update method
	updatedServiceResourceRequirements := retrievedServiceResourceRequirements
	updatedServiceResourceRequirements.VCPU = 4

	_, err = serviceResourceRequirementsRepo.Update(
		context.Background(),
		updatedServiceResourceRequirements.ID,
		updatedServiceResourceRequirements,
	)
	assert.NoError(t, err)
	retrievedServiceResourceRequirements, err = serviceResourceRequirementsRepo.Get(
		context.Background(),
		createdServiceResourceRequirements.ID,
	)
	assert.NoError(t, err)
	assert.Equal(
		t,
		updatedServiceResourceRequirements.VCPU,
		retrievedServiceResourceRequirements.VCPU,
	)

	// Test Delete method
	err = serviceResourceRequirementsRepo.Delete(
		context.Background(),
		updatedServiceResourceRequirements.ID,
	)
	assert.NoError(t, err)

	// Test Find method
	serviceResourceRequirements1, err := serviceResourceRequirementsRepo.Create(
		context.Background(),
		models.ServiceResourceRequirements{VCPU: 4},
	)
	assert.NoError(t, err)

	query := serviceResourceRequirementsRepo.GetQuery()
	query.Conditions = append(
		query.Conditions,
		repositories.EQ("VCPU", serviceResourceRequirements1.VCPU),
	)
	foundServiceResourceRequirements, err := serviceResourceRequirementsRepo.Find(
		context.Background(),
		query,
	)
	assert.NoError(t, err)
	assert.Equal(t, serviceResourceRequirements1.VCPU, foundServiceResourceRequirements.VCPU)

	// Test FindAll method
	serviceResourceRequirements2, err := serviceResourceRequirementsRepo.Create(
		context.Background(),
		models.ServiceResourceRequirements{VCPU: 4},
	)
	assert.NoError(t, err)

	allServiceResourceRequirements, err := serviceResourceRequirementsRepo.FindAll(
		context.Background(),
		serviceResourceRequirementsRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allServiceResourceRequirements, 2)

	// Clean up created records
	err = serviceResourceRequirementsRepo.Delete(
		context.Background(),
		serviceResourceRequirements1.ID,
	)
	err = serviceResourceRequirementsRepo.Delete(
		context.Background(),
		serviceResourceRequirements2.ID,
	)
}

// TestLibp2pInfoRepository is a test suite for the Libp2pInfoRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Libp2pInfo model behave as expected.
func TestLibp2pInfoRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	libp2pInfoRepo := NewLibp2pInfoRepository(db)

	// Test Save method
	_, err := libp2pInfoRepo.Save(
		context.Background(),
		models.Libp2pInfo{ServerMode: false},
	)
	assert.NoError(t, err)

	// Test Get method
	retrievedLibp2pInfo, err := libp2pInfoRepo.Get(context.Background())
	assert.NoError(t, err)

	// Test Save (update) method
	updatedLibp2pInfo := retrievedLibp2pInfo
	updatedLibp2pInfo.ServerMode = true

	_, err = libp2pInfoRepo.Save(context.Background(), updatedLibp2pInfo)
	assert.NoError(t, err)
	retrievedLibp2pInfo, err = libp2pInfoRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedLibp2pInfo.ServerMode, retrievedLibp2pInfo.ServerMode)

	// Test History method
	query := libp2pInfoRepo.GetQuery()
	query.Limit = 3
	history, err := libp2pInfoRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = libp2pInfoRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = libp2pInfoRepo.History(context.Background(), query)
	assert.Len(t, history, 0)
}

// TestMachineUUIDRepository is a test suite for the MachineUUIDRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the MachineUUID model behave as expected.
func TestMachineUUIDRepository(t *testing.T) {
	// Setup your database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	machineUUIDRepo := NewMachineUUIDRepository(db)

	// Test Save method
	createdMachineUUID, err := machineUUIDRepo.Save(
		context.Background(),
		models.MachineUUID{UUID: uuid.New().String()},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdMachineUUID.UUID)

	// Test Get method
	retrievedMachineUUID, err := machineUUIDRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdMachineUUID.UUID, retrievedMachineUUID.UUID)

	// Test Save (update) method
	updatedMachineUUID := retrievedMachineUUID
	updatedMachineUUID.UUID = uuid.New().String()

	_, err = machineUUIDRepo.Save(context.Background(), updatedMachineUUID)
	assert.NoError(t, err)
	retrievedMachineUUID, err = machineUUIDRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, updatedMachineUUID.UUID, retrievedMachineUUID.UUID)

	// Test History method
	query := machineUUIDRepo.GetQuery()
	query.Limit = 3
	history, err := machineUUIDRepo.History(context.Background(), query)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// Test Clear method
	err = machineUUIDRepo.Clear(context.Background())
	assert.NoError(t, err)
	history, err = machineUUIDRepo.History(context.Background(), query)
	assert.Len(t, history, 0)
}

// TestConnectionRepository is a test suite for the ConnectionRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Connection model behave as expected.
func TestConnectionRepository(t *testing.T) {
	// Setup database connection for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	connectionRepo := NewConnectionRepository(db)

	// Test Create method
	createdConnection, err := connectionRepo.Create(context.Background(), models.Connection{})
	assert.NoError(t, err)
	assert.NotZero(t, createdConnection.ID)

	// Test Get method
	retrievedConnection, err := connectionRepo.Get(context.Background(), createdConnection.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdConnection.ID, retrievedConnection.ID)

	// Test Update method
	updatedConnection := retrievedConnection
	updatedConnection.PeerID = uuid.New().String()

	_, err = connectionRepo.Update(
		context.Background(),
		updatedConnection.ID,
		updatedConnection,
	)
	assert.NoError(t, err)
	retrievedConnection, err = connectionRepo.Get(context.Background(), createdConnection.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedConnection.PeerID, retrievedConnection.PeerID)

	// Test Delete method
	err = connectionRepo.Delete(context.Background(), updatedConnection.ID)
	assert.NoError(t, err)

	// Test Find method
	connection1, err := connectionRepo.Create(
		context.Background(),
		models.Connection{PeerID: uuid.New().String()},
	)
	assert.NoError(t, err)

	query := connectionRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("PeerID", connection1.PeerID))
	foundConnection, err := connectionRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, connection1.PeerID, foundConnection.PeerID)

	// Test FindAll method
	connection2, err := connectionRepo.Create(
		context.Background(),
		models.Connection{PeerID: uuid.New().String()},
	)
	assert.NoError(t, err)

	allConnections, err := connectionRepo.FindAll(context.Background(), connectionRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allConnections, 2)

	// Clean up created records
	err = connectionRepo.Delete(context.Background(), connection1.ID)
	err = connectionRepo.Delete(context.Background(), connection2.ID)
}

// TestElasticTokenRepository is a test suite for the ElasticTokenRepository.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the ElasticToken model behave as expected.
func TestElasticTokenRepository(t *testing.T) {
	// Setup database elasticToken for testing
	db, path := setup()
	defer teardown(db, path)

	// Initialize the repository
	elasticTokenRepo := NewElasticTokenRepository(db)

	// Test Create method
	createdElasticToken, err := elasticTokenRepo.Create(context.Background(), models.ElasticToken{})
	assert.NoError(t, err)
	assert.NotZero(t, createdElasticToken.ID)

	// Test Get method
	retrievedElasticToken, err := elasticTokenRepo.Get(context.Background(), createdElasticToken.ID)
	assert.NoError(t, err)
	assert.Equal(t, createdElasticToken.ID, retrievedElasticToken.ID)

	// Test Update method
	updatedElasticToken := retrievedElasticToken
	updatedElasticToken.Token = uuid.New().String()

	_, err = elasticTokenRepo.Update(
		context.Background(),
		updatedElasticToken.ID,
		updatedElasticToken,
	)
	assert.NoError(t, err)
	retrievedElasticToken, err = elasticTokenRepo.Get(context.Background(), createdElasticToken.ID)
	assert.NoError(t, err)
	assert.Equal(t, updatedElasticToken.Token, retrievedElasticToken.Token)

	// Test Delete method
	err = elasticTokenRepo.Delete(context.Background(), updatedElasticToken.ID)
	assert.NoError(t, err)

	// Test Find method
	elasticToken1, err := elasticTokenRepo.Create(
		context.Background(),
		models.ElasticToken{Token: uuid.New().String()},
	)
	assert.NoError(t, err)

	query := elasticTokenRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Token", elasticToken1.Token))
	foundElasticToken, err := elasticTokenRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, elasticToken1.Token, foundElasticToken.Token)

	// Test FindAll method
	elasticToken2, err := elasticTokenRepo.Create(
		context.Background(),
		models.ElasticToken{Token: uuid.New().String()},
	)
	assert.NoError(t, err)

	allElasticTokens, err := elasticTokenRepo.FindAll(
		context.Background(),
		elasticTokenRepo.GetQuery(),
	)
	assert.NoError(t, err)
	assert.Len(t, allElasticTokens, 2)

	// Clean up created records
	err = elasticTokenRepo.Delete(context.Background(), elasticToken1.ID)
	err = elasticTokenRepo.Delete(context.Background(), elasticToken2.ID)
}
