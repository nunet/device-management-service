package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/models"
)

func TestLiteral(t *testing.T) {
	literal1 := "test"
	literal2 := "test"
	expectedValue := models.Equal
	actualValue := Compare(literal1, literal2)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGpuCompare(t *testing.T) {
	gpu1 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := models.GPU{Index: 1,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	expectedValue := models.Better
	actualValue := Compare(gpu1, gpu2)
	assert.Equal(t, expectedValue, actualValue)
}

func TestNumericComparator(t *testing.T) {
	leftValue := 3.26
	rightValue := 5
	// positive example
	actualValue := Compare(leftValue, rightValue)
	expectedValue := models.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative example
	actualValue = Compare(leftValue, rightValue)
	expectedValue = models.Equal
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestExecutorComparator(t *testing.T) {
	executor1 := models.Executor{models.ExecutorTypeDocker}
	executor2 := models.Executor{models.ExecutorTypeFirecracker}
	executor3 := models.Executor{models.ExecutorTypeDocker}
	executor4 := models.Executor{}

	// positive examples
	actualValue := Compare(executor1, executor2)
	expectedValue := models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor1, executor3)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor1, executor4)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executor4, executor1)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(executor1, executor2)
	expectedValue = models.Worse
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestJobTypesComparator(t *testing.T) {
	var jobs1 models.JobTypes
	jobs1 = append(jobs1, models.BATCH)
	jobs1 = append(jobs1, models.SINGLERUN)

	var jobs2 models.JobTypes
	jobs2 = append(jobs2, models.BATCH)
	jobs2 = append(jobs2, models.LONGRUNNING)

	var jobs3 models.JobTypes
	jobs3 = append(jobs3, models.RECURRING)
	jobs3 = append(jobs3, models.SINGLERUN)

	var jobs4 models.JobTypes
	jobs4 = append(jobs4, models.BATCH)

	var jobs5 models.JobTypes
	jobs5 = append(jobs5, models.BATCH)
	jobs5 = append(jobs5, models.LONGRUNNING)
	jobs5 = append(jobs5, models.SINGLERUN)

	// positive examples
	actualValue := Compare(jobs1, jobs2)
	// currently the comparator is implemented in a way that it will return Error
	// if compared job types contain slices that are not equal
	// we may want to change it to return models.Worse which is logically more correct
	expectedValue := models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs1, jobs3)
	// currently the comparator is implemented in a way that it will return Error
	// if compared job types contain slices that are not equal
	// we may want to change it to return models.Worse which is logically more correct
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs1, jobs4)
	expectedValue = models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs1)
	expectedValue = models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs2)
	expectedValue = models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(jobs5, jobs3)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(jobs1, jobs2)
	expectedValue = models.Equal
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestJobTypeComparator(t *testing.T) {

	// positive examples
	actualValue := Compare(models.BATCH, models.SINGLERUN)
	expectedValue := models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(models.LONGRUNNING, models.LONGRUNNING)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(models.BATCH, models.BATCH)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(models.BATCH, models.LONGRUNNING)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(models.LONGRUNNING, models.BATCH)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(models.BATCH, models.SINGLERUN)
	expectedValue = models.Worse
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestGpuComparator(t *testing.T) {
	gpu1 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := models.GPU{Index: 1,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu3 := models.GPU{Index: 2,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu4 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}

	// positive examples
	actualValue := Compare(gpu1, gpu2)
	expectedValue := models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpu1, gpu4)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpu2, gpu3)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(gpu1, gpu2)
	expectedValue = models.Worse
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestGPUsComparator(t *testing.T) {
	var gpus1 []models.GPU
	gpu1 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := models.GPU{Index: 1,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpus1 = append(gpus1, gpu1)
	gpus1 = append(gpus1, gpu2)

	var gpus2 []models.GPU
	gpu3 := models.GPU{Index: 2,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu4 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpus2 = append(gpus2, gpu3)
	gpus2 = append(gpus2, gpu4)

	var gpus3 []models.GPU
	gpu5 := models.GPU{Index: 2,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpu6 := models.GPU{Index: 1,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}
	gpus3 = append(gpus3, gpu5)
	gpus3 = append(gpus3, gpu6)
	gpus1 = append(gpus1, gpu5)

	// positive examples
	actualValue := Compare(gpus1, gpus2)
	expectedValue := models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpus3, gpus1)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(gpus1, gpus1)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

}

func TestExecutionResourcesComparator(t *testing.T) {

	// constructing some execution resources to test upon
	cpu1 := models.CPU{Cores: 8, Freq: 1024}
	cpu2 := models.CPU{Cores: 4, Freq: 2048}
	memory1 := models.Memory{Size: 16}
	memory2 := models.Memory{Size: 8}
	disk1 := models.Disk{Size: 1024}
	disk2 := models.Disk{Size: 512}
	gpu1 := models.GPU{Index: 0,
		Vendor:     models.GPUVendorNvidia,
		PCIAddress: "AAAA:BB:CC.C",
		Model:      "Tesla T4 A100",
		VRAM:       16384}
	gpu2 := models.GPU{Index: 1,
		Vendor:     models.GPUVendorIntel,
		PCIAddress: "AAAA:BB:CC.D",
		Model:      "Intel A770",
		VRAM:       8192}

	executionResources1 := models.ExecutionResources{
		CPU:    cpu1,
		Memory: memory1,
		Disk:   disk1,
		GPUs:   []models.GPU{gpu1, gpu2},
	}
	executionResources2 := models.ExecutionResources{
		CPU:    cpu2,
		Memory: memory2,
		Disk:   disk2,
		GPUs:   []models.GPU{gpu1},
	}
	executionResources3 := models.ExecutionResources{
		CPU:    cpu2,
		Memory: memory1,
		Disk:   disk2,
		GPUs:   []models.GPU{gpu2},
	}

	// positive examples
	actualValue := Compare(executionResources1, executionResources1)
	expectedValue := models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executionResources1, executionResources2)
	expectedValue = models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(executionResources3, executionResources1)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(executionResources1, executionResources1)
	expectedValue = models.Better
	assert.NotEqual(t, expectedValue, actualValue)

}

func TestLibraryComparator(t *testing.T) {
	library1 := models.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library2 := models.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library3 := models.Library{Name: "pytorch", Version: "1.8.1", Constraint: ">"}
	library4 := models.Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}
	library5 := models.Library{Name: "pytorch", Version: "1.7.1", Constraint: ">"}

	// positive examples
	actualValue := Compare(library1, library2)
	expectedValue := models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library3, library4)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library1, library3)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library4, library5)
	expectedValue = models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(library5, library4)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(library1, library3)
	expectedValue = models.Better
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLibrariesComparator(t *testing.T) {
	var libraries1 []models.Library
	library1 := models.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library2 := models.Library{Name: "pytorch", Version: "1.8.1", Constraint: ">"}
	libraries1 = append(libraries1, library1)
	libraries1 = append(libraries1, library2)

	var libraries2 []models.Library
	library3 := models.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library4 := models.Library{Name: "pytorch", Version: "1.8.1", Constraint: "="}
	libraries2 = append(libraries2, library3)
	libraries2 = append(libraries2, library4)

	var libraries3 []models.Library
	library5 := models.Library{Name: "tensorflow", Version: "2.4.1", Constraint: "="}
	library6 := models.Library{Name: "pytorch", Version: "1.7.1", Constraint: ">="}
	libraries3 = append(libraries3, library5)
	libraries3 = append(libraries3, library6)
	libraries3 = append(libraries3, library6)

	// positive examples
	actualValue := Compare(libraries1, libraries3)
	expectedValue := models.Better
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(libraries1, libraries2)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(libraries3, libraries1)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(libraries1, libraries2)
	expectedValue = models.Better
	assert.NotEqual(t, expectedValue, actualValue)

	actualValue = Compare(libraries1, libraries2)
	expectedValue = models.Worse
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLocalityComparator(t *testing.T) {
	locality1 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality2 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality3 := models.Locality{Kind: "GeoRegion", Name: "EU"}
	locality4 := models.Locality{Kind: "GeoRegion", Name: "US"}
	locality5 := models.Locality{Kind: "GeoRegion", Name: "EU"}

	// positive examples
	actualValue := Compare(locality1, locality2)
	expectedValue := models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality3, locality4)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality3, locality5)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(locality1, locality3)
	expectedValue = models.Error
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(locality1, locality3)
	expectedValue = models.Better
	assert.NotEqual(t, expectedValue, actualValue)
}

