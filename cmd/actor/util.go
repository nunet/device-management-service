// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/cmd/cap"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/utils"
)

type cmdResponse struct {
	val interface{}
}

func (r *cmdResponse) UnmarshalJSON(data []byte) error {
	var res struct {
		Message []byte `json:"msg"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	val := interface{}(nil)
	if err := json.Unmarshal(res.Message, &val); err != nil {
		return err
	}
	*r = cmdResponse{val: val}
	return nil
}

func (r cmdResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.val)
}

func getDMSHandle(client *utils.HTTPClient) (actor.Handle, error) {
	var handle actor.Handle

	body, code, err := client.MakeRequest("GET", "/actor/handle", nil)
	if err != nil {
		return handle, fmt.Errorf("unable to get source handle: %w", err)
	}
	if code != 200 {
		return handle, fmt.Errorf("request failed with status code: %d", code)
	}

	if err = json.Unmarshal(body, &handle); err != nil {
		return handle, fmt.Errorf("could not unmarshal response body: %w", err)
	}
	return handle, err
}

func newUserHandle(id crypto.ID, did did.DID, dmsHandle actor.Handle, inbox string) actor.Handle {
	return actor.Handle{
		ID:  id,
		DID: did,
		Address: actor.Address{
			HostID:       dmsHandle.Address.HostID,
			InboxAddress: inbox,
		},
	}
}

func newSecurityContext(fs afero.Afero, context string) (actor.SecurityContext, error) {
	if context == "" {
		context = DefaultUserContextName
	}

	// Generate ephemeral key pair
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key pair: %w", err)
	}

	// Create trust context
	var trustCtx did.TrustContext
	if cap.IsLedgerContext(context) {
		provider, err := did.NewLedgerWalletProvider(0)
		if err != nil {
			return nil, err
		}

		trustCtx = did.NewTrustContextWithProvider(provider)
		context = cap.LedgerContext(context)
	} else {
		var err error
		trustCtx, _, err = cap.CreateTrustContextFromKeyStore(fs, context)
		if err != nil {
			return nil, fmt.Errorf("failed to create trust context: %w", err)
		}
	}

	// Load capability context
	capCtx, err := cap.LoadCapabilityContext(trustCtx, context)
	if err != nil {
		return nil, fmt.Errorf("failed to load capability context: %w", err)
	}

	return actor.NewBasicSecurityContext(pubk, privk, capCtx)
}

func newActorMessage(fs afero.Afero, dmsHandle actor.Handle, destStr string, topic, behavior string, payload interface{}, timeout time.Duration, expiry time.Time, invocation bool, context string) (actor.Envelope, error) {
	var msg actor.Envelope
	var src actor.Handle
	var dest actor.Handle

	sctx, err := newSecurityContext(fs, context)
	if err != nil {
		return msg, fmt.Errorf("failed to create security context: %w", err)
	}

	nonce := sctx.Nonce()
	inbox := fmt.Sprintf("user-%d", nonce)
	src = newUserHandle(sctx.ID(), sctx.DID(), dmsHandle, inbox)

	opts := []actor.MessageOption{}
	replyTo := ""
	switch {
	case topic != "":
		opts = append(opts, actor.WithMessageTopic(topic))
		replyTo = fmt.Sprintf("/public/user/%d", nonce)
	case destStr != "":
		switch {
		case strings.HasPrefix(destStr, "did:"):
			dest, err = handleFromDID(destStr)
		case strings.HasPrefix(destStr, "{"):
			err = json.Unmarshal([]byte(destStr), &dest)
		default:
			dest, err = handleFromPeerID(destStr)
		}

		if err != nil {
			return msg, fmt.Errorf("could not create destination handle: %w", err)
		}
	default:
		dest = dmsHandle
	}

	if invocation {
		replyTo = fmt.Sprintf("/private/user/%d", nonce)
	}

	if !expiry.IsZero() {
		opts = append(opts, actor.WithMessageExpiry(uint64(expiry.UnixNano())))
	}

	if timeout > 0 {
		opts = append(opts, actor.WithMessageTimeout(timeout))
	}

	delegate := []ucan.Capability{}
	if replyTo != "" {
		opts = append(opts, actor.WithMessageReplyTo(replyTo))
		if topic == "" {
			delegate = append(delegate, ucan.Capability(replyTo))
		}
	}

	opts = append(opts, actor.WithMessageSignature(sctx, []ucan.Capability{ucan.Capability(behavior)}, delegate))

	msg, err = actor.Message(src, dest, behavior, payload, opts...)
	if err != nil {
		return msg, fmt.Errorf("could not construct message: %w", err)
	}

	return msg, nil
}

func handleFromPeerID(dest string) (actor.Handle, error) {
	peerID, err := peer.Decode(dest)
	if err != nil {
		return actor.Handle{}, err
	}

	pubk, err := peerID.ExtractPublicKey()
	if err != nil {
		return actor.Handle{}, err
	}

	if !crypto.AllowedKey(int(pubk.Type())) {
		return actor.Handle{}, fmt.Errorf("unexpected key type: %d", pubk.Type())
	}

	actorID, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		return actor.Handle{}, err
	}

	actorDID := did.FromPublicKey(pubk)
	handle := actor.Handle{
		ID:  actorID,
		DID: actorDID,
		Address: actor.Address{
			HostID:       peerID.String(),
			InboxAddress: "root",
		},
	}

	return handle, nil
}

func handleFromDID(dest string) (actor.Handle, error) {
	actorDID, err := did.FromString(dest)
	if err != nil {
		return actor.Handle{}, err
	}

	pubk, err := did.PublicKeyFromDID(actorDID)
	if err != nil {
		return actor.Handle{}, err
	}

	actorID, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		return actor.Handle{}, err
	}

	peerID, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		return actor.Handle{}, err
	}

	handle := actor.Handle{
		ID:  actorID,
		DID: actorDID,
		Address: actor.Address{
			HostID:       peerID.String(),
			InboxAddress: "root",
		},
	}

	return handle, nil
}
