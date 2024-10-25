// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type SubnetRoutingTable interface {
	Add(peerID peer.ID, addr string)
	Remove(peerID peer.ID)
	Get(peerID peer.ID) (string, bool)
	RemoveByIP(addr string)
	GetByIP(addr string) (peer.ID, bool)
	All() map[peer.ID]string
	Clear()
}

type rtable struct {
	mx     sync.RWMutex
	idx    map[peer.ID]string
	revIdx map[string]peer.ID
}

func NewRoutingTable() SubnetRoutingTable {
	return &rtable{
		idx:    make(map[peer.ID]string),
		revIdx: make(map[string]peer.ID),
	}
}

func (rt *rtable) Add(peerID peer.ID, addr string) {
	rt.mx.Lock()
	defer rt.mx.Unlock()

	rt.idx[peerID] = addr
	rt.revIdx[addr] = peerID
}

func (rt *rtable) Remove(peerID peer.ID) {
	rt.mx.Lock()
	defer rt.mx.Unlock()

	addr, ok := rt.idx[peerID]
	if !ok {
		return
	}

	delete(rt.idx, peerID)
	delete(rt.revIdx, addr)
}

func (rt *rtable) Get(peerID peer.ID) (string, bool) {
	rt.mx.RLock()
	defer rt.mx.RUnlock()

	addr, ok := rt.idx[peerID]
	return addr, ok
}

func (rt *rtable) RemoveByIP(addr string) {
	rt.mx.Lock()
	defer rt.mx.Unlock()

	peerID, ok := rt.revIdx[addr]
	if !ok {
		return
	}

	delete(rt.idx, peerID)
	delete(rt.revIdx, addr)
}

func (rt *rtable) GetByIP(addr string) (peer.ID, bool) {
	rt.mx.RLock()
	defer rt.mx.RUnlock()

	peerID, ok := rt.revIdx[addr]
	return peerID, ok
}

func (rt *rtable) All() map[peer.ID]string {
	rt.mx.RLock()
	defer rt.mx.RUnlock()

	idx := make(map[peer.ID]string)
	for k, v := range rt.idx {
		idx[k] = v
	}
	return idx
}

func (rt *rtable) Clear() {
	rt.mx.Lock()
	defer rt.mx.Unlock()

	rt.idx = make(map[peer.ID]string)
	rt.revIdx = make(map[string]peer.ID)
}
