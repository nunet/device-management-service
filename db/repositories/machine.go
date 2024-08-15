package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfo represents a repository for CRUD operations on PeerInfo entities.
type PeerInfo interface {
	GenericRepository[types.PeerInfo]
}

// Machine represents a repository for CRUD operations on Machine entities.
type Machine interface {
	GenericRepository[types.Machine]
}

// FreeResources represents a repository for CRUD operations on FreeResources entity.
type FreeResources interface {
	GenericEntityRepository[types.FreeResources]
}

// AvailableResources represents a repository for CRUD operations on AvailableResources entity.
type AvailableResources interface {
	GenericEntityRepository[types.AvailableResources]
}

// Services represents a repository for CRUD operations on Services entities.
type Services interface {
	GenericRepository[types.Services]
}

// ServiceResourceRequirements represents a repository for CRUD operations on ServiceResourceRequirements entities.
type ServiceResourceRequirements interface {
	GenericRepository[types.ServiceResourceRequirements]
}

// Libp2pInfo represents a repository for CRUD operations on Libp2pInfo entity.
type Libp2pInfo interface {
	GenericEntityRepository[types.Libp2pInfo]
}

// MachineUUID represents a repository for CRUD operations on MachineUUID entity.
type MachineUUID interface {
	GenericEntityRepository[types.MachineUUID]
}

// Connection represents a repository for CRUD operations on Connection entities.
type Connection interface {
	GenericRepository[types.Connection]
}

// ElasticToken represents a repository for CRUD operations on ElasticToken entities.
type ElasticToken interface {
	GenericRepository[types.ElasticToken]
}
