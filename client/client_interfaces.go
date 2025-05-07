package client

import (
	"context"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
)

// MessageOptions contains common options for actor message operations
type MessageOptions struct {
	// Timeout is the timeout duration for the operation (equivalent to --timeout/-t flag)
	Timeout time.Duration
	// Expiry is the expiration time for the message (equivalent to --expiry/-e flag)
	Expiry time.Time
	// Destination is the destination DMS DID, peer ID or handle (equivalent to --dest/-d flag)
	Destination string
	// Topic for broadcast messages
	Topic string
	// IsInvocation indicates whether the message is an invocation
	IsInvocation bool
	// ReplyTo specifies the reply address for the message
	ReplyTo string
}

// NewMessageOptions creates a new MessageOptions with default values
func NewMessageOptions(msgOpts ...Option) MessageOptions {
	opts := MessageOptions{
		Timeout:      0,
		Expiry:       time.Time{},
		Destination:  "",
		Topic:        "",
		IsInvocation: false,
		ReplyTo:      "",
	}

	for _, opt := range msgOpts {
		opt(&opts)
	}

	return opts
}

// Option is a function that configures a MessageOptions
type Option func(*MessageOptions)

// WithTimeout sets the timeout duration
func WithTimeout(timeout time.Duration) Option {
	return func(o *MessageOptions) {
		o.Timeout = timeout
	}
}

// WithExpiry sets the expiry time
func WithExpiry(expiry time.Time) Option {
	return func(o *MessageOptions) {
		o.Expiry = expiry
	}
}

// WithDestination sets the destination
func WithDestination(destination string) Option {
	return func(o *MessageOptions) {
		o.Destination = destination
	}
}

// WithTopic sets the topic for broadcast messages
func WithTopic(topic string) Option {
	return func(o *MessageOptions) {
		o.Topic = topic
		o.IsInvocation = false
	}
}

// WithInvocation sets whether the message is an invocation
func WithInvocation(isInvocation bool) Option {
	return func(o *MessageOptions) {
		o.IsInvocation = isInvocation
	}
}

// WithReplyTo sets the reply address for the message
func WithReplyTo(replyTo string) Option {
	return func(o *MessageOptions) {
		o.ReplyTo = replyTo
	}
}

// DmsClient is the top-level client interface for the DMS service
type DmsClient interface {
	ActorClient
	ActorBehaviorsClient
}

// ActorClient provides methods for general actor message operations
type ActorClient interface {
	// NewActorMessage creates a new actor message
	NewActorMessage(ctx context.Context, behavior string, payload any, msgOpts MessageOptions) (actor.Envelope, error)

	// SendMessageRaw sends a message to a specific actor
	SendMessageRaw(ctx context.Context, msg actor.Envelope) (actor.Envelope, error)

	// SendMessage creates a new actor message and sends it to a specific actor
	SendMessage(ctx context.Context, behavior string, payload any, msgOpts ...Option) (actor.Envelope, error)

	// InvokeBehaviorRaw invokes a behavior on an actor
	InvokeBehaviorRaw(ctx context.Context, msg actor.Envelope) (actor.Envelope, error)

	// InvokeBehavior creates a new actor message and invokes a behavior on an actor
	InvokeBehavior(ctx context.Context, behavior string, payload any, msgOpts ...Option) (actor.Envelope, error)

	// BroadcastMessageRaw broadcasts a message to a topic
	BroadcastMessageRaw(ctx context.Context, msg actor.Envelope) ([]actor.Envelope, error)

	// BroadcastMessage creates a new actor message and broadcasts a message to a topic
	BroadcastMessage(ctx context.Context, behavior string, topic string, payload any, msgOpts ...Option) ([]actor.Envelope, error)
}

// ActorBehaviorsClient provides access to actor behavior methods
type ActorBehaviorsClient interface {
	ActorPublicBehaviorClient
	ActorPeersBehaviorClient
	ActorOnboardingBehaviorClient
	ActorDeploymentBehaviorClient
	ActorAllocationsBehaviorClient
	ActorSubnetBehaviorClient
	ActorResourcesBehaviorClient
	ActorHardwareBehaviorClient
	ActorCapBehaviorClient
	ActorLoggerBehaviorClient
	ActorVolumeBehaviorClient
}

