package resources

import (
	"testing"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shoenig/go-m1cpu"
	"github.com/stretchr/testify/assert"
)

func TestNewAmd64DarwinSystemSpecs(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	assert.NotNil(t, systemSpecs)
}

func TestAmd64DarwinSystemSpecs_GetTotalMemory(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	totalMemory, err := systemSpecs.GetTotalMemory()
	assert.NoError(t, err)
	assert.Greater(t, totalMemory, uint64(0))
}

func TestAmd64DarwinSystemSpecs_GetTotalStorage(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	totalStorage, err := systemSpecs.GetTotalStorage()
	assert.NoError(t, err)
	assert.Greater(t, totalStorage, uint64(0))
}

func TestAmd64DarwinSystemSpecs_GetCPUInfo(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	cpuInfo, err := systemSpecs.GetCPUInfo()
	assert.NoError(t, err)
	expectedCPUInfo := getAmd64ExpectedCPUInfo(t)

	assert.Equal(t, expectedCPUInfo.NumCores, cpuInfo.NumCores)
}

func TestAmd64DarwinSystemSpecs_GetProvisionedResources(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	resources, err := systemSpecs.GetProvisionedResources()
	assert.NoError(t, err)
	expectedCPUInfo := getAmd64ExpectedCPUInfo(t)
	assert.Equal(t, expectedCPUInfo.Compute, resources.CPU)
	assert.Greater(t, resources.RAM, uint64(0))
	assert.Greater(t, resources.Disk, uint64(0))
}

func TestAmd64DarwinSystemSpecs_GetGPUVendors(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	gpuVendors, err := systemSpecs.GetGPUVendors()
	assert.NoError(t, err)
	assert.Empty(t, gpuVendors) // GPUs are not supported on Darwin yet
}

func TestAmd64DarwinSystemSpecs_GetGPUs(t *testing.T) {
	t.Parallel()

	if m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	gpus, err := systemSpecs.GetGPUs()
	assert.NoError(t, err)
	assert.Empty(t, gpus) // GPUs are not supported on Darwin yet
}

func getAmd64ExpectedCPUInfo(t *testing.T) types.CPUInfo {
	t.Helper()

	cpus, err := cpu.Info()
	assert.NoError(t, err)

	var (
		totalCompute float64
		totalCores   uint64
	)
	for c := range cpus {
		totalCompute += float64(cpus[c].Cores) * cpus[c].Mhz
		totalCores += uint64(cpus[c].Cores)
	}

	cpuInfo := types.CPUInfo{
		NumCores:   totalCores,
		MHzPerCore: cpus[0].Mhz,
		Compute:    totalCompute,
	}
	return cpuInfo
}
