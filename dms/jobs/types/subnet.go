// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

type SubnetManifest struct {
	CIDR              string            `json:"cidr"`
	GatewayIP         string            `json:"gateway_ip"`
	BroadcastIP       string            `json:"broadcast_ip"`
	UsedIPs           map[string]bool   `json:"used_ips"`
	RoutingTable      map[string]string `json:"routing_table"` // ip -> peerID
	IndexRoutingTable map[string]string `json:"index_routing_table"`
	DNSRecords        map[string]string `json:"dns_records"`
}
