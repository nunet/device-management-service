// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package network

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/lib/crypto"

	"github.com/spf13/afero"
	"gitlab.com/nunet/device-management-service/internal/config"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"

	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
)

type (
	PeerID            = libp2p.PeerID
	ProtocolID        = libp2p.ProtocolID
	Topic             = libp2p.Topic
	Validator         = libp2p.Validator
	ValidationResult  = libp2p.ValidationResult
	PeerScoreSnapshot = libp2p.PeerScoreSnapshot
)

const (
	ValidationAccept = libp2p.ValidationAccept
	ValidationReject = libp2p.ValidationReject
	ValidationIgnore = libp2p.ValidationIgnore
)

// Messenger defines the interface for sending messages.
type Messenger interface {
	// SendMessage asynchronously sends a message to the given peer.
	SendMessage(ctx context.Context, hostID string, msg types.MessageEnvelope, expiry time.Time) error

	// SendMessageSync synchronously sends a message to the given peer.
	// This method blocks until the message has been sent.
	SendMessageSync(ctx context.Context, hostID string, msg types.MessageEnvelope, expiry time.Time) error
}

type Network interface {
	// Messenger embedded interface
	Messenger

	// Init initializes the network
	Init(*config.Config) error
	// Start starts the network
	Start() error
	// Stat returns the network information
	Stat() types.NetworkStats
	// Ping pings the given address and returns the PingResult
	Ping(ctx context.Context, address string, timeout time.Duration) (types.PingResult, error)
	// GetHostID returns the host ID
	GetHostID() PeerID
	// GetPeerPubKey returns the public key for the given peerID
	GetPeerPubKey(peerID PeerID) crypto.PubKey
	// HandleMessage is responsible for registering a message type and its handler.
	HandleMessage(messageType string, handler func(data []byte)) error
	// UnregisterMessageHandler unregisters a stream handler for a specific protocol.
	UnregisterMessageHandler(messageType string)
	// ResolveAddress given an id it retruns the address of the peer.
	// In libp2p, id represents the peerID and the response is the addrinfo
	ResolveAddress(ctx context.Context, id string) ([]string, error)
	// Advertise advertises the given data with the given adId
	// such as advertising device capabilities on the DHT
	Advertise(ctx context.Context, key string, data []byte) error
	// Unadvertise stops advertising data corresponding to the given adId
	Unadvertise(ctx context.Context, key string) error
	// Query returns the network advertisement
	Query(ctx context.Context, key string) ([]*commonproto.Advertisement, error)
	// Publish publishes the given data to the given topic if the network
	// type allows publish/subscribe functionality such as gossipsub or nats
	Publish(ctx context.Context, topic string, data []byte) error
	// Subscribe subscribes to the given topic and calls the handler function
	// if the network type allows it similar to Publish()
	Subscribe(ctx context.Context, topic string, handler func(data []byte), validator libp2p.Validator) (uint64, error)
	// Unsubscribe from a topic
	Unsubscribe(topic string, subID uint64) error
	// SetupBroadcastTopic allows the application to configure pubsub topic directly
	SetupBroadcastTopic(topic string, setup func(*Topic) error) error
	// SetupBroadcastAppScore allows the application to configure application level
	// scoring for pubsub
	SetBroadcastAppScore(func(PeerID) float64)
	// GetBroadcastScore returns the latest broadcast score snapshot
	GetBroadcastScore() map[PeerID]*PeerScoreSnapshot
	// Notify allows the application to receive notifications about peer connections
	// and disconnecions
	Notify(ctx context.Context, preconnected func(PeerID, []ProtocolID, int), connected, disconnected func(PeerID), identified, updated func(PeerID, []ProtocolID)) error
	// PeerConnected returs true if the peer is currently connected
	PeerConnected(p PeerID) bool
	// Stop stops the network including any existing advertisements and subscriptions
	Stop() error

	// GetPeerIP returns the ipv4 or v6 of a peer
	GetPeerIP(p PeerID) string

	// CreateSubnet creates a subnet with the given subnetID and CIDR
	CreateSubnet(ctx context.Context, subnetID string, routingTable map[string]string) error

	// RemoveSubnet removes a subnet
	DestroySubnet(subnetID string) error

	// AddSubnetPeer adds a peer to the subnet
	AddSubnetPeer(subnetID, peerID, ip string) error

	// RemoveSubnetPeer removes a peer from the subnet
	RemoveSubnetPeer(subnetID, peerID, ip string) error

	// AcceptSubnetPeer accepts a peer to the subnet
	AcceptSubnetPeer(subnetID, peerID, ip string) error

	// MapPort maps a sourceIp:sourcePort to destIP:destPort
	MapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error

	// UnmapPort removes a previous port map
	UnmapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error

	// AddSubnetDNSRecords adds dns records to our local resolver
	AddSubnetDNSRecords(subnetID string, records map[string]string) error

	// RemoveDNSRecord removes a dns record from our local resolver
	RemoveSubnetDNSRecord(subnetID, name string) error
}

// NewNetwork returns a new network given the configuration.
func NewNetwork(netConfig *types.NetworkConfig, fs afero.Fs) (Network, error) {
	// TODO: probable additional params to receive: DB, FileSystem
	if netConfig == nil {
		return nil, errors.New("network configuration is nil")
	}
	switch netConfig.Type {
	case types.Libp2pNetwork:
		ln, err := libp2p.New(&netConfig.Libp2pConfig, fs)
		return ln, err
	case types.NATSNetwork:
		return nil, errors.New("not implemented")
	default:
		return nil, fmt.Errorf("unsupported network type: %s", netConfig.Type)
	}
}
