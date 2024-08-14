package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/types"
)

func TestLiteral(t *testing.T) {
	literal1 := "test"
	literal2 := "test"
	expectedValue := types.Equal
	actualValue := Compare(literal1, literal2)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGpuCompare(t *testing.T) {
	gpu1 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := types.GPU{Index: 1,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	expectedValue := types.Better
	actualValue := Compare(gpu1, gpu2)
	assert.Equal(t, expectedValue, actualValue)
}

func TestNumericComparator(t *testing.T) {
	leftValue := 3.26
	rightValue := 5
	// positive example
	actualValue := Compare(leftValue, rightValue)
	expectedValue := types.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative example
	actualValue = Compare(leftValue, rightValue)
	expectedValue = types.Equal
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestExecutorComparator(t *testing.T) {
	executor1 := types.Executor{types.ExecutorTypeDocker}
	executor2 := types.Executor{types.ExecutorTypeFirecracker}
	executor3 := types.Executor{types.ExecutorTypeDocker}
	executor4 := types.Executor{}

	// positive examples
	actualValue := Compare(executor1, executor2)
	expectedValue := types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor1, executor3)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor1, executor4)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor4, executor1)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(executor1, executor2)
	expectedValue = types.Worse
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestJobTypesComparator(t *testing.T) {
	var jobs1 types.JobTypes
	jobs1 = append(jobs1, types.BATCH)
	jobs1 = append(jobs1, types.SINGLERUN)

	var jobs2 types.JobTypes
	jobs2 = append(jobs2, types.BATCH)
	jobs2 = append(jobs2, types.LONGRUNNING)

	var jobs3 types.JobTypes
	jobs3 = append(jobs3, types.RECURRING)
	jobs3 = append(jobs3, types.SINGLERUN)

	var jobs4 types.JobTypes
	jobs4 = append(jobs4, types.BATCH)

	var jobs5 types.JobTypes
	jobs5 = append(jobs5, types.BATCH)
	jobs5 = append(jobs5, types.LONGRUNNING)
	jobs5 = append(jobs5, types.SINGLERUN)

	// positive examples
	actualValue := Compare(jobs1, jobs2)
	// currently the comparator is implemented in a way that it will return Error
	// if compared job types contain slices that are not equal
	// we may want to change it to return types.Worse which is logically more correct
	expectedValue := types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs1, jobs3)
	// currently the comparator is implemented in a way that it will return Error
	// if compared job types contain slices that are not equal
	// we may want to change it to return types.Worse which is logically more correct
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs1, jobs4)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs1)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs2)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs3)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(jobs1, jobs2)
	expectedValue = types.Equal
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestJobTypeComparator(t *testing.T) {

	// positive examples
	actualValue := Compare(types.BATCH, types.SINGLERUN)
	expectedValue := types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(types.LONGRUNNING, types.LONGRUNNING)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(types.BATCH, types.BATCH)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(types.BATCH, types.LONGRUNNING)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(types.LONGRUNNING, types.BATCH)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(types.BATCH, types.SINGLERUN)
	expectedValue = types.Worse
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestGpuComparator(t *testing.T) {
	gpu1 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := types.GPU{Index: 1,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu3 := types.GPU{Index: 2,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu4 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}

	// positive examples
	actualValue := Compare(gpu1, gpu2)
	expectedValue := types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpu1, gpu4)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpu2, gpu3)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(gpu1, gpu2)
	expectedValue = types.Worse
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestGPUsComparator(t *testing.T) {
	var gpus1 []types.GPU
	gpu1 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := types.GPU{Index: 1,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpus1 = append(gpus1, gpu1)
	gpus1 = append(gpus1, gpu2)

	var gpus2 []types.GPU
	gpu3 := types.GPU{Index: 2,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu4 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpus2 = append(gpus2, gpu3)
	gpus2 = append(gpus2, gpu4)

	var gpus3 []types.GPU
	gpu5 := types.GPU{Index: 2,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu6 := types.GPU{Index: 1,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpus3 = append(gpus3, gpu5)
	gpus3 = append(gpus3, gpu6)
	gpus1 = append(gpus1, gpu5)

	// positive examples
	actualValue := Compare(gpus1, gpus2)
	expectedValue := types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpus3, gpus1)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpus1, gpus1)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

}

func TestExecutionResourcesComparator(t *testing.T) {

	// constructing some execution resources to test upon
	cpu1 := types.CPU{Cores: 8, Freq: 1024}
	cpu2 := types.CPU{Cores: 4, Freq: 2048}
	memory1 := types.Memory{Size: 16}
	memory2 := types.Memory{Size: 8}
	disk1 := types.Disk{Size: 1024}
	disk2 := types.Disk{Size: 512}
	gpu1 := types.GPU{Index: 0,
		Vendor:     types.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := types.GPU{Index: 1,
		Vendor:     types.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}

	executionResources1 := types.ExecutionResources{
		CPU:    cpu1,
		Memory: memory1,
		Disk:   disk1,
		GPUs:   []types.GPU{gpu1, gpu2},
	}
	executionResources2 := types.ExecutionResources{
		CPU:    cpu2,
		Memory: memory2,
		Disk:   disk2,
		GPUs:   []types.GPU{gpu1},
	}
	executionResources3 := types.ExecutionResources{
		CPU:    cpu2,
		Memory: memory1,
		Disk:   disk2,
		GPUs:   []types.GPU{gpu2},
	}

	// positive examples
	actualValue := Compare(executionResources1, executionResources1)
	expectedValue := types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executionResources1, executionResources2)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executionResources3, executionResources1)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(executionResources1, executionResources1)
	expectedValue = types.Better
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestLibraryComparator(t *testing.T) {
	library1 := types.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library2 := types.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library3 := types.Library{Name: "pytorch", Version: "1.8.1", Constraint: ">"}
	library4 := types.Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}
	library5 := types.Library{Name: "pytorch", Version: "1.7.1", Constraint: ">"}

	// positive examples
	actualValue := Compare(library1, library2)
	expectedValue := types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library3, library4)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library1, library3)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library4, library5)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library5, library4)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(library1, library3)
	expectedValue = types.Better
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLibrariesComparator(t *testing.T) {
	var libraries1 []types.Library
	library1 := types.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library2 := types.Library{Name: "pytorch", Version: "1.8.1", Constraint: ">"}
	libraries1 = append(libraries1, library1)
	libraries1 = append(libraries1, library2)

	var libraries2 []types.Library
	library3 := types.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library4 := types.Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}
	libraries2 = append(libraries2, library3)
	libraries2 = append(libraries2, library4)

	var libraries3 []types.Library
	library5 := types.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library6 := types.Library{Name: "pytorch", Version: "1.7.1", Constraint: ">="}
	libraries3 = append(libraries3, library5)
	libraries3 = append(libraries3, library6)
	libraries3 = append(libraries3, library6)

	// positive examples
	actualValue := Compare(libraries1, libraries3)
	expectedValue := types.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(libraries1, libraries2)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(libraries3, libraries1)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(libraries1, libraries2)
	expectedValue = types.Better
	assert.NotEqual(t, expectedValue, actualValue)

	actualValue = Compare(libraries1, libraries2)
	expectedValue = types.Worse
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLocalityComparator(t *testing.T) {
	locality1 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality2 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality3 := types.Locality{Kind: "GeoRegion", Name: "EU"}
	locality4 := types.Locality{Kind: "GeoRegion", Name: "US"}
	locality5 := types.Locality{Kind: "GeoRegion", Name: "EU"}

	// positive examples
	actualValue := Compare(locality1, locality2)
	expectedValue := types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality3, locality4)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality3, locality5)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality1, locality3)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(locality1, locality3)
	expectedValue = types.Better
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLocalitiesComparator(t *testing.T) {
	var localities1 []types.Locality
	locality1 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality2 := types.Locality{Kind: "GeoRegion", Name: "EU"}
	localities1 = append(localities1, locality1)
	localities1 = append(localities1, locality2)

	var localities2 []types.Locality
	locality3 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality4 := types.Locality{Kind: "GeoRegion", Name: "US"}
	localities2 = append(localities2, locality3)
	localities2 = append(localities2, locality4)

	var localities3 []types.Locality
	locality5 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality6 := types.Locality{Kind: "GeoRegion", Name: "EU"}
	localities3 = append(localities3, locality5)
	localities3 = append(localities3, locality6)

	var localities4 []types.Locality
	locality7 := types.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality8 := types.Locality{Kind: "GeoRegion", Name: "EU"}
	locality9 := types.Locality{Kind: "GeoCounty", Name: "Belgium"}

	localities4 = append(localities4, locality7)
	localities4 = append(localities4, locality8)
	localities4 = append(localities4, locality9)

	// positive examples
	actualValue := Compare(localities4, localities1)
	expectedValue := types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities1, localities2)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities1, localities3)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities2, localities3)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(localities1, localities2)
	expectedValue = types.Equal
	assert.NotEqual(t, expectedValue, actualValue)
}
