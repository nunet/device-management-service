// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package api

import (
	"fmt"
	"net/http"
	"testing"

	"gitlab.com/nunet/device-management-service/network"

	"github.com/gin-gonic/gin"

	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/actor"

	"github.com/stretchr/testify/require"
)

func TestServer_Actor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serverPort := uint32(9138)

	ctrl := gomock.NewController(t)
	p2pHandler := NewMockNetwork(ctrl)

	_ = startServer(t, serverPort, p2pHandler)

	t.Run("must be able to call /api/v1/actor/handle", func(t *testing.T) {
		p2pHandler.EXPECT().GetHostID().Return(network.PeerID("hostID")).Times(1)
		p2pHandler.EXPECT().GetPeerPubKey(gomock.Any()).Return(getTestPublicKey(t)).Times(1)
		p2pHandler.EXPECT().GetHostID().Return(network.PeerID("hostID")).Times(1)

		resp, err := doRequest(t, http.MethodGet,
			fmt.Sprintf("http://localhost:%d/api/v1/actor/handle", serverPort),
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("must be able to call /api/v1/actor/send", func(t *testing.T) {
		p2pHandler.EXPECT().SendMessageSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		msg := actor.Envelope{}
		resp, err := doRequest(t, http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/actor/send", serverPort),
			getMockEnvelope(t, msg).Bytes(),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("must be able to call /api/v1/actor/invoke", func(t *testing.T) {
		var msg actor.Envelope
		p2pHandler.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(_ string, handler func(data []byte)) error {
			// call the handler
			handler(getMockEnvelope(t, msg).Bytes())
			return nil
		}).Times(1)
		p2pHandler.EXPECT().SendMessageSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		p2pHandler.EXPECT().UnregisterMessageHandler(gomock.Any()).Times(1)

		resp, err := doRequest(t, http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/actor/invoke", serverPort),
			getMockEnvelope(t, msg).Bytes(),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("must be able to call /api/v1/actor/broadcast", func(t *testing.T) {
		p2pHandler.EXPECT().HandleMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		p2pHandler.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		p2pHandler.EXPECT().UnregisterMessageHandler(gomock.Any()).Times(1)

		resp, err := doRequest(t, http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/actor/broadcast", serverPort),
			getMockBroadcastMessage(t).Bytes(),
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
