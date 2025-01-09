package node

import (
	"context"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

type resourcesResponse struct {
	OK        bool
	Resources types.Resources
	Error     string `json:"error,omitempty"`
}

func (n *Node) handleAllocatedResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	allocatedResources, err := n.resourceManager.GetTotalAllocation()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = allocatedResources
	n.sendReply(msg, resp)
}

func (n *Node) handleFreeResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	freeResources, err := n.resourceManager.GetFreeResources(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = freeResources.Resources
	n.sendReply(msg, resp)
}

func (n *Node) handleOnboardedResources(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	onboardedResources, err := n.resourceManager.GetOnboardedResources(context.Background())
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = onboardedResources.Resources
	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleHardwareUsage(msg actor.Envelope) {
	defer msg.Discard()
	resp := resourcesResponse{}

	hardwareUsage, err := n.hardware.GetUsage()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Resources = hardwareUsage
	n.sendReply(msg, resp)
}
