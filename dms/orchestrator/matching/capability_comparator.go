package matching

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
)

// CapabilityComparator compares two capabilities by ANDing the comparisons of their fields.
// it respects the following table of truth:
//
// |   AND  | Better |  Worse |  Equal |  Error |
// | ------ | ------ |--------|--------|--------|
// | Better | Better |  Worse | Better |  Error |
// | Worse  | Worse  |  Worse | Worse  |  Error |
// | Equal  | Better |  Worse | Equal  |  Error |
// | Error  | Error  |  Error | Error  |  Error |
//
// The comparison of the fields is done by the Compare function.
//
// Result = (Comparison of Executors) AND (Comparison of JobTypes) AND
// (Comparison of Resources) AND (Comparison of Libraries) AND
// (Comparison of Localities) AND (Comparison of Storage) AND
// (Comparison of Connectivity) AND (Comparison of Price) AND
// (Comparison of Time) AND (Comparison of KYCs)
func CapabilityComparator(l, r interface{}, _ ...Preference) types.Comparison {
	var result types.Comparison

	_, lok := l.(types.Capability)
	_, rok := r.(types.Capability)

	if !lok || !rok {
		fmt.Println(lok, rok)
		return types.Error
	}

	lcap := l.(types.Capability)
	rcap := r.(types.Capability)

	// Executors
	result = Compare(lcap.Executors, rcap.Executors)
	// JobTypes
	result = result.And(Compare(lcap.JobTypes, rcap.JobTypes))
	// Resources
	result = result.And(Compare(lcap.Resources, rcap.Resources))
	// Libraries
	result = result.And(Compare(lcap.Libraries, rcap.Libraries))
	// Localities
	result = result.And(Compare(lcap.Localities, rcap.Localities))
	// Storage
	result = result.And(Compare(lcap.Storage, rcap.Storage))
	// Connectivity
	result = result.And(Compare(lcap.Connectivity, rcap.Connectivity))
	// Price
	result = result.And(Compare(lcap.Price, rcap.Price))
	// Time
	result = result.And(Compare(lcap.Time, rcap.Time))
	// KYCs
	result = result.And(Compare(lcap.KYCs, rcap.KYCs))

	return result
}
