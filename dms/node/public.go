package node

import (
	"gitlab.com/nunet/device-management-service/dms/actor"
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
