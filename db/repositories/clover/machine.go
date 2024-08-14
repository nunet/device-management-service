package repositories_clover

import (
	"github.com/ostafen/clover/v2"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfoRepositoryClover is a Clover implementation of the PeerInfoRepository interface.
type PeerInfoRepositoryClover struct {
	repositories.GenericRepository[types.PeerInfo]
}

// NewPeerInfoRepository creates a new instance of PeerInfoRepositoryClover.
// It initializes and returns a Clover-based repository for PeerInfo entities.
func NewPeerInfoRepository(db *clover.DB) repositories.PeerInfoRepository {
	return &PeerInfoRepositoryClover{NewGenericRepository[types.PeerInfo](db)}
}

// MachineRepositoryClover is a Clover implementation of the MachineRepository interface.
type MachineRepositoryClover struct {
	repositories.GenericRepository[types.Machine]
}

// NewMachineRepository creates a new instance of MachineRepositoryClover.
// It initializes and returns a Clover-based repository for Machine entities.
func NewMachineRepository(db *clover.DB) repositories.MachineRepository {
	return &MachineRepositoryClover{NewGenericRepository[types.Machine](db)}
}

// FreeResourcesRepositoryClover is a Clover implementation of the FreeResourcesRepository interface.
type FreeResourcesRepositoryClover struct {
	repositories.GenericEntityRepository[types.FreeResources]
}

// NewFreeResourcesRepository creates a new instance of FreeResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for FreeResources entity.
func NewFreeResourcesRepository(db *clover.DB) repositories.FreeResourcesRepository {
	return &FreeResourcesRepositoryClover{NewGenericEntityRepository[types.FreeResources](db)}
}

// AvailableResourcesRepositoryClover is a Clover implementation of the AvailableResourcesRepository interface.
type AvailableResourcesRepositoryClover struct {
	repositories.GenericEntityRepository[types.AvailableResources]
}

// NewAvailableResourcesRepository creates a new instance of AvailableResourcesRepositoryClover.
// It initializes and returns a Clover-based repository for AvailableResources entity.
func NewAvailableResourcesRepository(db *clover.DB) repositories.AvailableResourcesRepository {
	return &AvailableResourcesRepositoryClover{
		NewGenericEntityRepository[types.AvailableResources](db),
	}
}

// ServicesRepositoryClover is a Clover implementation of the ServicesRepository interface.
type ServicesRepositoryClover struct {
	repositories.GenericRepository[types.Services]
}

// NewServicesRepository creates a new instance of ServicesRepositoryClover.
// It initializes and returns a Clover-based repository for Services entities.
func NewServicesRepository(db *clover.DB) repositories.ServicesRepository {
	return &ServicesRepositoryClover{NewGenericRepository[types.Services](db)}
}

// ServiceResourceRequirementsRepositoryClover is a Clover implementation of the ServiceResourceRequirementsRepository interface.
type ServiceResourceRequirementsRepositoryClover struct {
	repositories.GenericRepository[types.ServiceResourceRequirements]
}

// NewServiceResourceRequirementsRepository creates a new instance of ServiceResourceRequirementsRepositoryClover.
// It initializes and returns a Clover-based repository for ServiceResourceRequirements entities.
func NewServiceResourceRequirementsRepository(
	db *clover.DB,
) repositories.ServiceResourceRequirementsRepository {
	return &ServiceResourceRequirementsRepositoryClover{
		NewGenericRepository[types.ServiceResourceRequirements](db),
	}
}

// Libp2pInfoRepositoryClover is a Clover implementation of the Libp2pInfoRepository interface.
type Libp2pInfoRepositoryClover struct {
	repositories.GenericEntityRepository[types.Libp2pInfo]
}

// NewLibp2pInfoRepository creates a new instance of Libp2pInfoRepositoryClover.
// It initializes and returns a Clover-based repository for Libp2pInfo entity.
func NewLibp2pInfoRepository(db *clover.DB) repositories.Libp2pInfoRepository {
	return &Libp2pInfoRepositoryClover{NewGenericEntityRepository[types.Libp2pInfo](db)}
}

// MachineUUIDRepositoryClover is a Clover implementation of the MachineUUIDRepository interface.
type MachineUUIDRepositoryClover struct {
	repositories.GenericEntityRepository[types.MachineUUID]
}

// NewMachineUUIDRepository creates a new instance of MachineUUIDRepositoryClover.
// It initializes and returns a Clover-based repository for MachineUUID entity.
func NewMachineUUIDRepository(db *clover.DB) repositories.MachineUUIDRepository {
	return &MachineUUIDRepositoryClover{NewGenericEntityRepository[types.MachineUUID](db)}
}

// ConnectionRepositoryClover is a Clover implementation of the ConnectionRepository interface.
type ConnectionRepositoryClover struct {
	repositories.GenericRepository[types.Connection]
}

// NewConnectionRepository creates a new instance of ConnectionRepositoryClover.
// It initializes and returns a Clover-based repository for Connection entities.
func NewConnectionRepository(db *clover.DB) repositories.ConnectionRepository {
	return &ConnectionRepositoryClover{NewGenericRepository[types.Connection](db)}
}

// ElasticTokenRepositoryClover is a Clover implementation of the ElasticTokenRepository interface.
type ElasticTokenRepositoryClover struct {
	repositories.GenericRepository[types.ElasticToken]
}

// NewElasticTokenRepository creates a new instance of ElasticTokenRepositoryClover.
// It initializes and returns a Clover-based repository for ElasticToken entities.
func NewElasticTokenRepository(db *clover.DB) repositories.ElasticTokenRepository {
	return &ElasticTokenRepositoryClover{NewGenericRepository[types.ElasticToken](db)}
}
