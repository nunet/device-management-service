// Copyright 2025, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"encoding/json"
	"fmt"

	"gitlab.com/nunet/device-management-service/actor"
)

type CreateVolumeRequest struct {
	Name      string `json:"name"`
	ClientPEM string `json:"client_pem"`
}

type CreateVolumeResponse struct {
	OK     bool
	CAData string `json:"ca_data"`
	Error  string `json:"error,omitempty"`
}

func (n *Node) handleCreateVolume(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error creating volume: %s", err)
		n.sendReply(msg, CreateVolumeResponse{Error: err.Error()})
	}

	var request CreateVolumeRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	certificateAuthority, err := n.volumeController.CreateVolume(request.Name, request.ClientPEM)
	if err != nil {
		handleErr(err)
		return
	}

	n.lock.Lock()
	n.volumeOwners[request.Name] = msg.From.DID.String()
	n.lock.Unlock()

	resp := CreateVolumeResponse{
		OK:     true,
		CAData: certificateAuthority,
	}
	n.sendReply(msg, resp)
}

type DeleteVolumeRequest struct {
	Name string `json:"name"`
}

type DeleteVolumeResponse struct {
	OK    bool
	Error string `json:"error,omitempty"`
}

func (n *Node) handleDeleteVolume(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error deleting volume: %s", err)
		n.sendReply(msg, DeleteVolumeResponse{Error: err.Error()})
	}

	var request DeleteVolumeRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	owner, exists := n.volumeOwners[request.Name]
	if !exists {
		handleErr(fmt.Errorf("volume does not exist or ownership is unknown"))
		return
	}
	if owner != msg.From.DID.String() {
		handleErr(fmt.Errorf("permission denied: only the creator can delete the volume"))
		return
	}

	err := n.volumeController.DeleteVolume(request.Name)
	if err != nil {
		handleErr(err)
		return
	}

	n.lock.Lock()
	delete(n.volumeOwners, request.Name)
	n.lock.Unlock()

	resp := DeleteVolumeResponse{OK: true}
	n.sendReply(msg, resp)
}

type StartVolumeRequest struct {
	Name string `json:"name"`
}

type StartVolumeResponse struct {
	OK    bool
	Error string `json:"error,omitempty"`
}

func (n *Node) handleStartVolume(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error starting volume: %s", err)
		n.sendReply(msg, StartVolumeResponse{Error: err.Error()})
	}

	var request StartVolumeRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	resp := StartVolumeResponse{}
	err := n.volumeController.StartVolume(request.Name)
	if err != nil {
		log.Errorf("Error deleting volume: %s", err)
		n.sendReply(msg, StartVolumeResponse{Error: err.Error()})
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}
