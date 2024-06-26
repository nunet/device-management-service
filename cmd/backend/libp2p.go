package backend

import (
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
)

type P2P struct{}

func (p *P2P) ClearIncomingChatRequests() error {
	log.Println("WARNING: Bypassing ClearIncomingChatRequests() in libp2p.go")
	// TODO: Uncomment after refactoring
	// return libp2p.ClearIncomingChatRequests()
	// END
	return nil
}

func (p *P2P) Decode(id string) (peer.ID, error) {
	return peer.Decode(id)
}
