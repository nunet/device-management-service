package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfoRepository represents a repository for CRUD operations on PeerInfo entities.
type PeerInfoRepository interface {
	GenericRepository[types.PeerInfo]
}

// MachineRepository represents a repository for CRUD operations on Machine entities.
type MachineRepository interface {
	GenericRepository[types.Machine]
}

// FreeResourcesRepository represents a repository for CRUD operations on FreeResources entity.
type FreeResourcesRepository interface {
	GenericEntityRepository[types.FreeResources]
}

// AvailableResourcesRepository represents a repository for CRUD operations on AvailableResources entity.
type AvailableResourcesRepository interface {
	GenericEntityRepository[types.AvailableResources]
}

// ServicesRepository represents a repository for CRUD operations on Services entities.
type ServicesRepository interface {
	GenericRepository[types.Services]
}

// ServiceResourceRequirementsRepository represents a repository for CRUD operations on ServiceResourceRequirements entities.
type ServiceResourceRequirementsRepository interface {
	GenericRepository[types.ServiceResourceRequirements]
}

// Libp2pInfoRepository represents a repository for CRUD operations on Libp2pInfo entity.
type Libp2pInfoRepository interface {
	GenericEntityRepository[types.Libp2pInfo]
}

// MachineUUIDRepository represents a repository for CRUD operations on MachineUUID entity.
type MachineUUIDRepository interface {
	GenericEntityRepository[types.MachineUUID]
}

// ConnectionRepository represents a repository for CRUD operations on Connection entities.
type ConnectionRepository interface {
	GenericRepository[types.Connection]
}

// ElasticTokenRepository represents a repository for CRUD operations on ElasticToken entities.
type ElasticTokenRepository interface {
	GenericRepository[types.ElasticToken]
}
