package tokenomics

import (
	"dms"
	"orchestrator"
	"payments."
)

// Contract represents the contract details between nodes
type Contract struct {
	ContractID     int
	JobID          int
	Requestor      string 
	Provider       string
	PaymentDetails payments.Payment
	Signatures     []dms.nodeID
	Settled        bool
	Verification   orchestrator.JobVerificationResult
	ContractProof  orchestrator.ContractProof

}

// NewContract creates a new contract
func NewContract() Contract {
	return Contract{
		Signatures:    []dms.nodeID{},
	}
}


func initiateContractClosure(n1 dms.nodeID, n2 dms.nodeID, bid orchestrator.Bid) {
	var contract Contract
	contract = NewContract()

	contract.JobID = bid.JobID
	contract.PaymentDetails = new payments.Payment(bid.PriceBid.amount, bid.PriceBid.currency, ...) 
	// all the details for contract struct need to be updated here
	

	// Sign and notarize the contract
	contract.signContract(n1)
	contract.askForSignature(n1, n2)

	// Save the contract in the respective nodes' contract lists and the central database
	n1.Contracts = append(n1.Contracts, contract)
	n2.Contracts = append(n2.Contracts, contract)
	saveContractToDatabase(contract)

	contract.notarize()
}

func initiateContractSettlement(n1 dms.nodeID, n2 dms.nodeID, contract ContractID, verificationResult orchestrator.JobVerificationResult) {
	// Implement the logic for contract settlement as per the sequence diagram
	contract.Verification = verificationResult
	contract.settle()
	contract.markAsSettled()

	// Notify both nodes about the settlement
	n1.notifyContractSettlement(contract)
	n2.notifyContractSettlement(contract)

	// Update the contract in the central database
	updateContractInDatabase(contract)
}