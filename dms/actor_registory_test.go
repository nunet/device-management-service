package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActorRegistry_AddActorAddress(t *testing.T) {
	registry := NewActorRegistry()
	actor1 := &ActorAddrInfo{InboxAddress: "actor1", HostID: "123"}
	registry.AddActorAddress(actor1)
	_, ok := registry.actors["actor1"]
	assert.True(t, ok, "addActorAddress() should add actor1 to the registry")
}

func TestActorRegistry_SetParentAddress(t *testing.T) {
	registry := NewActorRegistry()

	actor1 := &ActorAddrInfo{InboxAddress: "actor1", HostID: "123"}
	actor2 := &ActorAddrInfo{InboxAddress: "actor2", HostID: "234"}

	registry.AddActorAddress(actor1)
	registry.AddActorAddress(actor2)
	registry.SetParentAddress("actor1", actor2)
	actorInfo, ok := registry.actors["actor1"]
	assert.True(t, ok, "actor1 should exist in the registry")
	assert.Equal(t, actor2, actorInfo.parent, "setParentAddress() should set actor2 as the parent of actor1")
}

func TestActorRegistry_AddChild(t *testing.T) {
	registry := NewActorRegistry()

	actor1 := &ActorAddrInfo{InboxAddress: "actor1", HostID: "123"}
	actor2 := &ActorAddrInfo{InboxAddress: "actor2", HostID: "234"}

	registry.AddActorAddress(actor1)
	registry.AddActorAddress(actor2)

	registry.AddChild("actor1", actor2)

	actorInfo, ok := registry.actors["actor1"]
	assert.True(t, ok, "actor1 should exist in the registry")
	assert.Len(t, actorInfo.childs, 1, "actor1 should have one child")
	assert.Equal(t, actor2, actorInfo.childs[0], "addChild() should add actor2 as a child of actor1")
}

func TestActorRegistry_GetActorAddress(t *testing.T) {
	registry := NewActorRegistry()

	actor1 := &ActorAddrInfo{InboxAddress: "actor1"}

	registry.AddActorAddress(actor1)

	addrInfo, ok := registry.GetActorAddress("actor1")
	assert.True(t, ok, "getActorAddress() should return true for an existing actor address")
	assert.Equal(t, actor1, addrInfo, "getActorAddress() should return the correct ActorAddrInfo for actor1")
}
