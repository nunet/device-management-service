// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func makeSecurityContext(t *testing.T) actor.SecurityContext {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err, "generate key")
	ctxDID, trustCtx := actor.MakeRootTrustContext(t)
	capCtx, err := ucan.NewCapabilityContext(trustCtx, ctxDID, nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	require.NoError(t, err, "create capability context")
	sctx, err := actor.NewBasicSecurityContext(pub, priv, capCtx)
	require.NoError(t, err, "create security context")
	return sctx
}

type handlerFunc func(req *http.Request) (*http.Response, error)

func (h handlerFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return h(req)
}

func withActorHandleResponse(t *testing.T, handle actor.Handle, handler handlerFunc) handlerFunc {
	return func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, client.ActorHandleEndpoint) {
			// return an http response with a handle
			body, err := json.Marshal(handle)
			assert.NoError(t, err, "marshal handle")
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}
		return handler(req)
	}
}

func makeMockClientConfig() client.Config {
	return client.Config{
		Host:      "localhost",
		Protocol:  client.ConnectionTCP,
		APIPrefix: "",
		Version:   "",
	}
}

func makeMockBehaviorClient(t *testing.T, expectedPath string, handler func(t *testing.T, env *actor.Envelope) (int, any)) (*client.Client, actor.SecurityContext, error) {
	cfg := makeMockClientConfig()
	sctx := makeSecurityContext(t)
	dmsSctx := makeSecurityContext(t)
	dmsHandle, err := actor.HandleFromDID(dmsSctx.DID().String())
	require.NoError(t, err)
	client, err := client.NewClientWithTransport(
		cfg,
		withActorHandleResponse(t, dmsHandle, func(req *http.Request) (*http.Response, error) {
			// validate HTTP request
			assert.Equal(t, expectedPath, req.URL.Path, "unexpected request path")
			assert.Equal(t, http.MethodPost, req.Method, "wrong http method")

			// validate actor envelope
			var env actor.Envelope
			err := json.NewDecoder(req.Body).Decode(&env)
			assert.NoError(t, err)

			if env.Expired() {
				return &http.Response{
					StatusCode: http.StatusRequestTimeout,
					Body: io.NopCloser(bytes.NewReader(
						[]byte(`{"error": "request timed out"}`),
					)),
				}, nil
			}

			// handle request
			statusCode, resp := handler(t, &env)
			if statusCode < 200 || statusCode >= 300 {
				body, err := json.Marshal(resp)
				assert.NoError(t, err)
				return &http.Response{
					StatusCode: statusCode,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}

			var body []byte
			switch {
			case env.IsBroadcast():
				var messages []actor.Envelope
				if resp != nil {
					resps, ok := resp.([]any)
					assert.True(t, ok, "broadcast response not an array")
					for _, r := range resps {
						_, pub, err := crypto.GenerateKeyPair(crypto.Ed25519)
						assert.NoError(t, err, "generate random actor DID")
						fromDID := did.FromPublicKey(pub)
						handle, err := actor.HandleFromDID(fromDID.String())
						assert.NoError(t, err, "generate random broadcast reply source actor handle")
						msg, err := actor.ReplyTo(env, r, actor.WithMessageSource(handle))
						assert.NoError(t, err, "create reply")
						messages = append(messages, msg)
					}
				}
				body, err = json.Marshal(messages)
				assert.NoError(t, err, "marshal reply")
			case env.Options.ReplyTo != "":
				msg, err := actor.ReplyTo(env, resp)
				assert.NoError(t, err, "create reply")
				body, err = json.Marshal(msg)
				assert.NoError(t, err, "marshal reply")
			default:
				body = []byte("{\"message\": \"message sent\"}")
			}
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}),
		sctx,
	)
	return client, sctx, err
}