func TestLocalitiesComparator(t *testing.T) {
	var localities1 []models.Locality
	locality1 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality2 := models.Locality{Kind: "GeoRegion", Name: "EU"}
	localities1 = append(localities1, locality1)
	localities1 = append(localities1, locality2)

	var localities2 []models.Locality
	locality3 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality4 := models.Locality{Kind: "GeoRegion", Name: "US"}
	localities2 = append(localities2, locality3)
	localities2 = append(localities2, locality4)

	var localities3 []models.Locality
	locality5 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality6 := models.Locality{Kind: "GeoRegion", Name: "EU"}
	localities3 = append(localities3, locality5)
	localities3 = append(localities3, locality6)

	var localities4 []models.Locality
	locality7 := models.Locality{Kind: "NuNetRegion", Name: "us-west-1"}
	locality8 := models.Locality{Kind: "GeoRegion", Name: "EU"}
	locality9 := models.Locality{Kind: "GeoCounty", Name: "Belgium"}

	localities4 = append(localities4, locality7)
	localities4 = append(localities4, locality8)
	localities4 = append(localities4, locality9)

	// positive examples
	actualValue := Compare(localities4, localities1)
	expectedValue := models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities1, localities2)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities1, localities3)
	expectedValue = models.Equal
	assert.Equal(t, expectedValue, actualValue)

	actualValue = Compare(localities2, localities3)
	expectedValue = models.Worse
	assert.Equal(t, expectedValue, actualValue)

	// negative examples
	actualValue = Compare(localities1, localities2)
	expectedValue = models.Equal
	assert.NotEqual(t, expectedValue, actualValue)
}
