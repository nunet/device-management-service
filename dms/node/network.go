// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/observability"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
)

func includesRootProtocol(protocols []protocol.ID) bool {
	for _, proto := range protocols {
		if proto == rootProto {
			return true
		}
	}

	return false
}

func (n *Node) subscribe(topics ...string) error {
	for _, topic := range topics {
		if err := n.actor.Subscribe(topic, n.setupBroadcast); err != nil {
			return fmt.Errorf("error subscribing to %s: %w", topic, err)
		}
	}

	n.network.SetBroadcastAppScore(n.broadcastScore)
	if err := n.network.Notify(n.actor.Context(),
		n.peerPreConnected,
		n.peerConnected,
		n.peerDisconnected,
		n.peerIdentified,
		n.peerIdentified,
	); err != nil {
		return fmt.Errorf("setting up peer notifications: %w", err)
	}

	return nil
}

func (n *Node) setupBroadcast(topic string) error {
	return n.network.SetupBroadcastTopic(topic, func(t *network.Topic) error {
		return t.SetScoreParams(&pubsub.TopicScoreParams{
			SkipAtomicValidation:           true,
			TopicWeight:                    1.0,
			TimeInMeshWeight:               0.00027, // ~1/3600
			TimeInMeshQuantum:              time.Second,
			TimeInMeshCap:                  1.0,
			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		})
	})
}

func (n *Node) broadcastScore(p peer.ID) float64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	st, ok := n.peers[p]
	if !ok {
		return 0
	}

	if st.helloIn && st.helloOut {
		return 5
	}

	if st.hasRoot {
		return 1
	}

	return 0
}

func (n *Node) peerConnected(p peer.ID) {
	logConn.Debugf("peer connected: %s", p)
	n.lock.Lock()
	defer n.lock.Unlock()

	st, ok := n.peers[p]
	if !ok {
		st = &peerState{}
		n.peers[p] = st
	}
	st.numConnections++
}

func (n *Node) peerPreConnected(p peer.ID, protocols []protocol.ID, numConnections int) {
	logConn.Debugf("peer preconnected: %s %s (%d)", p, protocols, numConnections)
	n.lock.Lock()
	defer n.lock.Unlock()

	st := &peerState{numConnections: numConnections}
	n.peers[p] = st

	if includesRootProtocol(protocols) {
		st.hasRoot = true
		st.helloPending = true
		st.helloAttempts = 1
		go n.sayHello(p)
	}
}

func (n *Node) peerIdentified(p peer.ID, protocols []protocol.ID) {
	logConn.Debugf("peer identified: %s %s", p, protocols)
	n.lock.Lock()
	defer n.lock.Unlock()

	st, ok := n.peers[p]
	if !ok {
		st = &peerState{}
		n.peers[p] = st
	}

	if includesRootProtocol(protocols) {
		st.hasRoot = true
		if !st.helloOut && !st.helloPending {
			st.helloPending = true
			st.helloAttempts++
			go n.sayHello(p)
		}
	}
}

func (n *Node) peerDisconnected(p peer.ID) {
	logConn.Debugf("peer disconnected: %s", p)
	n.lock.Lock()
	defer n.lock.Unlock()

	st, ok := n.peers[p]
	if !ok {
		return
	}
	st.numConnections--

	if st.numConnections <= 0 {
		delete(n.peers, p)
	}
}

func (n *Node) sayHello(p peer.ID) {
	pubk, err := p.ExtractPublicKey()
	if err != nil {
		log.Debugf("extract public key: %s", err)
		return
	}

	if !crypto.AllowedKey(int(pubk.Type())) {
		log.Debugf("unexpected key type: %d", pubk.Type())
		return
	}

	actorID, err := crypto.IDFromPublicKey(pubk)
	if err != nil {
		log.Debugf("extract actor ID: %s", err)
		return
	}

	actorDID := did.FromPublicKey(pubk)
	handle := actor.Handle{
		ID:  actorID,
		DID: actorDID,
		Address: actor.Address{
			HostID:       p.String(),
			InboxAddress: "root",
		},
	}

	wait := helloMinDelay + time.Duration(rand.Int63n(int64(helloMaxDelay-helloMinDelay)))
	time.Sleep(wait)

	n.lock.Lock()
	st, ok := n.peers[p]
	if !ok {
		n.lock.Unlock()
		return
	}

	if !n.network.PeerConnected(p) {
		st.helloPending = false
		n.lock.Unlock()
		return
	}
	n.lock.Unlock()

	msg, err := actor.Message(
		n.actor.Handle(),
		handle,
		behaviors.PublicHelloBehavior,
		nil,
		actor.WithMessageTimeout(helloTimeout),
	)
	if err != nil {
		log.Debugf("construct hello message: %s", err)
		return
	}

	logConn.Debugf("saying hello to %s", handle.Address.HostID)
	replyCh, err := n.actor.Invoke(msg)
	if err != nil {
		n.lock.Lock()
		if st, ok = n.peers[p]; ok {
			if st.helloAttempts < helloAttempts {
				st.helloAttempts++
				go n.sayHello(p)
			} else {
				st.helloPending = false
			}
		}
		n.lock.Unlock()
		logConn.Debugf("invoking hello: %s", err)
		return
	}

	select {
	case reply := <-replyCh:
		reply.Discard()
		n.lock.Lock()
		if st, ok = n.peers[p]; ok {
			st.helloOut = true
			st.helloPending = false
		} else if n.network.PeerConnected(p) {
			// race with connected notification
			st = &peerState{helloOut: true}
			n.peers[p] = st
		}
		n.lock.Unlock()
		log.Infow("got hello response from", "labels", string(observability.LabelNode), "hostID", handle.Address.HostID)

	case <-time.After(time.Until(msg.Expiry())):
		n.lock.Lock()
		if st, ok = n.peers[p]; ok {
			if st.helloAttempts < helloAttempts {
				st.helloAttempts++
				go n.sayHello(p)
			} else {
				st.helloPending = false
			}
		}
		n.lock.Unlock()
		logConn.Debugw("hello timeout", "labels", string(observability.LabelNode), "hostID", handle.Address.HostID)
	}
}
