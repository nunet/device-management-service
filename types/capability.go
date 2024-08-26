package types

// note: this data type may be moved to dms or jobs package in the future

type Capability struct {
	Executors    Executors          `json:"executor" description:"Executor type required for the job (docker, vm, wasm, or others)"`
	JobTypes     JobTypes           `json:"type" description:"Details about type of the job (One time, batch, recurring, long running). Refer to dms.jobs package for jobType data model"`
	Resources    ExecutionResources `json:"resources" description:"Resources required for the job"`
	Libraries    []Library          `json:"libraries" description:"Libraries required for the job"`
	Localities   []Locality         `json:"locality" description:"Preferred localities of the machine for execution"`
	Storage      []Storage          `json:"storage" description:"Preferred storage options that the machine should have"`
	Connectivity Connectivity       `json:"connectivity" description:"Network configuration required"`
	Price        []PriceInformation `json:"price" description:"Pricing information"`
	Time         TimeInformation    `json:"time" description:"Time constraints"`
	KYCs         []KYC
}

type Connectivity struct {
	Ports []int `json:"ports" description:"Ports that need to be open for the job to run"`
	VPN   bool  `json:"vpn" description:"Whether VPN is required"`
}

type PriceInformation struct {
	Currency        string `json:"currency" description:"Currency used for pricing"`
	CurrencyPerHour int    `json:"currency_per_hour" description:"Price charged per hour"`
	TotalPerJob     int    `json:"total_per_job" description:"Maximum total price or budget of the job"`
	Preference      int    `json:"preference" description:"Pricing preference"`
}

type TimeInformation struct {
	Units      string `json:"units" description:"Time units"`
	MaxTime    int    `json:"max_time" description:"Maximum time that job should run"`
	Preference int    `json:"preference" description:"Time preference"`
}

type Library struct {
	Name       string `json:"name" description:"Name of the library"`
	Constraint string `json:"constraint" description:"Constraint of the library"`
	Version    string `json:"version" description:"Version of the library"`
}

type Locality struct {
	Kind string `json:"kind" description:"Kind of the region (geographic, nunet-defined, etc)"`
	Name string `json:"name" description:"Name of the region"`
}

type Storage struct {
	Type   StorageType `json:"type" description:"Type of storage"`
	Size   int         `json:"size" description:"Size of storage"`
	Amount int         `json:"amount" description:"Amount of storage"`
}

type StorageType string

const (
	//nolint
	SSD_STORAGE_TYPE StorageType = "ssd"
	//nolint
	HDD_STORAGE_TYPE StorageType = "hdd"
)

type KYC struct {
	Type string `json:"type" description:"Type of KYC"`
	Data string `json:"data" description:"Data required for KYC"`
}

type JobTypes []JobType

type JobType string

const (
	BATCH       JobType = "batch"
	SINGLERUN   JobType = "single_run"
	RECURRING   JobType = "recurring"
	LONGRUNNING JobType = "long_running"
)

type Executors []Executor