// ActorPublicBehaviorClient provides methods for public behaviors
type ActorPublicBehaviorClient interface {
	// Hello sends a hello message
	Hello(ctx context.Context, opts ...Option) (node.HelloResponse, error)

	// BroadcastHello broadcasts a hello message to a topic
	BroadcastHello(ctx context.Context, opts ...Option) ([]node.HelloResponse, error)

	// Status retrieves the status of the actor
	Status(ctx context.Context, opts ...Option) (node.PublicStatusResponse, error)
}

// ActorPeersBehaviorClient provides methods for peer-related behaviors
type ActorPeersBehaviorClient interface {
	// PeersSelf retrieves information about the actor's own peer
	PeersSelf(ctx context.Context, opts ...Option) (node.PeerAddrInfoResponse, error)

	// PeersList lists the peers connected to the actor
	PeersList(ctx context.Context, opts ...Option) (node.PeersListResponse, error)

	// PeersListFromDht lists peers from the DHT
	PeersListFromDHT(ctx context.Context, opts ...Option) (node.PeerDHTResponse, error)

	// PeersPing pings a peer
	PeerPing(ctx context.Context, req node.PingRequest, opts ...Option) (node.PingResponse, error)

	// PeersConnect connects to a peer
	PeerConnect(ctx context.Context, req node.PeerConnectRequest, opts ...Option) (node.PeerConnectResponse, error)

	// PeersScore retrieves the score of peers
	PeerScore(ctx context.Context, opts ...Option) (node.PeerScoreResponse, error)
}

// ActorOnboardingBehaviorClient provides methods for onboarding
type ActorOnboardingBehaviorClient interface {
	// Onboard performs onboarding
	Onboard(ctx context.Context, req node.OnboardRequest, opts ...Option) (node.OnboardResponse, error)

	// Offboard performs offboarding
	Offboard(ctx context.Context, req node.OffboardRequest, opts ...Option) (node.OffboardResponse, error)

	// OnboardStatus retrieves onboarding status
	OnboardStatus(ctx context.Context, opts ...Option) (node.OnboardStatusResponse, error)
}

// ActorDeploymentBehaviorClient provides methods for deployment
type ActorDeploymentBehaviorClient interface {
	// DeploymentList lists deployments
	DeploymentList(ctx context.Context, req node.DeploymentListRequest, opts ...Option) (node.DeploymentListResponse, error)

	// DeploymentStatus retrieves the status of a deployment
	DeploymentStatus(ctx context.Context, req node.DeploymentStatusRequest, opts ...Option) (node.DeploymentStatusResponse, error)

	// DeploymentLogs retrieves the logs of a deployment
	DeploymentLogs(ctx context.Context, req node.DeploymentLogsRequest, opts ...Option) (node.DeploymentLogsResponse, error)

	// DeploymentManifest retrieves the manifest of a deployment
	DeploymentManifest(ctx context.Context, req node.DeploymentManifestRequest, opts ...Option) (node.DeploymentManifestResponse, error)

	// DeploymentShutdown shuts down a deployment
	DeploymentShutdown(ctx context.Context, req node.DeploymentShutdownRequest, opts ...Option) (node.DeploymentShutdownResponse, error)

	// DeploymentNew creates a new deployment
	DeploymentNew(ctx context.Context, req node.NewDeploymentRequest, opts ...Option) (node.NewDeploymentResponse, error)
}

// ActorAllocationsBehaviorClient provides methods for allocations view
type ActorAllocationsBehaviorClient interface {
	AllocationsList(ctx context.Context, opts ...Option) (node.AllocationsListResponse, error)
}

