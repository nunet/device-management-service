package types

// ContractConfig represents a contract between parties
type ContractConfig struct {
	DID  string `json:"did"` // DID of the contract
	Host string `json:"host"`
}
