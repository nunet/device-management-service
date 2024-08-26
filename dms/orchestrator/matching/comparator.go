package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

// generic compare function for comparing any custom types given a custom comparator
// for simple types (i.e which are not nested in the map[string]interface{} structure)

type Comparator func(l, r interface{}, preference ...Preference) types.Comparison

func Compare(l, r interface{}, _ ...Preference) types.Comparison {
	// TODO: it would be better to pass a pointer as this is a global structure
	comparatorMap := initComparatorMap()

	// check if the type is numeric
	if _, numeric := validate.ConvertNumericToFloat64(l); numeric {
		comparator := comparatorMap["Numeric"]
		if comparator == nil {
			return types.Error
		}
		return comparator(l, r)
	}

	typeName := reflect.TypeOf(l).Name()
	// this means that the type is probably a slice of custom types
	// we have to get the element types and then map it to existing custom types that know
	// so that we can call a correct comparator for that
	//nolint
	switch reflect.TypeOf(l).Kind() {
	// check if we have a slice of further types
	// we need to mention each type explicitly
	case reflect.Slice:
		if _, ok := l.([]types.GPU); ok {
			typeName = "GPUs"
		}
		if _, ok := l.([]types.Library); ok {
			typeName = "Libraries"
		}
		if _, ok := l.([]types.Locality); ok {
			typeName = "Localities"
		}
		if _, ok := l.([]types.Storage); ok {
			typeName = "Storages"
		}
		if _, ok := l.([]types.PriceInformation); ok {
			typeName = "PricesInformation"
		}
		if _, ok := l.([]types.KYC); ok {
			typeName = "KYCs"
		}
	}

	// select the comparator based on type
	comparator := comparatorMap[typeName]
	if comparator == nil {
		return types.Error
	}
	return comparator(l, r)
}

type ComparatorMap map[string]Comparator

func initComparatorMap() ComparatorMap {
	// comparatorMap holds all defined comparators in a variable that can be passed
	// around and searched / referenced
	comparatorMap := make(map[string]Comparator)

	comparatorMap["Numeric"] = NumericComparator
	comparatorMap["Capability"] = CapabilityComparator
	comparatorMap["string"] = LiteralComparator
	comparatorMap["Executors"] = ExecutorsComparator
	comparatorMap["ExecutorType"] = ExecutorTypeComparator
	comparatorMap["JobType"] = JobTypeComparator
	comparatorMap["JobTypes"] = JobTypesComparator
	comparatorMap["GPUVendor"] = GPUVendorComparator
	comparatorMap["GPUs"] = GPUsComparator
	comparatorMap["GPU"] = GpuComparator
	comparatorMap["Executor"] = ExecutorComparator
	comparatorMap["ExecutionResources"] = ExecutionResourcesComparator
	comparatorMap["CPU"] = CPUComparator
	comparatorMap["RAM"] = MemoryComparator
	comparatorMap["Disk"] = DiskComparator
	comparatorMap["Library"] = LibraryComparator
	comparatorMap["Libraries"] = LibrariesComparator
	comparatorMap["Locality"] = LocalityComparator
	comparatorMap["Localities"] = LocalitiesComparator
	comparatorMap["Storage"] = StorageComparator
	comparatorMap["Storages"] = StoragesComparator
	comparatorMap["Connectivity"] = ConnectivityComparator
	comparatorMap["PriceInformation"] = PriceInformationComparator
	comparatorMap["PricesInformation"] = PricesInformationComparator
	comparatorMap["TimeInformation"] = TimeInformationComparator
	comparatorMap["KYC"] = KYCComparator
	comparatorMap["KYCs"] = KYCsComparator

	return comparatorMap
}

type Preference struct {
	TypeName                  string
	Strength                  PreferenceString
	DefaultComparatorOverride Comparator
}

type PreferenceString string

const (
	Hard PreferenceString = "Hard"
	Soft PreferenceString = "Soft"
)
