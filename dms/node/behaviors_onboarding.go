package node

import (
	"context"
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardRequest struct {
	NoGPU  bool
	GPUs   string
	Config types.OnboardingConfig
}

type OnboardResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Config  types.OnboardingConfig `json:"config,omitempty"`
}

func (n *Node) handleOnboard(msg actor.Envelope) {
	defer msg.Discard()

	resp := OnboardResponse{}

	var request OnboardRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	config, err := n.onboarder.Onboard(context.Background(), request.Config)
	if err != nil {
		resp.Error = err.Error()
		resp.Success = false
		n.sendReply(msg, resp)
		return
	}

	resp.Config = config
	resp.Success = true
	n.sendReply(msg, resp)
}

type OffboardRequest struct{}

type OffboardResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (n *Node) handleOffboard(msg actor.Envelope) {
	defer msg.Discard()

	resp := OffboardResponse{}
	if err := n.onboarder.Offboard(context.Background()); err != nil {
		resp.Success = false
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Success = true
	n.sendReply(msg, resp)
}

type OnboardStatusResponse struct {
	Onboarded bool   `json:"onboarded"`
	Error     string `json:"error,omitempty"`
}

func (n *Node) handleOnboardStatus(msg actor.Envelope) {
	defer msg.Discard()
	resp := OnboardStatusResponse{}

	onboarded, err := n.onboarder.IsOnboarded()
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.Onboarded = onboarded
	n.sendReply(msg, resp)
}
