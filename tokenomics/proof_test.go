// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build ignore

package tokenomics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ProofInterfaceTestSuite is the test suite for the ProofInterface methods.
type ProofInterfaceTestSuite struct {
	suite.Suite
	proofInterface *ContractProofOperations
}

// SetupTest sets up the test suite by initializing a new ContractProofOperations.
func (s *ProofInterfaceTestSuite) SetupTest() {
	s.proofInterface = &ContractProofOperations{
		ContractDatabase: make(map[string]string),
	}
}

// TestInitiateContractApproval tests the InitiateContractApproval method.
func (s *ProofInterfaceTestSuite) TestInitiateContractApproval() {
	err := s.proofInterface.InitiateContractApproval()
	require.NoError(s.T(), err, "InitiateContractApproval should not return an error")
}

// TestCreateContractProof tests the CreateContractProof method.
func (s *ProofInterfaceTestSuite) TestCreateContractProof() {
	proof, err := s.proofInterface.CreateContractProof()
	require.NoError(s.T(), err, "CreateContractProof should not return an error")
	require.NotEmpty(s.T(), proof, "CreateContractProof should return a non-empty proof")
}

// TestSaveProof tests the SaveProof method.
func (s *ProofInterfaceTestSuite) TestSaveProof() {
	proof := "testProof"
	contractID := "testContractID"
	err := s.proofInterface.SaveProof(contractID, proof)
	require.NoError(s.T(), err, "SaveProof should not return an error")

	// Verify that the proof is saved correctly
	savedProof, exists := s.proofInterface.ContractDatabase[contractID]
	require.True(s.T(), exists, "Proof should be saved in the ContractDatabase")
	require.Equal(s.T(), proof, savedProof, "Saved proof should match the original proof")
}

// TestVerifyProof tests the VerifyProof method with various scenarios.
func (s *ProofInterfaceTestSuite) TestVerifyProof() {
	proof := "testProof"
	contractID := "testContractID"

	// Save the proof first
	err := s.proofInterface.SaveProof(contractID, proof)
	require.NoError(s.T(), err, "SaveProof should not return an error")

	// Scenario 1: Verify with correct proof
	isValid, err := s.proofInterface.VerifyProof(contractID, proof)
	require.NoError(s.T(), err, "VerifyProof should not return an error")
	require.True(s.T(), isValid, "VerifyProof should return true for the correct proof")

	// Scenario 2: Verify with incorrect proof
	isValid, err = s.proofInterface.VerifyProof(contractID, "wrongProof")
	require.NoError(s.T(), err, "VerifyProof should not return an error")
	require.False(s.T(), isValid, "VerifyProof should return false for the incorrect proof")

	// Scenario 3: Verify with a non-existent contractID
	isValid, err = s.proofInterface.VerifyProof("nonExistentID", proof)
	require.Error(s.T(), err, "VerifyProof should return an error for a non-existent contractID")
	require.False(s.T(), isValid, "VerifyProof should return false for a non-existent contractID")
}

// TestAuthenticate tests the Authenticate method with various scenarios.
func (s *ProofInterfaceTestSuite) TestAuthenticate() {
	auth := &Authentication{
		encryption: "validEncryptionKey",
		ZKProof:    "validZKProofKey",
		OffChain:   map[string]interface{}{}, // Add relevant data if needed
	}

	// Scenario 1: Token-based encryption authentication
	valid := auth.Authenticate("nodeID", "tokenBasedEncryption", "validToken")
	require.True(s.T(), valid, "Authenticate should return true for valid token-based encryption credentials")

	// Scenario 2: ZKProof authentication
	valid = auth.Authenticate("nodeID", "ZKProof", "validZKProof")
	require.True(s.T(), valid, "Authenticate should return true for valid ZKProof credentials")

	// Scenario 3: OffChainData authentication
	valid = auth.Authenticate("nodeID", "OffChainData", "validOffChainData")
	require.True(s.T(), valid, "Authenticate should return true for valid OffChainData credentials")

	// Scenario 4: Invalid method
	valid = auth.Authenticate("nodeID", "invalidMethod", "someCredentials")
	require.False(s.T(), valid, "Authenticate should return false for invalid authentication method")

	// Scenario 5: Token-based encryption with invalid credentials
	valid = auth.Authenticate("nodeID", "tokenBasedEncryption", "invalidToken")
	require.False(s.T(), valid, "Authenticate should return false for invalid token-based encryption credentials")

	// Scenario 6: ZKProof with invalid credentials
	valid = auth.Authenticate("nodeID", "ZKProof", "invalidZKProof")
	require.False(s.T(), valid, "Authenticate should return false for invalid ZKProof credentials")

	// Scenario 7: OffChainData with invalid credentials
	valid = auth.Authenticate("nodeID", "OffChainData", "invalidOffChainData")
	require.False(s.T(), valid, "Authenticate should return false for invalid OffChainData credentials")
}

// TestProofInterfaceTestSuite runs the test suite for the ProofInterface.
func TestProofInterfaceTestSuite(t *testing.T) {
	suite.Run(t, new(ProofInterfaceTestSuite))
}
