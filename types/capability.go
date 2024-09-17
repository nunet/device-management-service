package types

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/go-version"
)

// Connectivity represents the network configuration
type Connectivity struct {
	Ports []int `json:"ports" description:"Ports that need to be open for the job to run"`
	VPN   bool  `json:"vpn" description:"Whether VPN is required"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[Connectivity] = (*Connectivity)(nil)
	_ Calculable[Connectivity] = (*Connectivity)(nil)
)

func (c *Connectivity) Compare(other Connectivity) Comparison {
	if reflect.DeepEqual(*c, other) {
		return Equal
	}

	if IsStrictlyContainedInt(c.Ports, other.Ports) && (c.VPN && other.VPN || c.VPN && !other.VPN) {
		return Better
	}

	return Worse
}

func (c *Connectivity) Add(other Connectivity) error {
	portSet := make(map[int]struct{}, len(c.Ports))

	for _, port := range c.Ports {
		portSet[port] = struct{}{}
	}

	for _, port := range other.Ports {
		if _, exists := portSet[port]; !exists {
			c.Ports = append(c.Ports, port)
		}
	}

	// Set VPN to true if other has VPN
	c.VPN = c.VPN || other.VPN

	return nil
}

func (c *Connectivity) Subtract(other Connectivity) error {
	if other.VPN {
		c.VPN = false
	}

	// Filter out the ports that exist in 'other' from 'c'
	filteredPorts := c.Ports[:0]
	for _, port := range c.Ports {
		if !slices.Contains(other.Ports, port) {
			filteredPorts = append(filteredPorts, port)
		}
	}

	// Set the filtered ports back to c
	if len(filteredPorts) == 0 {
		c.Ports = nil
	} else {
		c.Ports = filteredPorts
	}

	return nil
}

// TimeInformation represents the time constraints
type TimeInformation struct {
	Units      string `json:"units" description:"Time units"`
	MaxTime    int    `json:"max_time" description:"Maximum time that job should run"`
	Preference int    `json:"preference" description:"Time preference"`
}

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[TimeInformation] = (*TimeInformation)(nil)
	_ Calculable[TimeInformation] = (*TimeInformation)(nil)
)

func (t *TimeInformation) Add(other TimeInformation) error {
	t.MaxTime += other.MaxTime

	// If there are other time fields, add them here
	// Example:
	// t.MinTime += other.MinTime
	// t.AvgTime += other.AvgTime
	return nil
}

func (t *TimeInformation) Subtract(other TimeInformation) error {
	t.MaxTime -= other.MaxTime

	// If there are other time fields, subtract them here
	// Example:
	// t.MinTime -= other.MinTime
	// t.AvgTime -= other.AvgTime
	return nil
}

func (t *TimeInformation) TotalTime() int {
	switch t.Units {
	case "seconds":
		return t.MaxTime
	case "minutes":
		return t.MaxTime * 60
	case "hours":
		return t.MaxTime * 60 * 60
	case "days":
		return t.MaxTime * 60 * 60 * 24
	default:
		return t.MaxTime
	}
}

func (t *TimeInformation) Compare(other TimeInformation) Comparison {
	if reflect.DeepEqual(t, other) {
		return Equal
	}
	ownTotalTime := t.TotalTime()
	otherTotalTime := other.TotalTime()

	if ownTotalTime == otherTotalTime {
		return Equal
	}

	if ownTotalTime < otherTotalTime {
		return Worse
	}
	return Better
}

// PriceInformation represents the pricing information
type PriceInformation struct {
	Currency        string `json:"currency" description:"Currency used for pricing"`
	CurrencyPerHour int    `json:"currency_per_hour" description:"Price charged per hour"`
	TotalPerJob     int    `json:"total_per_job" description:"Maximum total price or budget of the job"`
	Preference      int    `json:"preference" description:"Pricing preference"`
}

// implementing Comparable interface
var _ Comparable[PriceInformation] = (*PriceInformation)(nil)

func (p *PriceInformation) Compare(other PriceInformation) Comparison {
	if reflect.DeepEqual(p, other) {
		return Equal
	}

	if p.Currency == other.Currency {
		if p.TotalPerJob == other.TotalPerJob {
			if p.CurrencyPerHour == other.CurrencyPerHour {
				return Equal
			} else if p.CurrencyPerHour < other.CurrencyPerHour {
				return Better
			}

			return Worse
		}

		if p.TotalPerJob < other.TotalPerJob {
			if p.CurrencyPerHour <= other.CurrencyPerHour {
				return Better
			}

			return Worse
		}

		return Worse
	}

	return Error
}

func (p *PriceInformation) Equal(price PriceInformation) bool {
	return p.Currency == price.Currency &&
		p.CurrencyPerHour == price.CurrencyPerHour &&
		p.TotalPerJob == price.TotalPerJob &&
		p.Preference == price.Preference
}

// Library represents the libraries
type Library struct {
	Name       string `json:"name" description:"Name of the library"`
	Constraint string `json:"constraint" description:"Constraint of the library"`
	Version    string `json:"version" description:"Version of the library"`
}

// implementing Comparable interface
var _ Comparable[Library] = (*Library)(nil)

func (lib *Library) Compare(other Library) Comparison {
	ownVersion, err := version.NewVersion(lib.Version)
	if err != nil {
		return Error
	}

	// return 'Error' if the version of the left library is not valid
	constraints, err := version.NewConstraint(other.Constraint + " " + other.Version)
	if err != nil {
		return Error
	}

	// return 'Error' if the names of the libraries are different
	if lib.Name != other.Name {
		return Error
	}

	// else return 'Equal if versions of libraries are equal and the constraint is '='
	if other.Constraint == "=" && constraints.Check(ownVersion) {
		return Equal
	}

	// else return 'Better' if versions of libraries match the constraint
	if constraints.Check(ownVersion) {
		return Better
	}

	// else return 'Worse'
	return Worse
}

func (lib *Library) Equal(library Library) bool {
	if lib.Name == library.Name && lib.Constraint == library.Constraint && lib.Version == library.Version {
		return true
	}
	return false
}

// Locality represents the locality
type Locality struct {
	Kind string `json:"kind" description:"Kind of the region (geographic, nunet-defined, etc)"`
	Name string `json:"name" description:"Name of the region"`
}

// implementing Comparable interface
var _ Comparable[Locality] = (*Locality)(nil)

func (loc *Locality) Compare(other Locality) Comparison {
	if loc.Kind == other.Kind {
		if loc.Name == other.Name {
			return Equal
		}
		return Worse
	}

	return Error
}

func (loc *Locality) Equal(locality Locality) bool {
	if loc.Kind == locality.Kind && loc.Name == locality.Name {
		return true
	}

	return false
}

// KYC represents the KYC data
type KYC struct {
	Type string `json:"type" description:"Type of KYC"`
	Data string `json:"data" description:"Data required for KYC"`
}

// implementing Comparable interface
var _ Comparable[KYC] = (*KYC)(nil)

func (k *KYC) Compare(other KYC) Comparison {
	if reflect.DeepEqual(*k, other) {
		return Equal
	}

	return Error
}

func (k *KYC) Equal(kyc KYC) bool {
	if k.Type == kyc.Type && k.Data == kyc.Data {
		return true
	}
	return false
}

// JobType represents the type of the job
type JobType string

const (
	Batch       JobType = "batch"
	SingleRun   JobType = "single_run"
	Recurring   JobType = "recurring"
	LongRunning JobType = "long_running"
)

// implementing Comparable and Calculable interface
var _ Comparable[JobType] = (*JobType)(nil)

func (j JobType) Compare(other JobType) Comparison {
	if reflect.DeepEqual(j, other) {
		return Equal
	}
	return Error
}

// JobTypes a slice of JobType
type JobTypes []JobType

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[JobTypes] = (*JobTypes)(nil)
	_ Calculable[JobTypes] = (*JobTypes)(nil)
)

func (j *JobTypes) Add(other JobTypes) error {
	existing := *j
	result := existing[:0]

	existingSet := make(map[JobType]struct{}, len(result))
	for _, job := range *j {
		result = append(result, job)
		existingSet[job] = struct{}{}
	}

	for _, job := range other {
		if _, exists := existingSet[job]; !exists {
			result = append(result, job)
			existingSet[job] = struct{}{}
		}
	}

	*j = result[:len(result):len(result)] // Resize the slice to the new length
	return nil
}

func (j *JobTypes) Subtract(other JobTypes) error {
	existing := *j
	result := existing[:0]

	toRemove := make(map[JobType]struct{}, len(other))
	for _, job := range other {
		toRemove[job] = struct{}{}
	}

	for _, job := range existing {
		if _, found := toRemove[job]; !found {
			result = append(result, job)
		}
	}

	*j = result[:len(result):len(result)] // Resize the slice to the new length
	return nil
}

func (j *JobTypes) Compare(other JobTypes) Comparison {
	// we know that interfaces here are slices, so need to assert first
	l := ConvertTypedSliceToUntypedSlice(*j)
	r := ConvertTypedSliceToUntypedSlice(other)

	if !IsSameShallowType(l, r) {
		return Error
	}

	switch {
	case reflect.DeepEqual(l, r):
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		return Equal

	case IsStrictlyContained(l, r):
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'
		return Better

	case IsStrictlyContained(r, l):
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		return Worse

		// TODO: this comparator does not take into account options when several job types are available and several job types are required
		// in the same data structure; this is why the test fails;
	}

	return Error
}

func (j *JobTypes) Contains(jobType JobType) bool {
	for _, j := range *j {
		if j == jobType {
			return true
		}
	}
	return false
}

type Libraries []Library

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[Libraries] = (*Libraries)(nil)
	_ Calculable[Libraries] = (*Libraries)(nil)
)

func (l *Libraries) Add(other Libraries) error {
	existing := make(map[Library]struct{}, len(*l))

	for _, lib := range *l {
		existing[lib] = struct{}{}
	}

	for _, otherLibrary := range other {
		if _, found := existing[otherLibrary]; !found {
			*l = append(*l, otherLibrary)
			existing[otherLibrary] = struct{}{}
		}
	}

	return nil
}

func (l *Libraries) Subtract(other Libraries) error {
	// remove from array
	existing := *l
	result := existing[:0] // Reuse the underlying slice to avoid allocations

	toRemove := make(map[Library]struct{}, len(other))
	for _, ex := range other {
		toRemove[ex] = struct{}{}
	}

	for _, ex2 := range existing {
		if _, found := toRemove[ex2]; !found {
			result = append(result, ex2)
		}
	}

	*l = result
	return nil
}

func (l *Libraries) Compare(other Libraries) Comparison {
	interimComparison1 := make([][]Comparison, 0)
	for _, otherLibrary := range other {
		var interimComparison2 []Comparison
		for _, ownLibrary := range *l {
			interimComparison2 = append(interimComparison2, ownLibrary.Compare(otherLibrary))
		}
		// this matrix structure will hold the comparison results for each GPU on the right
		// with each GPU on the left in the order they are in the slices
		// first dimension represents left GPUs
		// second dimension represents right GPUs
		interimComparison1 = append(interimComparison1, interimComparison2)
	}
	// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right

	finalComparison := make([]Comparison, 0)
	for i := 0; i < len(interimComparison1); i++ {
		// we need to find the best match for each GPU on the right
		if len(interimComparison1[i]) < i {
			break
		}
		c := interimComparison1[i]
		bestMatch, index := returnBestMatch(c)
		finalComparison = append(finalComparison, bestMatch)
		interimComparison1 = removeIndex(interimComparison1, index)
	}

	if slices.Contains(finalComparison, Error) {
		return Error
	}
	if slices.Contains(finalComparison, Worse) {
		return Worse
	}
	if SliceContainsOneValue(finalComparison, Equal) {
		return Equal
	}
	return Better
}

func (l *Libraries) Contains(library Library) bool {
	for _, lib := range *l {
		if lib.Equal(library) {
			return true
		}
	}
	return false
}

type Localities []Locality

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[Localities] = (*Localities)(nil)
	_ Calculable[Localities] = (*Localities)(nil)
)

func (l *Localities) Compare(other Localities) Comparison {
	interimComparison := make([]map[string]Comparison, 0)
	for _, otherLocality := range other {
		field := make(map[string]Comparison)
		field[otherLocality.Kind] = Error
		for _, ownLocality := range *l {
			if ownLocality.Kind == otherLocality.Kind {
				field[otherLocality.Kind] = ownLocality.Compare(otherLocality)
				// this is to make sure that we have a comparison even if slice dimentiones do not match
			}
		}
		interimComparison = append(interimComparison, field)
	}
	// we can now implement a logic to figure out if each required GPU on the left has a matching GPU on the right
	var finalComparison []Comparison
	for _, c := range interimComparison {
		for _, v := range c { // we know that there is only one value in the map
			finalComparison = append(finalComparison, v)
		}
	}

	if slices.Contains(finalComparison, Error) {
		return Error
	}
	if slices.Contains(finalComparison, Worse) {
		return Worse
	}
	if SliceContainsOneValue(finalComparison, Equal) {
		return Equal
	}
	return Better
}

func (l *Localities) Add(other Localities) error {
	existing := make(map[Locality]struct{}, len(*l))

	for _, loc := range *l {
		existing[loc] = struct{}{}
	}

	for _, otherLocality := range other {
		if _, found := existing[otherLocality]; !found {
			*l = append(*l, otherLocality)
			existing[otherLocality] = struct{}{}
		}
	}

	return nil
}

func (l *Localities) Subtract(other Localities) error {
	// remove from array
	existing := *l
	result := existing[:0] // Reuse the underlying slice to avoid allocations

	toRemove := make(map[Locality]struct{}, len(other))
	for _, ex := range other {
		toRemove[ex] = struct{}{}
	}

	for _, ex := range existing {
		if _, found := toRemove[ex]; !found {
			result = append(result, ex)
		}
	}

	*l = result
	return nil
}

func (l *Localities) Contains(locality Locality) bool {
	for _, loc := range *l {
		if loc.Equal(locality) {
			return true
		}
	}
	return false
}

type KYCs []KYC

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[KYCs] = (*KYCs)(nil)
	_ Calculable[KYCs] = (*KYCs)(nil)
)

func (k *KYCs) Compare(other KYCs) Comparison {
	if reflect.DeepEqual(*k, other) {
		return Equal
	} else if len(other) == 0 && len(*k) != 0 {
		return Better
	}

	for _, ownKYC := range *k {
		for _, otherKYC := range other {
			if comp := ownKYC.Compare(otherKYC); comp == Equal {
				return Equal
			}
		}
	}

	return Error
}

func (k *KYCs) Add(other KYCs) error {
	existing := make(map[KYC]struct{}, len(*k))

	for _, kyc := range *k {
		existing[kyc] = struct{}{}
	}

	for _, otherKYC := range other {
		if _, found := existing[otherKYC]; !found {
			*k = append(*k, otherKYC)
			existing[otherKYC] = struct{}{}
		}
	}

	return nil
}

func (k *KYCs) Subtract(other KYCs) error {
	// remove from array
	existing := *k
	result := existing[:0] // Reuse the underlying slice to avoid allocations

	toRemove := make(map[KYC]struct{}, len(other))
	for _, ex := range other {
		toRemove[ex] = struct{}{}
	}

	for _, ex := range existing {
		if _, found := toRemove[ex]; !found {
			result = append(result, ex)
		}
	}

	*k = result
	return nil
}

func (k *KYCs) Contains(kyc KYC) bool {
	for _, k := range *k {
		if k.Equal(kyc) {
			return true
		}
	}
	return false
}

type PricesInformation []PriceInformation

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[PricesInformation] = (*PricesInformation)(nil)
	_ Calculable[PricesInformation] = (*PricesInformation)(nil)
)

func (ps *PricesInformation) Add(other PricesInformation) error {
	existing := make(map[PriceInformation]struct{}, len(*ps))

	for _, p := range *ps {
		existing[p] = struct{}{}
	}

	for _, otherPrice := range other {
		if _, found := existing[otherPrice]; !found {
			*ps = append(*ps, otherPrice)
			existing[otherPrice] = struct{}{}
		}
	}

	return nil
}

func (ps *PricesInformation) Subtract(other PricesInformation) error {
	if len(other) == 0 {
		return nil // Nothing to subtract, no operation needed
	}

	toRemove := make(map[PriceInformation]struct{}, len(other))
	for _, ex := range other {
		toRemove[ex] = struct{}{}
	}

	result := (*ps)[:0] // Reuse the slice's underlying array
	for _, ex := range *ps {
		if _, found := toRemove[ex]; !found {
			result = append(result, ex) // Keep entries not in 'toRemove'
		}
	}

	*ps = result[:len(result):len(result)] // Resize the slice to the new length
	return nil
}

func (ps *PricesInformation) Compare(other PricesInformation) Comparison {
	if reflect.DeepEqual(*ps, other) {
		return Equal
	}

	comparison := Error
	for _, ownPrice := range *ps {
		for _, otherPrice := range other {
			if comparison = ownPrice.Compare(otherPrice); comparison != Error {
				return comparison
			}
		}
	}

	return comparison
}

func (ps *PricesInformation) Contains(price PriceInformation) bool {
	for _, p := range *ps {
		if p.Equal(price) {
			return true
		}
	}
	return false
}

// HardwareCapability represents the hardware capability of the machine
type HardwareCapability struct {
	Executors    Executors         `json:"executor" description:"Executor type required for the job (docker, vm, wasm, or others)"`
	JobTypes     JobTypes          `json:"type" description:"Details about type of the job (One time, batch, recurring, long running). Refer to dms.jobs package for jobType data model"`
	Resources    Resources         `json:"resources" description:"Resources required for the job"`
	Libraries    Libraries         `json:"libraries" description:"Libraries required for the job"`
	Localities   Localities        `json:"locality" description:"Preferred localities of the machine for execution"`
	Connectivity Connectivity      `json:"connectivity" description:"Network configuration required"`
	Price        PricesInformation `json:"price" description:"Pricing information"`
	Time         TimeInformation   `json:"time" description:"Time constraints"`
	KYCs         KYCs
}

// implementing Comparable and Calculable interfaces
var (
	_ Comparable[HardwareCapability] = (*HardwareCapability)(nil)
	_ Calculable[HardwareCapability] = (*HardwareCapability)(nil)
)

// Compare compares two HardwareCapability objects
func (c *HardwareCapability) Compare(other HardwareCapability) Comparison {
	comparisonMap := ComplexComparison{
		"Executors":    c.Executors.Compare(other.Executors),
		"JobTypes":     c.JobTypes.Compare(other.JobTypes),
		"Resources":    c.Resources.Compare(other.Resources),
		"Libraries":    c.Libraries.Compare(other.Libraries),
		"Localities":   c.Localities.Compare(other.Localities),
		"Price":        c.Price.Compare(other.Price),
		"KYCs":         c.KYCs.Compare(other.KYCs),
		"Time":         c.Time.Compare(other.Time),
		"Connectivity": c.Connectivity.Compare(other.Connectivity),
	}

	return comparisonMap.Result()
}

// Add adds the resources of the given HardwareCapability to the current HardwareCapability
func (c *HardwareCapability) Add(other HardwareCapability) error {
	// Executors
	if err := c.Executors.Add(other.Executors); err != nil {
		return fmt.Errorf("error adding Executors")
	}

	// JobTypes
	if err := c.JobTypes.Add(other.JobTypes); err != nil {
		return fmt.Errorf("error adding JobTypes: %v", err)
	}

	// Resources
	if err := c.Resources.Add(other.Resources); err != nil {
		return err
	}

	// Libraries
	if err := c.Libraries.Add(other.Libraries); err != nil {
		return fmt.Errorf("error adding Libraries: %v", err)
	}

	// Localities
	if err := c.Localities.Add(other.Localities); err != nil {
		return fmt.Errorf("error adding Localities: %v", err)
	}

	// Connectivity
	if err := c.Connectivity.Add(other.Connectivity); err != nil {
		return fmt.Errorf("error adding Connectivity: %v", err)
	}

	// Price
	if err := c.Price.Add(other.Price); err != nil {
		return fmt.Errorf("error adding Price: %v", err)
	}

	// Time
	if err := c.Time.Add(other.Time); err != nil {
		return fmt.Errorf("error adding Time: %v", err)
	}

	// KYCs
	if err := c.KYCs.Add(other.KYCs); err != nil {
		return fmt.Errorf("error adding KYCs: %v", err)
	}

	return nil
}

// Subtract subtracts the resources of the given HardwareCapability from the current HardwareCapability
func (c *HardwareCapability) Subtract(cap HardwareCapability) error {
	// Executors
	if err := c.Executors.Subtract(cap.Executors); err != nil {
		return fmt.Errorf("error subtracting Executors: %v", err)
	}

	// JobTypes
	if err := c.JobTypes.Subtract(cap.JobTypes); err != nil {
		return fmt.Errorf("error comparing JobTypes: %v", err)
	}

	// Resources
	if err := c.Resources.Subtract(cap.Resources); err != nil {
		return fmt.Errorf("error subtracting Resources: %v", err)
	}

	// Libraries
	if err := c.Libraries.Subtract(cap.Libraries); err != nil {
		return fmt.Errorf("error subtracting Libraries: %v", err)
	}

	// Localities
	if err := c.Localities.Subtract(cap.Localities); err != nil {
		return fmt.Errorf("error subtracting Localities: %v", err)
	}

	// Connectivity
	if err := c.Connectivity.Subtract(cap.Connectivity); err != nil {
		return fmt.Errorf("error subtracting Connectivity: %v", err)
	}

	// Price
	if err := c.Price.Subtract(cap.Price); err != nil {
		return fmt.Errorf("error subtracting Price: %v", err)
	}

	// Time
	if err := c.Time.Subtract(cap.Time); err != nil {
		return fmt.Errorf("error subtracting Time: %v", err)
	}

	// KYCs
	if err := c.KYCs.Subtract(cap.KYCs); err != nil {
		return fmt.Errorf("error subtracting KYCs: %v", err)
	}

	return nil
}