// ActorSubnetBehaviorClient provides methods for subnet management
type ActorSubnetBehaviorClient interface {
	// SubnetCreate creates a new subnet
	SubnetCreate(ctx context.Context, req orchestrator.SubnetCreateRequest, opts ...Option) (orchestrator.SubnetCreateResponse, error)

	// SubnetDestroy destroys a subnet
	SubnetDestroy(ctx context.Context, req orchestrator.SubnetDestroyRequest, opts ...Option) (orchestrator.SubnetDestroyResponse, error)

	// SubnetJoin joins a subnet
	SubnetJoin(ctx context.Context, req orchestrator.SubnetJoinRequest, opts ...Option) (orchestrator.SubnetJoinResponse, error)

	// SubnetAddPeer adds a peer to a subnet
	SubnetAddPeer(ctx context.Context, req jobs.SubnetAddPeerRequest, opts ...Option) (jobs.SubnetAddPeerResponse, error)

	// SubnetRemovePeer removes a peer from a subnet
	SubnetRemovePeer(ctx context.Context, req jobs.SubnetRemovePeerRequest, opts ...Option) (jobs.SubnetRemovePeerResponse, error)

	// SubnetAcceptPeer accepts a peer in a subnet
	SubnetAcceptPeer(ctx context.Context, req jobs.SubnetAcceptPeerRequest, opts ...Option) (jobs.SubnetAcceptPeerResponse, error)

	// SubnetMapPort maps a port in a subnet
	SubnetMapPort(ctx context.Context, req jobs.SubnetMapPortRequest, opts ...Option) (jobs.SubnetMapPortResponse, error)

	// SubnetUnmapPort unmaps a port in a subnet
	SubnetUnmapPort(ctx context.Context, req jobs.SubnetUnmapPortRequest, opts ...Option) (jobs.SubnetUnmapPortResponse, error)

	// SubnetDNSAddRecords adds DNS records to a subnet
	SubnetDNSAddRecords(ctx context.Context, req jobs.SubnetDNSAddRecordsRequest, opts ...Option) (jobs.SubnetDNSAddRecordsResponse, error)

	// SubnetDNSRemoveRecord removes a DNS record from a subnet
	SubnetDNSRemoveRecord(ctx context.Context, req jobs.SubnetDNSRemoveRecordRequest, opts ...Option) (jobs.SubnetDNSRemoveRecordResponse, error)
}

// ActorResourcesBehaviorClient provides methods for resource management
type ActorResourcesBehaviorClient interface {
	// ResourcesAllocated retrieves allocated resources
	ResourcesAllocated(ctx context.Context, opts ...Option) (node.ResourcesResponse, error)

	// ResourcesFree retrieves free resources
	ResourcesFree(ctx context.Context, opts ...Option) (node.ResourcesResponse, error)

	// ResourcesOnboarded retrieves onboarded resources
	ResourcesOnboarded(ctx context.Context, opts ...Option) (node.ResourcesResponse, error)
}

// ActorHardwareBehaviorClient provides methods for hardware information
type ActorHardwareBehaviorClient interface {
	// HardwareSpec retrieves hardware specifications
	HardwareSpec(ctx context.Context, opts ...Option) (node.ResourcesResponse, error)

	// HardwareUsage retrieves hardware usage
	HardwareUsage(ctx context.Context, opts ...Option) (node.ResourcesResponse, error)
}

// ActorCapBehaviorClient provides methods for capability management
type ActorCapBehaviorClient interface {
	// CapList retrieves capability list
	CapList(ctx context.Context, req node.CapListRequest, opts ...Option) (node.CapListResponse, error)

	// CapAnchor updates capabilities
	CapAnchor(ctx context.Context, req node.CapAnchorRequest, opts ...Option) (node.CapAnchorResponse, error)
}

// ActorLoggerBehaviorClient provides methods for logger configuration
type ActorLoggerBehaviorClient interface {
	// LoggerConfig configures the logger
	LoggerConfig(ctx context.Context, req node.LoggerConfigRequest, opts ...Option) (node.LoggerConfigResponse, error)
}

// ActorVolumeBehaviorClient provides methods for volume management
type ActorVolumeBehaviorClient interface {
	// CreateVolume creates a new volume
	CreateVolume(ctx context.Context, req node.CreateVolumeRequest, opts ...Option) (node.CreateVolumeResponse, error)

	// DeleteVolume deletes a volume
	DeleteVolume(ctx context.Context, req node.DeleteVolumeRequest, opts ...Option) (node.DeleteVolumeResponse, error)

	// StartVolume starts a volume
	StartVolume(ctx context.Context, req node.StartVolumeRequest, opts ...Option) (node.StartVolumeResponse, error)
}
