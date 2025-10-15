// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client

import (
	"context"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
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
	ActorResourcesBehaviorClient
	ActorHardwareBehaviorClient
	ActorCapBehaviorClient
	ActorLoggerBehaviorClient
	ActorVolumeBehaviorClient
	ActorContractBehaviorClient
}

// ActorPublicBehaviorClient provides methods for public behaviors
type ActorPublicBehaviorClient interface {
	// Hello sends a hello message
	Hello(ctx context.Context, opts ...Option) (node.HelloResponse, error)

	// BroadcastHello broadcasts a hello message to a topic
	BroadcastHello(ctx context.Context, opts ...Option) ([]node.HelloResponse, error)

	// Status retrieves the status of the actor
	Status(ctx context.Context, opts ...Option) (node.PublicStatusResponse, error)

	// Discovery retrieves the discovery information of the actor
	Discovery(ctx context.Context, opts ...Option) (node.DiscoveryStatusResponse, error)

	// DiscoveryBroadcast broadcasts the discovery information of the actor
	DiscoveryBroadcast(ctx context.Context, opts ...Option) ([]node.DiscoveryStatusResponse, error)
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

	// Flightrec dump a flight recorder snapshot
	Flightrec(ctx context.Context, opts ...Option) (node.PingResponse, error)
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

	// DeploymentUpdate updates a running deployment
	DeploymentUpdate(ctx context.Context, req node.UpdateDeploymentRequest, opts ...Option) (node.UpdateDeploymentResponse, error)

	// DeploymentPrune removes old deployments
	DeploymentPrune(ctx context.Context, req node.DeploymentPruneRequest, opts ...Option) (node.DeploymentPruneResponse, error)

	// DeploymentDelete removes a specific deployment
	DeploymentDelete(ctx context.Context, req node.DeploymentDeleteRequest, opts ...Option) (node.DeploymentDeleteResponse, error)
}

// ActorAllocationsBehaviorClient provides methods for allocations view
type ActorAllocationsBehaviorClient interface {
	AllocationsList(ctx context.Context, opts ...Option) (node.AllocationsListResponse, error)
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

// ActorContractBehaviorClient provides methods for contracts
type ActorContractBehaviorClient interface {
	NewContract(ctx context.Context, req contracts.CreateContractRequestBehaviour, opts ...Option) (contracts.CreateContractResponseBehaviour, error)
	ContractStatus(ctx context.Context, req contracts.ContractStatusRequestBehaviour, opts ...Option) (contracts.ContractStatusResponseBehaviour, error)
	ApproveLocal(ctx context.Context, req contracts.ContractApproveLocalRequestBehaviour, opts ...Option) (contracts.ContractApproveLocalResponseBehaviour, error)
	ListIncoming(ctx context.Context, opts ...Option) (contracts.ContractListIncomingResponseBehaviour, error)
	ListTransactions(ctx context.Context, opts ...Option) (contracts.ContractListLocalTransactionsResponse, error)
	CollectUsagesAndForwardToPaymentProviders(ctx context.Context, opts ...Option) (contracts.CollectUsagesAndForwardToPaymentProvidersReponse, error)
	ConfirmTransaction(ctx context.Context, req contracts.ContractConfirmLocalTransactionRequest, opts ...Option) (contracts.ContractConfirmLocalTransactionResponse, error)
	GetPaymentStatus(ctx context.Context, req contracts.ContractPaymentStatusRequest, opts ...Option) (contracts.ContractPaymentStatusResponse, error)
	TerminateContract(ctx context.Context, req contracts.ContractTerminationRequestBehaviour, opts ...Option) (contracts.ContractTerminationResponseBehaviour, error)
	CompleteContract(ctx context.Context, req contracts.ContractCompletionRequestBehaviour, opts ...Option) (contracts.ContractCompletionResponseBehaviour, error)
	ValidateContract(ctx context.Context, req contracts.ContractValidateRequestBehaviour, opts ...Option) (contracts.ContractValidateResponseBehaviour, error)
	SettleContract(ctx context.Context, req contracts.ContractSettleRequestBehaviour, opts ...Option) (contracts.ContractSettleResponseBehaviour, error)
}
