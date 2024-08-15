package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfoGORM is a GORM implementation of the PeerInfo interface.
type PeerInfoGORM struct {
	repositories.GenericRepository[types.PeerInfo]
}

// NewPeerInfo creates a new instance of PeerInfoGORM.
// It initializes and returns a GORM-based repository for PeerInfo entities.
func NewPeerInfo(db *gorm.DB) repositories.PeerInfo {
	return &PeerInfoGORM{NewGenericRepository[types.PeerInfo](db)}
}

// MachineGORM is a GORM implementation of the Machine interface.
type MachineGORM struct {
	repositories.GenericRepository[types.Machine]
}

// NewMachine creates a new instance of MachineGORM.
// It initializes and returns a GORM-based repository for Machine entities.
func NewMachine(db *gorm.DB) repositories.Machine {
	return &MachineGORM{NewGenericRepository[types.Machine](db)}
}

// FreeResourcesGORM is a GORM implementation of the FreeResources interface.
type FreeResourcesGORM struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResources creates a new instance of FreeResourcesGORM.
// It initializes and returns a GORM-based repository for FreeResources entity.
func NewFreeResources(db *gorm.DB) repositories.FreeResources {
	return &FreeResourcesGORM{NewGenericEntityRepository[types.FreeResources](db)}
}

// AvailableResourcesGORM is a GORM implementation of the AvailableResources interface.
type AvailableResourcesGORM struct {
	repositories.GenericEntityRepository[types.AvailableResources]
}

// NewAvailableResources creates a new instance of AvailableResourcesGORM.
// It initializes and returns a GORM-based repository for AvailableResources entity.
func NewAvailableResources(db *gorm.DB) repositories.AvailableResources {
	return &AvailableResourcesGORM{
		NewGenericEntityRepository[types.AvailableResources](db),
	}
}

// ServicesGORM is a GORM implementation of the Services interface.
type ServicesGORM struct {
	repositories.GenericRepository[types.Services]
}

// NewServices creates a new instance of ServicesGORM.
// It initializes and returns a GORM-based repository for Services entities.
func NewServices(db *gorm.DB) repositories.Services {
	return &ServicesGORM{NewGenericRepository[types.Services](db)}
}

// ServiceResourceRequirementsGORM is a GORM implementation of the ServiceResourceRequirements interface.
type ServiceResourceRequirementsGORM struct {
	repositories.GenericRepository[types.ServiceResourceRequirements]
}

// NewServiceResourceRequirements creates a new instance of ServiceResourceRequirementsGORM.
// It initializes and returns a GORM-based repository for ServiceResourceRequirements entities.
func NewServiceResourceRequirements(
	db *gorm.DB,
) repositories.ServiceResourceRequirements {
	return &ServiceResourceRequirementsGORM{
		NewGenericRepository[types.ServiceResourceRequirements](db),
	}
}

// Libp2pInfoGORM is a GORM implementation of the Libp2pInfo interface.
type Libp2pInfoGORM struct {
	repositories.GenericEntityRepository[types.Libp2pInfo]
}

// NewLibp2pInfo creates a new instance of Libp2pInfoGORM.
// It initializes and returns a GORM-based repository for Libp2pInfo entity.
func NewLibp2pInfo(db *gorm.DB) repositories.Libp2pInfo {
	return &Libp2pInfoGORM{NewGenericEntityRepository[types.Libp2pInfo](db)}
}

// MachineUUIDGORM is a GORM implementation of the MachineUUID interface.
type MachineUUIDGORM struct {
	repositories.GenericEntityRepository[types.MachineUUID]
}

// NewMachineUUID creates a new instance of MachineUUIDGORM.
// It initializes and returns a GORM-based repository for MachineUUID entity.
func NewMachineUUID(db *gorm.DB) repositories.MachineUUID {
	return &MachineUUIDGORM{NewGenericEntityRepository[types.MachineUUID](db)}
}

// ConnectionGORM is a GORM implementation of the Connection interface.
type ConnectionGORM struct {
	repositories.GenericRepository[types.Connection]
}

// NewConnection creates a new instance of ConnectionGORM.
// It initializes and returns a GORM-based repository for Connection entities.
func NewConnection(db *gorm.DB) repositories.Connection {
	return &ConnectionGORM{NewGenericRepository[types.Connection](db)}
}

// ElasticTokenGORM is a GORM implementation of the ElasticToken interface.
type ElasticTokenGORM struct {
	repositories.GenericRepository[types.ElasticToken]
}

// NewElasticToken creates a new instance of ElasticTokenGORM.
// It initializes and returns a GORM-based repository for ElasticToken entities.
func NewElasticToken(db *gorm.DB) repositories.ElasticToken {
	return &ElasticTokenGORM{NewGenericRepository[types.ElasticToken](db)}
}
