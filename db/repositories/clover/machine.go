package clover

import (
	"github.com/ostafen/clover/v2"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// PeerInfoClover is a Clover implementation of the PeerInfo interface.
type PeerInfoClover struct {
	repositories.GenericRepository[types.PeerInfo]
}

// NewPeerInfo creates a new instance of PeerInfoClover.
// It initializes and returns a Clover-based repository for PeerInfo entities.
func NewPeerInfo(db *clover.DB) repositories.PeerInfo {
	return &PeerInfoClover{NewGenericRepository[types.PeerInfo](db)}
}

// MachineClover is a Clover implementation of the Machine interface.
type MachineClover struct {
	repositories.GenericRepository[types.Machine]
}

// NewMachine creates a new instance of MachineClover.
// It initializes and returns a Clover-based repository for Machine entities.
func NewMachine(db *clover.DB) repositories.Machine {
	return &MachineClover{NewGenericRepository[types.Machine](db)}
}

// ServicesClover is a Clover implementation of the Services interface.
type ServicesClover struct {
	repositories.GenericRepository[types.Services]
}

// NewServices creates a new instance of ServicesClover.
// It initializes and returns a Clover-based repository for Services entities.
func NewServices(db *clover.DB) repositories.Services {
	return &ServicesClover{NewGenericRepository[types.Services](db)}
}

// ServiceResourceRequirementsClover is a Clover implementation of the ServiceResourceRequirements interface.
type ServiceResourceRequirementsClover struct {
	repositories.GenericRepository[types.ServiceResourceRequirements]
}

// NewServiceResourceRequirements creates a new instance of ServiceResourceRequirementsClover.
// It initializes and returns a Clover-based repository for ServiceResourceRequirements entities.
func NewServiceResourceRequirements(
	db *clover.DB,
) repositories.ServiceResourceRequirements {
	return &ServiceResourceRequirementsClover{
		NewGenericRepository[types.ServiceResourceRequirements](db),
	}
}

// Libp2pInfoClover is a Clover implementation of the Libp2pInfo interface.
type Libp2pInfoClover struct {
	repositories.GenericEntityRepository[types.Libp2pInfo]
}

// NewLibp2pInfo creates a new instance of Libp2pInfoClover.
// It initializes and returns a Clover-based repository for Libp2pInfo entity.
func NewLibp2pInfo(db *clover.DB) repositories.Libp2pInfo {
	return &Libp2pInfoClover{NewGenericEntityRepository[types.Libp2pInfo](db)}
}

// MachineUUIDClover is a Clover implementation of the MachineUUID interface.
type MachineUUIDClover struct {
	repositories.GenericEntityRepository[types.MachineUUID]
}

// NewMachineUUID creates a new instance of MachineUUIDClover.
// It initializes and returns a Clover-based repository for MachineUUID entity.
func NewMachineUUID(db *clover.DB) repositories.MachineUUID {
	return &MachineUUIDClover{NewGenericEntityRepository[types.MachineUUID](db)}
}

// ConnectionClover is a Clover implementation of the Connection interface.
type ConnectionClover struct {
	repositories.GenericRepository[types.Connection]
}

// NewConnection creates a new instance of ConnectionClover.
// It initializes and returns a Clover-based repository for Connection entities.
func NewConnection(db *clover.DB) repositories.Connection {
	return &ConnectionClover{NewGenericRepository[types.Connection](db)}
}

// ElasticTokenClover is a Clover implementation of the ElasticToken interface.
type ElasticTokenClover struct {
	repositories.GenericRepository[types.ElasticToken]
}

// NewElasticToken creates a new instance of ElasticTokenClover.
// It initializes and returns a Clover-based repository for ElasticToken entities.
func NewElasticToken(db *clover.DB) repositories.ElasticToken {
	return &ElasticTokenClover{NewGenericRepository[types.ElasticToken](db)}
}
