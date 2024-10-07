package jobs

import (
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
)

type EnsembleManifest struct {
	ID           string                        // ensemble globally unique id
	Orchestrator actor.Handle                  // orchestrator actor
	Allocations  map[string]AllocationManifest // allocation name -> manifest
	Nodes        map[string]NodeManifest       // node name -> manifest
}

type AllocationManifest struct {
	ID       string       // allocation unique id
	NodeID   string       // allocation node
	Handle   actor.Handle // handle of the allocation control actor
	DNSName  string       // (internal) DNS name of the allocation
	PrivAddr string       // (VPN) private IP address of the allocation peer
	Ports    map[int]int  // port mapping, public -> private
}

type NodeManifest struct {
	ID          string       // node unique id
	Peer        string       // peer where the node is running
	Handle      actor.Handle // handle of the control actor for the node
	PubAddrss   []string     // public IP4/6 address of the node peer
	Location    Location     // location of the peer
	Allocations []string     // allocations in the nod
}

func (mf *EnsembleManifest) Clone() EnsembleManifest {
	var clone EnsembleManifest

	bytes, err := json.Marshal(mf)
	if err != nil {
		log.Errorf("error marshaling ensemble manifest: %s", err)
		return clone
	}

	if err := json.Unmarshal(bytes, &clone); err != nil {
		log.Errorf("error unmarshaling ensemble manifest: %s", err)
	}

	return clone
}
