// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

// ContractConfig represents a contract between parties
type ContractConfig struct {
	DID       string `json:"did"`                 // Contract DID (required)
	Host      string `json:"host"`                // Contract host DID (required)
	Provider  string `json:"provider,omitempty"`  // Provider DID (optional, for chain detection)
	Requestor string `json:"requestor,omitempty"` // Requestor DID (optional, for chain detection)
}

type PaymentAddressInfo struct {
	Blockchain    string `json:"blockchain"` // ETHEREUM, CARDANO etc..
	Currency      string `json:"currency"`
	RequesterAddr string `json:"requester_addr"`
	ProviderAddr  string `json:"provider_addr"`
}
