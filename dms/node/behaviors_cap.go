package node

import (
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

type CapListRequest struct {
	Context string
}

type CapListResponse struct {
	OK      bool
	Error   string
	Roots   []did.DID
	Require ucan.TokenList
	Provide ucan.TokenList
	Revoke  ucan.TokenList
}

func (n *Node) handleCapList(msg actor.Envelope) {
	defer msg.Discard()
	var request CapListRequest
	resp := CapListResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	roots, require, provide, revoke := n.rootCap.ListRoots()
	resp.OK = true
	resp.Roots = roots
	resp.Require = require
	resp.Provide = provide
	resp.Revoke = revoke
	n.sendReply(msg, resp)
}

type CapAnchorRequest struct {
	Root    []did.DID
	Require ucan.TokenList
	Provide ucan.TokenList
	Revoke  ucan.TokenList
}

type CapAnchorResponse struct {
	OK    bool
	Error string
}

func (n *Node) handleCapAnchor(msg actor.Envelope) {
	defer msg.Discard()
	var request CapAnchorRequest
	resp := CapAnchorResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if err := n.rootCap.AddRoots(nil, request.Require, request.Provide, request.Revoke); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if err := SaveCapabilityContext(n.rootCap, &n.dmsConfig); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}
