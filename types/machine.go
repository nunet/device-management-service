package types

import (
	"time"

	"gorm.io/gorm"
)

type IP []any

type PeerInfo struct {
	BaseDBModel
	NodeID    string `json:"nodeID,omitempty"`
	Key       string `json:"key,omitempty"`
	Mid       string `json:"mid,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	Address   string `json:"_address,omitempty"`
}

type Machine struct {
	BaseDBModel
	NodeId               string
	PeerInfo             int
	IpAddr               string
	AvailableResources   int
	FreeResources        int
	TokenomicsAddress    string
	TokenomicsBlockchain string
}

// FreeResources are the resources free to be used by new services,
// plugins and any other processes started by DMS. It's basically
// the subtraction between AvailableResources and the amount of resources
// already used by DMS and its processes (mostly services)
type FreeResources struct {
	BaseDBModel
	TotCpuHz          int     `json:"tot_cpu_hz"`
	PriceCpu          float64 `json:"price_cpu"`
	Ram               int     `json:"ram"`
	PriceRam          float64 `json:"price_ram"`
	Vcpu              int     `json:"vcpu"`
	Disk              float64 `json:"disk"`
	PriceDisk         float64 `json:"price_disk"`
	NTXPricePerMinute float64 `json:"ntx_price"`
}

// AvailableResources are the amount of resources onboarded which
// can be used by NuNet
type AvailableResources struct {
	BaseDBModel
	TotCpuHz          int
	CpuNo             int
	CpuHz             float64
	PriceCpu          float64
	Ram               int
	PriceRam          float64
	Vcpu              int
	Disk              float64
	PriceDisk         float64
	NTXPricePerMinute float64
}

type Services struct {
	BaseDBModel
	TxHash               string
	TransactionType      string // transaction type can be running, done, withdraw, refund and distribute
	JobStatus            string // whether job is running or exited; one of these 'running', 'finished without errors', 'finished with errors'
	JobDuration          int64  // job duration in minutes
	EstimatedJobDuration int64  // job duration in minutes
	ServiceName          string
	ContainerID          string
	ResourceRequirements int // ID of ServiceResourceRequirements record
	ImageID              string
	LogURL               string
	LastLogFetch         time.Time
	ServiceProviderAddr  string
	ComputeProviderAddr  string
	MetadataHash         string
	WithdrawHash         string
	RefundHash           string // saving hashes for call the `/request-reward` endpoint by SPD
	Distribute_50Hash    string
	Distribute_75Hash    string
	SignatureDatum       string
	MessageHashDatum     string
	Datum                string
	SignatureAction      string // saving signatures for removing redundancy of calling Oracle
	MessageHashAction    string
	Action               string
	// TODO: Add ContainerType field

}

type ServiceResourceRequirements struct {
	BaseDBModel
	CPU  int
	RAM  int
	VCPU int
	HDD  int
}

type ContainerImages struct {
	gorm.Model
	ImageID   string
	ImageName string
	Digest    string
}

type Libp2pInfo struct {
	BaseDBModel
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
	ServerMode bool   `json:"server_mode"`
	Available  bool   `json:"available"`
}

type MachineUUID struct {
	BaseDBModel
	UUID string `json:"uuid"`
}

type resources struct {
	TotCpuHz  float64
	PriceCpu  float64
	Ram       int
	PriceRam  float64
	Vcpu      int
	Disk      float64
	PriceDisk float64
}

type PeerData struct {
	PeerID               string        `json:"peer_id"`
	IsAvailable          bool          `json:"is_available"`
	HasGpu               bool          `json:"has_gpu"`
	AllowCardano         bool          `json:"allow_cardano"`
	GpuInfo              []GPU         `json:"gpu_info"`
	TokenomicsAddress    string        `json:"tokenomics_addrs"`
	TokenomicsBlockchain string        `json:"tokenomics_blockchain"`
	AvailableResources   FreeResources `json:"available_resources"`
	Services             []Services    `json:"services"`
	Timestamp            int64         `json:"timestamp,omitempty"`
}

type Connection struct {
	BaseDBModel
	PeerID     string `json:"peer_id"`
	Multiaddrs string `json:"multiaddrs"`
}

type PingResult struct {
	RTT     time.Duration
	Success bool
	Error   error
}

type Machines map[string]PeerData

// machine info for KAD-DHT
type KadDHTMachineUpdate struct {
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}

type ElasticToken struct {
	BaseDBModel
	NodeId      string
	Token       string
	ChannelName string
}
