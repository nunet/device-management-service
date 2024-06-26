package libp2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"gitlab.com/nunet/device-management-service/models"
)

type Libp2pPeer struct {
}

func (p *Libp2pPeer) Config() error {
	return nil
}

func (p *Libp2pPeer) Init() error {
	return nil
}

func (p *Libp2pPeer) EventRegister() error {
	return nil
}

func (p *Libp2pPeer) Status() error {
	return nil
}

func (p *Libp2pPeer) Stop() error {
	return nil
}

func CleanupPeer(id peer.ID) error {
	zlog.Warn("CleanupPeer: Stub")
	return nil
}

func PingPeer(ctx context.Context, target peer.ID) (bool, *ping.Result) {
	zlog.Warn("PingPeer: Stub")
	return false, nil
}

func DumpKademliaDHT(ctx context.Context) ([]models.PeerData, error) {
	zlog.Warn("DumpKademliaDHT: Stub")
	return nil, nil
}

func OldPingPeer(ctx context.Context, target peer.ID) (bool, *models.PingResult) {
	zlog.Warn("OldPingPeer: Stub")
	return false, nil
}
