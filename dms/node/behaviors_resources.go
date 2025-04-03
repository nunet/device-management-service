// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

type ResourcesResponse struct {
	OK        bool
	Resources types.Resources
	Error     string `json:"error,omitempty"`
}

func (n *Node) handleAllocatedResources(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling allocated resources: %s", err)
		n.sendReply(msg, ResourcesResponse{Error: err.Error()})
	}

	resp := ResourcesResponse{}

	allocatedResources, err := n.resourceManager.GetTotalAllocation()
	if err != nil {
		handleErr(err)
		return
	}

	resp.Resources = allocatedResources
	n.sendReply(msg, resp)
}

func (n *Node) handleFreeResources(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling free resources: %s", err)
		n.sendReply(msg, ResourcesResponse{Error: err.Error()})
	}

	resp := ResourcesResponse{}

	freeResources, err := n.resourceManager.GetFreeResources(context.Background())
	if err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	resp.Resources = freeResources.Resources
	n.sendReply(msg, resp)
}

func (n *Node) handleOnboardedResources(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling onboarded resources: %s", err)
		n.sendReply(msg, ResourcesResponse{Error: err.Error()})
	}

	resp := ResourcesResponse{}

	onboardedResources, err := n.resourceManager.GetOnboardedResources(context.Background())
	if err != nil {
		handleErr(err)
		return
	}

	resp.Resources = onboardedResources.Resources
	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleHardwareUsage(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling hardware usage: %s", err)
		n.sendReply(msg, ResourcesResponse{Error: err.Error()})
	}

	resp := ResourcesResponse{}

	hardwareUsage, err := n.hardware.GetUsage()
	if err != nil {
		handleErr(err)
		return
	}

	resp.Resources = hardwareUsage
	n.sendReply(msg, resp)
}
