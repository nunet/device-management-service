package node

import (
	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	PublicHelloBehavior    = "/public/hello"
	PublicStatusBehavior   = "/public/status"
	BroadcastHelloBehavior = "/broadcast/hello"
	BroadcastHelloTopic    = "/nunet/hello"
)

type PublicStatusResponse struct {
	Status    string
	Resources types.Resources
}

func (n *Node) publicHelloBehavior(msg actor.Envelope) {
	pubk, err := did.PublicKeyFromDID(msg.From.DID)
	if err != nil {
		log.Debugf("failed to extract public key from DID: %s", err)
		return
	}

	p, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		log.Debugf("failed to extract peer ID from public key: %s", err)
		return
	}

	n.mx.Lock()
	if st, ok := n.peers[p]; ok {
		st.helloIn = true
	} else if n.network.PeerConnected(p) {
		// rance with connected notification
		st = &peerState{helloIn: true}
		n.peers[p] = st
	}
	n.mx.Unlock()

	n.handleHello(msg)
}

func (n *Node) broadcastHelloBehavior(msg actor.Envelope) {
	n.handleHello(msg)
}

func (n *Node) handleHello(msg actor.Envelope) {
	defer msg.Discard()
	log.Debugf("hello from %s", msg.From)

	n.sendReply(msg, nil)
}

func (n *Node) publicStatusBehavior(msg actor.Envelope) {
	defer msg.Discard()

	var resp PublicStatusResponse
	machineResources, err := n.resourceManager.SystemSpecs().GetMachineResources()
	if err != nil {
		resp.Status = "ERROR"
	} else {
		resp.Status = "OK"
		resp.Resources = machineResources.Resources
	}

	n.sendReply(msg, resp)
}
