package types

import (
	"errors"
)

// note: this data type may be moved to dms or jobs package in the future

type CapabilityAdder interface {
	Add(Capability) error
}

type CapabilitySubtractor interface {
	Subtract(Capability) error
}

type CapabilityAddSubtractor interface {
	CapabilityAdder
	CapabilitySubtractor
}

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

var _ CapabilityAddSubtractor = &Capability{}

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

type (
	Executors         []Executor
	Libraries         []Library
	Localities        []Locality
	KYCs              []KYC
	Storages          []Storage
	PricesInformation []PriceInformation
)

func (lib *Library) Equal(library Library) bool {
	if lib.Name == library.Name && lib.Constraint == library.Constraint && lib.Version == library.Version {
		return true
	}
	return false
}

func (loc *Locality) Equal(locality Locality) bool {
	if loc.Kind == locality.Kind && loc.Name == locality.Name {
		return true
	}

	return false
}

func (e *Executor) Equal(executor Executor) bool {
	return e.ExecutorType == executor.ExecutorType
}

func (k *KYC) Equal(kyc KYC) bool {
	if k.Type == kyc.Type && k.Data == kyc.Data {
		return true
	}
	return false
}

func (p *PriceInformation) Equal(price PriceInformation) bool {
	if p.Currency == price.Currency && p.CurrencyPerHour == price.CurrencyPerHour && p.TotalPerJob == price.TotalPerJob && p.Preference == price.Preference {
		return true
	}
	return false
}

func (es Executors) Contains(executor Executor) bool {
	for _, e := range es {
		if e.Equal(executor) {
			return true
		}
	}
	return false
}

func (j JobTypes) Contains(jobType JobType) bool {
	for _, j := range j {
		if j == jobType {
			return true
		}
	}
	return false
}

func (l Libraries) Contains(library Library) bool {
	for _, lib := range l {
		if lib.Equal(library) {
			return true
		}
	}
	return false
}

func (l Localities) Contains(locality Locality) bool {
	for _, loc := range l {
		if loc.Equal(locality) {
			return true
		}
	}
	return false
}

func (s Storages) Contains(storage Storage) bool {
	for _, s := range s {
		if s.Type == storage.Type {
			return true
		}
	}
	return false
}

func (k KYCs) Contains(kyc KYC) bool {
	for _, k := range k {
		if k.Equal(kyc) {
			return true
		}
	}
	return false
}

func (ps PricesInformation) Contains(price PriceInformation) bool {
	for _, p := range ps {
		if p.Equal(price) {
			return true
		}
	}
	return false
}

// Add adds the resources of the given Capability to the current Capability
func (c *Capability) Add(cap Capability) error {
	// Executors
	for _, executor := range cap.Executors {
		if !c.Executors.Contains(executor) {
			c.Executors = append(c.Executors, executor)
		}
	}

	// JobTypes
	for _, jobType := range cap.JobTypes {
		if !c.JobTypes.Contains(jobType) {
			c.JobTypes = append(c.JobTypes, jobType)
		}
	}

	// Resources
	if c.Resources.CPU.Architecture == cap.Resources.CPU.Architecture {
		c.Resources.CPU.Cores += cap.Resources.CPU.Cores
		c.Resources.CPU.ClockSpeedHz += cap.Resources.CPU.ClockSpeedHz
	}

	c.Resources.Memory.ClockSpeedHz += cap.Resources.Memory.ClockSpeedHz // does it make sense?
	c.Resources.Memory.Size += cap.Resources.Memory.Size

	if c.Resources.Disk.Type == cap.Resources.Disk.Type {
		c.Resources.Disk.Size += cap.Resources.Disk.Size
	}

	c.Resources.GPUs = append(c.Resources.GPUs, cap.Resources.GPUs...)

	// Libraries
	var myLibraries Libraries = c.Libraries
	for _, library := range cap.Libraries {
		if !myLibraries.Contains(library) {
			c.Libraries = append(c.Libraries, library)
		}
	}

	// Localities
	var myLocalities Localities = c.Localities
	for _, locality := range cap.Localities {
		if !myLocalities.Contains(locality) {
			c.Localities = append(c.Localities, locality)
		}
	}

	// Storage
	var myStorages Storages = c.Storage
	for _, storage := range cap.Storage {
		if !myStorages.Contains(storage) {
			c.Storage = append(c.Storage, storage)
		} else {
			for i, s := range c.Storage {
				if s.Type == storage.Type {
					c.Storage[i].Size += storage.Size
					c.Storage[i].Amount += storage.Amount
				}
			}
		}
	}

	// Connectivity
	if cap.Connectivity.VPN {
		c.Connectivity.VPN = true
	}

	for _, port := range cap.Connectivity.Ports {
		if !sliceContainsInt(c.Connectivity.Ports, port) {
			c.Connectivity.Ports = append(c.Connectivity.Ports, port)
		}
	}

	// Price
	var myPrice PricesInformation = c.Price
	for _, price := range cap.Price {
		if !myPrice.Contains(price) {
			c.Price = append(c.Price, price)
		}
	}

	// Time
	c.Time.MaxTime += cap.Time.MaxTime

	// KYCs
	var myKYCs KYCs = c.KYCs
	for _, kyc := range cap.KYCs {
		if !myKYCs.Contains(kyc) {
			c.KYCs = append(c.KYCs, kyc)
		}
	}

	return nil
}

