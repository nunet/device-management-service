// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

// Should use constants for authentication methods
const (
	TokenBasedEncryption = "tokenBasedEncryption"
	ZKProofAuth          = "ZKProof"
	OffChainDataAuth     = "OffChainData"
)

// GeneralAuthentication contains general authentication methods
type Authentication struct {
	Encryption string
	ZKProof    string
	OffChain   map[string]interface{}
}

// ContractProofOperations implements the ProofInterface
type ContractProofOperations struct {
	GeneralAuth      *Authentication
	ContractDatabase map[string]string // Simulates a database of contract proofs
}
