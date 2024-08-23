package libp2p

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"google.golang.org/protobuf/proto"
)

// Bootstrap using a list.
func (l *Libp2p) Bootstrap(ctx context.Context, bootstrapPeers []multiaddr.Multiaddr) error {
	if err := l.DHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to prepare this node for bootstraping: %w", err)
	}

	// bootstrap all nodes at the same time.
	if len(bootstrapPeers) > 0 {
		var wg sync.WaitGroup
		for _, addr := range bootstrapPeers {
			wg.Add(1)
			go func(peerAddr multiaddr.Multiaddr) {
				defer wg.Done()
				addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
				if err != nil {
					zlog.Sugar().Errorf("failed to convert multi addr to addr info %v - %v", peerAddr, err)
					return
				}
				if err := l.Host.Connect(ctx, *addrInfo); err != nil {
					zlog.Sugar().Errorf("failed to connect to bootstrap node %s - %v", addrInfo.ID.String(), err)
				} else {
					zlog.Sugar().Infof("connected to Bootstrap Node %s", addrInfo.ID.String())
				}
			}(addr)
		}
		wg.Wait()
	}

	return nil
}

type dhtValidator struct {
	PS              peerstore.Peerstore
	customNamespace string
}

// Validate validates an item placed into the dht.
func (d dhtValidator) Validate(key string, value []byte) error {
	// empty value is considered deleting an item from the dht
	if len(value) == 0 {
		return nil
	}

	if !strings.HasPrefix(key, d.customNamespace) {
		return errors.New("invalid key namespace")
	}

	// verify signature
	var envelope commonproto.Advertisement
	err := proto.Unmarshal(value, &envelope)
	if err != nil {
		return fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	pubKey, err := crypto.UnmarshalSecp256k1PublicKey(envelope.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	concatenatedBytes := bytes.Join([][]byte{
		[]byte(envelope.PeerId),
		{byte(envelope.Timestamp)},
		envelope.Data,
		envelope.PublicKey,
	}, nil)
	ok, err := pubKey.Verify(concatenatedBytes, envelope.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify envelope: %w", err)
	}

	if !ok {
		return errors.New("failed to verify envelope, public key didn't sign payload")
	}

	return nil
}
func (dhtValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

// TODO remove the below when network package is fully implemented
// UpdateKadDHT is a stub
func (l *Libp2p) UpdateKadDHT() {
	zlog.Warn("UpdateKadDHT: Stub")
}

// ListKadDHTPeers is a stub
func (l *Libp2p) ListKadDHTPeers(_ context.Context, _ *gin.Context) ([]string, error) {
	zlog.Warn("ListKadDHTPeers: Stub")
	return nil, nil
}