// Subtract subtracts the resources of the given Capability from the current Capability
func (c *Capability) Subtract(cap Capability) error {
	// Executors
	// No Subtract operation for Executors

	// JobTypes
	// No Subtract operation for JobTypes

	// Resources
	if c.Resources.CPU.Cores < cap.Resources.CPU.Cores || c.Resources.CPU.ClockSpeedHz < cap.Resources.CPU.ClockSpeedHz {
		return errors.New("cpu resources are not enough")
	}

	if c.Resources.Memory.Size < cap.Resources.Memory.Size {
		return errors.New("memory resources are not enough")
	}

	if c.Resources.Disk.Size < cap.Resources.Disk.Size {
		return errors.New("disk resources are not enough")
	}

	if c.Resources.CPU.Architecture == cap.Resources.CPU.Architecture {
		c.Resources.CPU.Cores -= cap.Resources.CPU.Cores
		c.Resources.CPU.Threads -= cap.Resources.CPU.Threads
		c.Resources.CPU.ClockSpeedHz -= cap.Resources.CPU.ClockSpeedHz
	}

	c.Resources.Memory.ClockSpeedHz -= cap.Resources.Memory.ClockSpeedHz // does it make sense?
	c.Resources.Memory.Size -= cap.Resources.Memory.Size

	if c.Resources.Disk.Type == cap.Resources.Disk.Type {
		c.Resources.Disk.Size -= cap.Resources.Disk.Size
	}

	// Remove the GPUs from the current Capability
	// This is a naive implementation and may not work as expected
	// if the GPUs are not unique
	for _, gpu := range cap.Resources.GPUs {
		for i, cgpu := range c.Resources.GPUs {
			//nolint
			if cgpu.Equal(&gpu) {
				c.Resources.GPUs = append(c.Resources.GPUs[:i], c.Resources.GPUs[i+1:]...)
			}
		}
	}

	// Libraries
	// No Subtract operation for Libraries

	// Localities
	// No Subtract operation for Localities

	// Storage
	for _, storage := range cap.Storage {
		for i, myStorage := range c.Storage {
			if storage.Type == myStorage.Type && storage.Amount == myStorage.Amount && storage.Size == myStorage.Size {
				c.Storage = append(c.Storage[:i], c.Storage[i+1:]...)
			} else if storage.Type == myStorage.Type {
				c.Storage[i].Size -= storage.Size
				c.Storage[i].Amount -= storage.Amount
			}
		}
	}

	// Connectivity
	for _, port := range cap.Connectivity.Ports {
		for i, cport := range c.Connectivity.Ports {
			if cport == port {
				c.Connectivity.Ports = append(c.Connectivity.Ports[:i], c.Connectivity.Ports[i+1:]...)
			}
		}
	}

	// Price
	// No Subtract operation for Price

	// Time
	c.Time.MaxTime -= cap.Time.MaxTime

	// KYCs
	// No Subtract operation for KYCs

	return nil
}

// SliceContainsInt checks if a integer  exists in a slice
func sliceContainsInt(s []int, val int) bool {
	for _, v := range s {
		if v == val {
			return true
		}
	}
	return false
}
