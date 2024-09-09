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

	reply, err := actor.ReplyTo(msg, nil)
	if err != nil {
		log.Debugf("error creating hello reply: %s", err)
		return
	}

	if err := n.actor.Send(reply); err != nil {
		log.Debugf("error sending hello reply: %s", err)
	}
}

func (n *Node) publicStatusBehavior(msg actor.Envelope) {
	defer msg.Discard()

	var resp PublicStatusResponse
	rc, err := n.resourceManager.SystemSpecs().GetProvisionedResources()
	if err != nil {
		resp.Status = "ERROR"
	} else {
		resp.Status = "OK"
		resp.Resources = rc
	}

	reply, err := actor.ReplyTo(msg, resp)
	if err != nil {
		log.Debugf("error creating status reply: %s", err)
		return
	}

	if err = n.actor.Send(reply); err != nil {
		log.Debugf("error sending status reply: %s", err)
	}
}
