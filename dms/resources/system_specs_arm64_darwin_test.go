package resources

import (
	"github.com/shoenig/go-m1cpu"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArm64DarwinNewSystemSpecs(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	assert.NotNil(t, systemSpecs)
}

func TestArm64DarwinSystemSpecs_GetTotalMemory(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	totalMemory, err := systemSpecs.GetTotalMemory()
	assert.NoError(t, err)
	assert.Greater(t, totalMemory, uint64(0))
}

func TestArm64DarwinSystemSpecs_GetTotalStorage(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	totalStorage, err := systemSpecs.GetTotalStorage()
	assert.NoError(t, err)
	assert.Greater(t, totalStorage, uint64(0))
}

func TestArm64DarwinSystemSpecs_GetCPUInfo(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	cpuInfo, err := systemSpecs.GetCPUInfo()
	assert.NoError(t, err)
	assert.Greater(t, cpuInfo.NumCores, uint64(0))
	assert.Greater(t, cpuInfo.MHzPerCore, float64(0))
	assert.Equal(t, getArm64DarwinExpectedCompute(t), cpuInfo.Compute)
}

func TestArm64DarwinSystemSpecs_GetProvisionedResources(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	resources, err := systemSpecs.GetProvisionedResources()
	assert.NoError(t, err)
	assert.Equal(t, getArm64DarwinExpectedCompute(t), resources.CPU)
	assert.Greater(t, resources.RAM, uint64(0))
	assert.Greater(t, resources.Disk, uint64(0))
}

func TestArm64DarwinSystemSpecs_GetGPUVendors(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	gpuVendors, err := systemSpecs.GetGPUVendors()
	assert.NoError(t, err)
	assert.Empty(t, gpuVendors) // GPUs are not supported on Darwin yet
}

func TestArm64DarwinSystemSpecs_GetGPUs(t *testing.T) {
	t.Parallel()

	if !m1cpu.IsAppleSilicon() {
		t.Skip("wrong cpu type")
	}

	systemSpecs := newSystemSpecs()
	gpus, err := systemSpecs.GetGPUs()
	assert.NoError(t, err)
	assert.Empty(t, gpus) // GPUs are not supported on Darwin yet
}

// getArm64DarwinExpectedCompute returns the expected compute value for an arm64 darwin system
func getArm64DarwinExpectedCompute(t *testing.T) float64 {
	t.Helper()

	return float64(m1cpu.ECoreCount())*m1cpu.ECoreGHz()*1000 + float64(m1cpu.PCoreCount())*m1cpu.PCoreGHz()*1000
}
