package tokenomics

import (

	"jobs"
	"orchestrator"
	"payments"
	"proofs"
	"contract"
)


// Contract defines the methods for contract operations
type contract interface {
	NewContract() Contract
	InitiateContractClosure(n1 dms.NodeID, n2 dms.NodeID, bid orchestrator.Bid)
	InitiateContractSettlement(n1 dms.NodeID, n2 dms.NodeID, contractID int, verificationResult orchestrator.JobVerificationResult)
}

// Proof defines the methods for proof-based operations
type proofs interface {
	InitiateContractApproval() error
	CreateContractProof() (string, error)
	SaveProof(contractID, proof string) error
	VerifyProof(contractID, proof string) (bool, error)
}

// Payment defines the operations for managing payments and settlements
type payments interface {
	Deposit(contractID int, payment Payment, pricing PricingMethod) error
	SettleContract(contractID int, verificationResult jobs.JobVerificationResult) error
}


