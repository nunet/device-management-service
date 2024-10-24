// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

// import (
// 	"testing"

// 	"github.com/multiformats/go-multiaddr"
// 	"github.com/stretchr/testify/assert"
// 	"gitlab.com/nunet/device-management-service/actor"
// 	"gitlab.com/nunet/device-management-service/dms/onboarding"
// 	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
// 	gomock "go.uber.org/mock/gomock"
// )

// func TestBehaviours(t *testing.T) {
// 	rootCap := createRootCapabilityContext(t)
// 	rootCap2 := createRootCapabilityContext(t)
// 	net := createNetwork(t, []multiaddr.Multiaddr{}, "13957")
// 	peer1p2pAddrs, err := net.GetMultiaddr()
// 	assert.NoError(t, err)
// 	net2 := createNetwork(t, peer1p2pAddrs, "13937")

// 	ctrl := gomock.NewController(t)
// 	t.Cleanup(ctrl.Finish)
// 	resourceManager := NewMockResourceManager(ctrl)

// 	node1, err := New(&onboarding.Onboarding{}, rootCap, net.Host.ID().String(), net, resourceManager, bt.NewScheduler(1))
// 	assert.NoError(t, err)
// 	assert.NotNil(t, node1)
// 	err = node1.Start()
// 	assert.NoError(t, err)
// 	node2, err := New(&onboarding.Onboarding{}, rootCap2, net2.Host.ID().String(), net2, resourceManager, bt.NewScheduler(1))
// 	assert.NoError(t, err)
// 	err = node2.Start()
// 	assert.NoError(t, err)

// 	type payload struct{ name string}

// 	msg, err := actor.Message(
// 		node1.actor.Handle(),
// 		node2.actor.Handle(),
// 		"/test/ping",
// 		payload{name: "random name"},
// 		actor.WithMessageTopic()
// 	)

// }
