package repositories_gorm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// TestPeerInfo is a test suite for the PeerInfo.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the PeerInfo model behave as expected.
func TestPeerInfo(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	peerInfoRepo := NewPeerInfo(db)

	// Test Create method
	createdPeerInfo, err := peerInfoRepo.Create(context.Background(), types.PeerInfo{})
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
		types.PeerInfo{Address: "127.0.0.1"},
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
		types.PeerInfo{Address: "127.0.0.2"},
	)
	assert.NoError(t, err)

	allPeerInfos, err := peerInfoRepo.FindAll(context.Background(), peerInfoRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allPeerInfos, 2)

	// Clean up created records
	err = peerInfoRepo.Delete(context.Background(), peerInfo1.ID)
	err = peerInfoRepo.Delete(context.Background(), peerInfo2.ID)
}

// TestMachine is a test suite for the Machine.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Machine model behave as expected.
func TestMachine(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	machineRepo := NewMachine(db)

	// Test Create method
	createdMachine, err := machineRepo.Create(context.Background(), types.Machine{})
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
	machine1, err := machineRepo.Create(context.Background(), types.Machine{IpAddr: "127.0.0.1"})
	assert.NoError(t, err)

	query := machineRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("IpAddr", machine1.IpAddr))
	foundMachine, err := machineRepo.Find(context.Background(), query)
	assert.NoError(t, err)
	assert.Equal(t, machine1.IpAddr, foundMachine.IpAddr)

	// Test FindAll method
	machine2, err := machineRepo.Create(context.Background(), types.Machine{IpAddr: "127.0.0.2"})
	assert.NoError(t, err)

	allMachines, err := machineRepo.FindAll(context.Background(), machineRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allMachines, 2)

	// Clean up created records
	err = machineRepo.Delete(context.Background(), machine1.ID)
	err = machineRepo.Delete(context.Background(), machine2.ID)
}

// TestServices is a test suite for the Services.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Services model behave as expected.
func TestServices(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	servicesRepo := NewServices(db)

	// Test Create method
	createdServices, err := servicesRepo.Create(context.Background(), types.Services{})
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
		types.Services{JobStatus: "finished without errors"},
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
		types.Services{JobStatus: "finished with errors"},
	)
	assert.NoError(t, err)

	allServices, err := servicesRepo.FindAll(context.Background(), servicesRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allServices, 2)

	// Clean up created records
	err = servicesRepo.Delete(context.Background(), service1.ID)
	err = servicesRepo.Delete(context.Background(), service2.ID)
}

// TestServiceResourceRequirements is a test suite for the ServiceResourceRequirements.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the ServiceResourceRequirements model behave as expected.
func TestServiceResourceRequirements(t *testing.T) {
	// Setup database serviceResourceRequirements for testing
	setup()
	defer teardown()

	// Initialize the repository
	serviceResourceRequirementsRepo := NewServiceResourceRequirements(
		db,
	)

	// Test Create method
	createdServiceResourceRequirements, err := serviceResourceRequirementsRepo.Create(
		context.Background(),
		types.ServiceResourceRequirements{},
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
		types.ServiceResourceRequirements{VCPU: 4},
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
		types.ServiceResourceRequirements{VCPU: 4},
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

// TestLibp2pInfo is a test suite for the Libp2pInfo.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Libp2pInfo model behave as expected.
func TestLibp2pInfo(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	libp2pInfoRepo := NewLibp2pInfo(db)

	// Test Save method
	createdLibp2pInfo, err := libp2pInfoRepo.Save(
		context.Background(),
		types.Libp2pInfo{ServerMode: false},
	)
	assert.NoError(t, err)
	assert.NotZero(t, createdLibp2pInfo.ID)

	// Test Get method
	retrievedLibp2pInfo, err := libp2pInfoRepo.Get(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, createdLibp2pInfo.ID, retrievedLibp2pInfo.ID)

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

// TestMachineUUID is a test suite for the MachineUUID.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the MachineUUID model behave as expected.
func TestMachineUUID(t *testing.T) {
	// Setup your database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	machineUUIDRepo := NewMachineUUID(db)

	// Test Save method
	createdMachineUUID, err := machineUUIDRepo.Save(
		context.Background(),
		types.MachineUUID{UUID: uuid.New().String()},
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

	machines, _ := machineUUIDRepo.History(context.Background(), machineUUIDRepo.GetQuery())
	assert.Len(t, machines, 2)

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

// TestConnection is a test suite for the Connection.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the Connection model behave as expected.
func TestConnection(t *testing.T) {
	// Setup database connection for testing
	setup()
	defer teardown()

	// Initialize the repository
	connectionRepo := NewConnection(db)

	// Test Create method
	createdConnection, err := connectionRepo.Create(context.Background(), types.Connection{})
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
		types.Connection{PeerID: uuid.New().String()},
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
		types.Connection{PeerID: uuid.New().String()},
	)
	assert.NoError(t, err)

	allConnections, err := connectionRepo.FindAll(context.Background(), connectionRepo.GetQuery())
	assert.NoError(t, err)
	assert.Len(t, allConnections, 2)

	// Clean up created records
	err = connectionRepo.Delete(context.Background(), connection1.ID)
	err = connectionRepo.Delete(context.Background(), connection2.ID)
}

// TestElasticToken is a test suite for the ElasticToken.
// It includes test cases that cover the basic CRUD operations and custom repository functions if there are any.
// This test suite ensures that the repository functions for the ElasticToken model behave as expected.
func TestElasticToken(t *testing.T) {
	// Setup database elasticToken for testing
	setup()
	defer teardown()

	// Initialize the repository
	elasticTokenRepo := NewElasticToken(db)

	// Test Create method
	createdElasticToken, err := elasticTokenRepo.Create(context.Background(), types.ElasticToken{})
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
		types.ElasticToken{Token: uuid.New().String()},
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
		types.ElasticToken{Token: uuid.New().String()},
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
