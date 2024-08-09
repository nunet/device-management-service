package matching

import (
	"reflect"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

// generic compare function for comparing any custom types given a custom comparator
// for simple types (i.e which are not nested in the map[string]interface{} structure)

type Comparator func(l, r interface{}, preference ...Preference) models.Comparison

func Compare(l, r interface{}, preference ...Preference) models.Comparison {

	// TODO: it would be better to pass a pointer as this is a global structure
	var comparatorMap = initComparatorMap()
	
	// check if the type is numeric
	if _, numeric := validate.ConvertNumericToFloat64(l); numeric {
		comparator := comparatorMap["Numeric"]
		if comparator == nil {
			return models.Error
		}
		return comparator(l, r)
	}

    typeName := reflect.TypeOf(l).Name()
	// this means that the type is probably a slice of custom types
	// we have to get the element types and then map it to existing custom types that know
	// so that we can call a correct comparator for that
	switch reflect.TypeOf(l).Kind() { 
		// check if we have a slice of further types
		// we need to mention each type explicitly
		case reflect.Slice:
			if _, ok := l.([]models.GPU); ok {
				typeName = "GPUs"
			}
			if _, ok := l.([]models.Library); ok {
				typeName = "Libraries"
			}
			if _, ok := l.([]models.Locality); ok {
				typeName = "Localities"
			}
			if _, ok := l.([]models.Locality); ok {
				typeName = "Localities"
			}
	}
	// select the comparator based on type
	comparator := comparatorMap[typeName]
	if comparator == nil {
		return models.Error
	}
	return comparator(l, r)

}

type ComparatorMap map[string]Comparator

func initComparatorMap() ComparatorMap {

	// comparatorMap holds all defined comparators in a variable that can be passed
	// around and searched / referenced
	var comparatorMap = make(map[string]Comparator)

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
	comparatorMap["CPU"] = CpuComparator	
	comparatorMap["Memory"] = MemoryComparator	
	comparatorMap["Disk"] = DiskComparator	
	comparatorMap["Library"] = LibraryComparator
	comparatorMap["Libraries"] = LibrariesComparator
	comparatorMap["Locality"] = LocalityComparator
	comparatorMap["Localities"] = LocalitiesComparator
	return comparatorMap
}

type Preference struct {
	TypeName string
	Strength PreferenceString
	DefaultComparatorOverride Comparator
}

type PreferenceString string

const (
	Hard PreferenceString = "Hard"
	Soft PreferenceString = "Soft"
)
