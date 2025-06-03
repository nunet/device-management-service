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
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardRequest struct {
	NoGPU  bool
	GPUs   string
	Config types.OnboardingConfig
}

type OnboardResponse struct {
	OK     bool                   `json:"success"`
	Error  string                 `json:"error,omitempty"`
	Config types.OnboardingConfig `json:"config,omitempty"`
}

func (n *Node) handleOnboard(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("onboard_error",
			"labels", string(observability.LabelNode),
			"error", err)
		n.sendReply(msg, OnboardResponse{Error: err.Error()})
	}

	resp := OnboardResponse{}

	var request OnboardRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	config, err := n.onboarding.Onboard(context.Background(), request.Config)
	if err != nil {
		handleErr(err)
		return
	}

	resp.Config = config
	resp.OK = true
	n.sendReply(msg, resp)
}

type OffboardRequest struct{}

type OffboardResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (n *Node) handleOffboard(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("offboard_error",
			"labels", string(observability.LabelNode),
			"error", err)
		n.sendReply(msg, OffboardResponse{Error: err.Error()})
	}

	resp := OffboardResponse{}
	if err := n.onboarding.Offboard(context.Background()); err != nil {
		handleErr(err)
		return
	}

	resp.Success = true
	n.sendReply(msg, resp)
}

type OnboardStatusResponse struct {
	Onboarded bool `json:"onboarded"`
}

func (n *Node) handleOnboardStatus(msg actor.Envelope) {
	defer msg.Discard()

	resp := OnboardStatusResponse{}

	resp.Onboarded = n.onboarding.IsOnboarded()
	n.sendReply(msg, resp)
}
