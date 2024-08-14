package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfoRepositoryGORM is a GORM implementation of the PeerInfoRepository interface.
type PeerInfoRepositoryGORM struct {
	repositories.GenericRepository[types.PeerInfo]
}

// NewPeerInfoRepository creates a new instance of PeerInfoRepositoryGORM.
// It initializes and returns a GORM-based repository for PeerInfo entities.
func NewPeerInfoRepository(db *gorm.DB) repositories.PeerInfoRepository {
	return &PeerInfoRepositoryGORM{NewGenericRepository[types.PeerInfo](db)}
}

// MachineRepositoryGORM is a GORM implementation of the MachineRepository interface.
type MachineRepositoryGORM struct {
	repositories.GenericRepository[types.Machine]
}

// NewMachineRepository creates a new instance of MachineRepositoryGORM.
// It initializes and returns a GORM-based repository for Machine entities.
func NewMachineRepository(db *gorm.DB) repositories.MachineRepository {
	return &MachineRepositoryGORM{NewGenericRepository[types.Machine](db)}
}

// FreeResourcesRepositoryGORM is a GORM implementation of the FreeResourcesRepository interface.
type FreeResourcesRepositoryGORM struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResourcesRepository creates a new instance of FreeResourcesRepositoryGORM.
// It initializes and returns a GORM-based repository for FreeResources entity.
func NewFreeResourcesRepository(db *gorm.DB) repositories.FreeResourcesRepository {
	return &FreeResourcesRepositoryGORM{NewGenericEntityRepository[types.FreeResources](db)}
}

// AvailableResourcesRepositoryGORM is a GORM implementation of the AvailableResourcesRepository interface.
type AvailableResourcesRepositoryGORM struct {
	repositories.GenericEntityRepository[types.AvailableResources]
}

// NewAvailableResourcesRepository creates a new instance of AvailableResourcesRepositoryGORM.
// It initializes and returns a GORM-based repository for AvailableResources entity.
func NewAvailableResourcesRepository(db *gorm.DB) repositories.AvailableResourcesRepository {
	return &AvailableResourcesRepositoryGORM{
		NewGenericEntityRepository[types.AvailableResources](db),
	}
}

// ServicesRepositoryGORM is a GORM implementation of the ServicesRepository interface.
type ServicesRepositoryGORM struct {
	repositories.GenericRepository[types.Services]
}

// NewServicesRepository creates a new instance of ServicesRepositoryGORM.
// It initializes and returns a GORM-based repository for Services entities.
func NewServicesRepository(db *gorm.DB) repositories.ServicesRepository {
	return &ServicesRepositoryGORM{NewGenericRepository[types.Services](db)}
}

// ServiceResourceRequirementsRepositoryGORM is a GORM implementation of the ServiceResourceRequirementsRepository interface.
type ServiceResourceRequirementsRepositoryGORM struct {
	repositories.GenericRepository[types.ServiceResourceRequirements]
}

// NewServiceResourceRequirementsRepository creates a new instance of ServiceResourceRequirementsRepositoryGORM.
// It initializes and returns a GORM-based repository for ServiceResourceRequirements entities.
func NewServiceResourceRequirementsRepository(
	db *gorm.DB,
) repositories.ServiceResourceRequirementsRepository {
	return &ServiceResourceRequirementsRepositoryGORM{
		NewGenericRepository[types.ServiceResourceRequirements](db),
	}
}

// Libp2pInfoRepositoryGORM is a GORM implementation of the Libp2pInfoRepository interface.
type Libp2pInfoRepositoryGORM struct {
	repositories.GenericEntityRepository[types.Libp2pInfo]
}

// NewLibp2pInfoRepository creates a new instance of Libp2pInfoRepositoryGORM.
// It initializes and returns a GORM-based repository for Libp2pInfo entity.
func NewLibp2pInfoRepository(db *gorm.DB) repositories.Libp2pInfoRepository {
	return &Libp2pInfoRepositoryGORM{NewGenericEntityRepository[types.Libp2pInfo](db)}
}

// MachineUUIDRepositoryGORM is a GORM implementation of the MachineUUIDRepository interface.
type MachineUUIDRepositoryGORM struct {
	repositories.GenericEntityRepository[types.MachineUUID]
}

// NewMachineUUIDRepository creates a new instance of MachineUUIDRepositoryGORM.
// It initializes and returns a GORM-based repository for MachineUUID entity.
func NewMachineUUIDRepository(db *gorm.DB) repositories.MachineUUIDRepository {
	return &MachineUUIDRepositoryGORM{NewGenericEntityRepository[types.MachineUUID](db)}
}

// ConnectionRepositoryGORM is a GORM implementation of the ConnectionRepository interface.
type ConnectionRepositoryGORM struct {
	repositories.GenericRepository[types.Connection]
}

// NewConnectionRepository creates a new instance of ConnectionRepositoryGORM.
// It initializes and returns a GORM-based repository for Connection entities.
func NewConnectionRepository(db *gorm.DB) repositories.ConnectionRepository {
	return &ConnectionRepositoryGORM{NewGenericRepository[types.Connection](db)}
}

// ElasticTokenRepositoryGORM is a GORM implementation of the ElasticTokenRepository interface.
type ElasticTokenRepositoryGORM struct {
	repositories.GenericRepository[types.ElasticToken]
}

// NewElasticTokenRepository creates a new instance of ElasticTokenRepositoryGORM.
// It initializes and returns a GORM-based repository for ElasticToken entities.
func NewElasticTokenRepository(db *gorm.DB) repositories.ElasticTokenRepository {
	return &ElasticTokenRepositoryGORM{NewGenericRepository[types.ElasticToken](db)}
}
