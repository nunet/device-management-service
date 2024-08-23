package types

import (
	"time"
)

type DeploymentRequest struct {
	RequesterWalletAddress string    `json:"address_user"` // service provider wallet address
	MaxNtx                 int       `json:"max_ntx"`
	Blockchain             string    `json:"blockchain"`
	TxHash                 string    `json:"tx_hash"`
	ServiceType            string    `json:"service_type"`
	Timestamp              time.Time `json:"timestamp"`
	MetadataHash           string    `json:"metadata_hash"`
	WithdrawHash           string    `json:"withdraw_hash"`
	RefundHash             string    `json:"refund_hash"`
	Distribute50Hash       string    `json:"distribute_50_hash"`
	Distribute75Hash       string    `json:"distribute_75_hash"`
	Params                 struct {
		ImageID   string `json:"image_id"`
		ModelURL  string `json:"model_url"`
		ResumeJob struct {
			Resume       bool   `json:"resume"`
			ProgressFile string `json:"progress_file"` // file path
		} `json:"resume_job"`
		Packages        []string `json:"packages"`
		RemoteNodeID    string   `json:"node_id"`          // NodeID of compute provider (machine to deploy the job on)
		RemotePublicKey string   `json:"public_key"`       // Public key of compute provider
		LocalNodeID     string   `json:"local_node_id"`    // NodeID of service provider (machine triggering the job)
		LocalPublicKey  string   `json:"local_public_key"` // Public key of service provider
		MachineType     string   `json:"machine_type"`
	} `json:"params"`
	Constraints struct {
		Complexity string `json:"complexity"`
		CPU        int    `json:"cpu"`
		RAM        int    `json:"ram"`
		Vram       int    `json:"vram"`
		Power      int    `json:"power"`
		Time       int    `json:"time"`
	} `json:"constraints"`
	TraceInfo struct {
		TraceID     string `json:"trace_id"`
		SpanID      string `json:"span_id"`
		TraceFlags  string `json:"trace_flags"`
		TraceStates string `json:"trace_state"`
	} `json:"traceinfo"`
}

type DeploymentResponse struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
}

type DeploymentUpdate struct {
	MsgType string `json:"msg_type"`
	Msg     string `json:"msg"`
}

type DeploymentRequestFlat struct {
	BaseDBModel
	DeploymentRequest string `json:"deployment_request"`
	// represents job status from services table; goal is to keep then in sync (both tables are on different DMSes).
	JobStatus string `json:"job_status"`
}

type BlockchainTxStatus struct {
	TransactionType   string `json:"transaction_type"` // No need of this param maybe be deprecated in future
	TransactionStatus string `json:"transaction_status"`
	TxHash            string `json:"tx_hash"`
}
