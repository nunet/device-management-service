package actor

type NewDeploymentRequestCmd struct {
	Config string
}

type UpdateDeploymentRequestCmd struct {
	NewDeploymentRequestCmd
	EnsembleID string
}

type CapAnchorRequestCmd struct {
	Token string
}

type CreateVolumeRequestCmd struct {
	ClientPEMFile string
	VolumeName    string
	CAOutputDir   string
}

type CreateContractRequestCmd struct {
	ContractFile string
}

type ContractStatusRequestCmd struct {
	ContractDID  string
	ContractHost string
}

type ContractApproveLocalRequestCmd struct {
	ContractDID string
}

type ContractConfirmLocalTransactionCmd struct {
	UniqueID   string
	TxHash     string
	Blockchain string
	QuoteID    string
}

type ContractListLocalTransactionsCmd struct {
	Metadata            map[string]string `json:"metadata,omitempty"`
	Status              []string          `json:"status,omitempty"`
	ContractDID         string            `json:"contract_did,omitempty"`
	PaymentValidatorDID string            `json:"payment_validator_did,omitempty"`
	UniqueID            string            `json:"unique_id,omitempty"`
	TxHash              string            `json:"tx_hash,omitempty"`
	Blockchain          string            `json:"blockchain,omitempty"`
	FromAddress         string            `json:"from_address,omitempty"`
	ToAddress           string            `json:"to_address,omitempty"`
	Limit               int               `json:"limit,omitempty"`
	Offset              int               `json:"offset,omitempty"`
	SortBy              string            `json:"sort_by,omitempty"`
}

type ContractGetPaymentQuoteCmd struct {
	UniqueID string
}

type ContractValidatePaymentQuoteCmd struct {
	QuoteID string
}

type ContractCancelPaymentQuoteCmd struct {
	QuoteID string
}

type ContractPaymentStatusCmd struct {
	UniqueID string
}

type CollectUsagesAndForwardToPaymentProvidersCmd struct {
	ContractDID string `json:"contract_did,omitempty"`
}

type ContractTerminateCmd struct {
	ContractDID  string
	ContractHost string
}

type ContractCompleteCmd struct {
	ContractDID  string
	ContractHost string
}

type ContractValidateCmd struct {
	ContractDID  string
	ContractHost string
}

type ContractSettleCmd struct {
	ContractDID  string
	ContractHost string
}

type DeploymentListCmd struct {
	Metadata      map[string]string `json:"metadata,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
	Status        []string          `json:"status,omitempty"`
	CreatedAfter  string            `json:"created_after,omitempty"`
	CreatedBefore string            `json:"created_before,omitempty"`
	UpdatedAfter  string            `json:"updated_after,omitempty"`
	UpdatedBefore string            `json:"updated_before,omitempty"`
	SortBy        string            `json:"sort_by,omitempty"`
}
