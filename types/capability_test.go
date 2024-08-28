package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityAddSubtract(t *testing.T) {
	cap1 := Capability{
		Executors: []Executor{
			{ExecutorType: ExecutorTypeDocker},
		},
		JobTypes: []JobType{
			BATCH,
		},
		Resources: ExecutionResources{
			CPU: CPU{
				Architecture: "x86",
				Cores:        1,
				ClockSpeedHz: 1000,
			},
			Memory: RAM{
				Size:         1024,
				ClockSpeedHz: 1000,
			},
			Disk: Disk{
				Type: "ssd",
				Size: 100,
			},
			GPUs: []GPU{
				{
					Index:      1,
					Vendor:     GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "RTX4010",
					TotalVRAM:  8024,
					FreeVRAM:   8024,
				},
			},
		},
		Libraries: []Library{
			{Name: "lib1", Constraint: "constraint1", Version: "1.0.0"},
		},
		Localities: []Locality{
			{Kind: "geographic", Name: "zone1"},
		},
		Storage: []Storage{
			{Type: "ssd", Size: 100, Amount: 1},
		},
		Connectivity: Connectivity{
			VPN: false,
			Ports: []int{
				80,
				8080,
				3000,
			},
		},
		Price: []PriceInformation{
			{Currency: "USD", CurrencyPerHour: 20, TotalPerJob: 100, Preference: 0},
		},
		Time: TimeInformation{
			MaxTime:    100,
			Units:      "seconds",
			Preference: 0,
		},
		KYCs: []KYC{
			{Type: "type1", Data: "data1"},
		},
	}

	cap2 := Capability{
		Executors: []Executor{
			{ExecutorType: ExecutorTypeWasm},
		},
		JobTypes: []JobType{
			BATCH,
			RECURRING,
		},
		Resources: ExecutionResources{
			CPU: CPU{
				Architecture: "x86",
				Cores:        1,
				ClockSpeedHz: 1000,
			},
			Memory: RAM{
				Size:         1024,
				ClockSpeedHz: 1000,
			},
			Disk: Disk{
				Type: "ssd",
				Size: 100,
			},
			GPUs: []GPU{
				{
					Index:      1,
					Vendor:     GPUVendorAMDATI,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  8024,
					FreeVRAM:   8024,
				},
			},
		},
		Libraries: []Library{
			{Name: "lib1", Constraint: "constraint1", Version: "1.0.0"},
			{Name: "lib2", Constraint: "constraint2", Version: "2.0.0"},
		},
		Localities: []Locality{
			{Kind: "geographic", Name: "zone1"},
			{Kind: "geographic", Name: "zone2"},
		},
		Storage: []Storage{
			{Type: "ssd", Size: 100, Amount: 1},
			{Type: "hdd", Size: 1000, Amount: 1},
		},
		Connectivity: Connectivity{
			VPN: true,
			Ports: []int{
				80,
				8080,
				3000,
				3001,
				3002,
			},
		},
		Price: []PriceInformation{
			{Currency: "USD", CurrencyPerHour: 20, TotalPerJob: 100, Preference: 0},
			{Currency: "EUR", CurrencyPerHour: 20, TotalPerJob: 100, Preference: 0},
		},
		Time: TimeInformation{
			MaxTime:    100,
			Units:      "seconds",
			Preference: 0,
		},
		KYCs: []KYC{
			{Type: "type1", Data: "data1"},
			{Type: "type2", Data: "data2"},
		},
	}

	require.NoError(t, cap1.Add(cap2))

	assert.Len(t, cap1.Executors, 2)
	assert.Equal(t, ExecutorType(ExecutorTypeDocker), cap1.Executors[0].ExecutorType)
	assert.Equal(t, ExecutorType(ExecutorTypeWasm), cap1.Executors[1].ExecutorType)

	assert.Len(t, cap1.JobTypes, 2)
	assert.Equal(t, BATCH, cap1.JobTypes[0])
	assert.Equal(t, RECURRING, cap1.JobTypes[1])

	assert.Equal(t, uint64(2), cap1.Resources.CPU.Cores)
	assert.Equal(t, uint64(2000), cap1.Resources.CPU.ClockSpeedHz)
	assert.Equal(t, uint64(2048), cap1.Resources.Memory.Size)
	assert.Equal(t, uint64(2000), cap1.Resources.Memory.ClockSpeedHz)
	assert.Equal(t, uint64(200), cap1.Resources.Disk.Size)
	assert.Len(t, cap1.Resources.GPUs, 2)
	assert.Equal(t, uint64(1), cap1.Resources.GPUs[0].Index)
	assert.Equal(t, GPUVendorNvidia, cap1.Resources.GPUs[0].Vendor)
	assert.Equal(t, "AAAA:BB:CC.C", cap1.Resources.GPUs[0].PCIAddress)
	assert.Equal(t, "RTX4010", cap1.Resources.GPUs[0].Model)
	assert.Equal(t, uint64(8024), cap1.Resources.GPUs[0].TotalVRAM)
	assert.Equal(t, uint64(8024), cap1.Resources.GPUs[0].FreeVRAM)
	assert.Equal(t, uint64(1), cap1.Resources.GPUs[1].Index)
	assert.Equal(t, GPUVendorAMDATI, cap1.Resources.GPUs[1].Vendor)
	assert.Equal(t, "AAAA:BB:CC.C", cap1.Resources.GPUs[1].PCIAddress)
	assert.Equal(t, "A100", cap1.Resources.GPUs[1].Model)
	assert.Equal(t, uint64(8024), cap1.Resources.GPUs[1].TotalVRAM)

	assert.Len(t, cap1.Libraries, 2)
	assert.Equal(t, "lib1", cap1.Libraries[0].Name)
	assert.Equal(t, "constraint1", cap1.Libraries[0].Constraint)
	assert.Equal(t, "1.0.0", cap1.Libraries[0].Version)
	assert.Equal(t, "lib2", cap1.Libraries[1].Name)
	assert.Equal(t, "constraint2", cap1.Libraries[1].Constraint)
	assert.Equal(t, "2.0.0", cap1.Libraries[1].Version)

	assert.Len(t, cap1.Localities, 2)
	assert.Equal(t, "geographic", cap1.Localities[0].Kind)
	assert.Equal(t, "zone1", cap1.Localities[0].Name)
	assert.Equal(t, "geographic", cap1.Localities[1].Kind)
	assert.Equal(t, "zone2", cap1.Localities[1].Name)

	assert.Len(t, cap1.Storage, 2)
	assert.Equal(t, "ssd", cap1.Storage[0].Type)
	assert.Equal(t, 200, cap1.Storage[0].Size)
	assert.Equal(t, 2, cap1.Storage[0].Amount)
	assert.Equal(t, "hdd", cap1.Storage[1].Type)
	assert.Equal(t, 1000, cap1.Storage[1].Size)

	assert.Equal(t, true, cap1.Connectivity.VPN)
	assert.Len(t, cap1.Connectivity.Ports, 5)
	assert.Equal(t, 80, cap1.Connectivity.Ports[0])
	assert.Equal(t, 8080, cap1.Connectivity.Ports[1])
	assert.Equal(t, 3000, cap1.Connectivity.Ports[2])
	assert.Equal(t, 3001, cap1.Connectivity.Ports[3])
	assert.Equal(t, 3002, cap1.Connectivity.Ports[4])

	assert.Equal(t, 2, len(cap1.Price))
	assert.Equal(t, "USD", cap1.Price[0].Currency)
	assert.Equal(t, 20, cap1.Price[0].CurrencyPerHour)
	assert.Equal(t, 100, cap1.Price[0].TotalPerJob)
	assert.Equal(t, 0, cap1.Price[0].Preference)
	assert.Equal(t, "EUR", cap1.Price[1].Currency)
	assert.Equal(t, 20, cap1.Price[1].CurrencyPerHour)
	assert.Equal(t, 100, cap1.Price[1].TotalPerJob)
	assert.Equal(t, 0, cap1.Price[1].Preference)

	assert.Equal(t, 200, cap1.Time.MaxTime)

	assert.Equal(t, 2, len(cap1.KYCs))
	assert.Equal(t, "type1", cap1.KYCs[0].Type)
	assert.Equal(t, "data1", cap1.KYCs[0].Data)
	assert.Equal(t, "type2", cap1.KYCs[1].Type)
	assert.Equal(t, "data2", cap1.KYCs[1].Data)

	require.NoError(t, cap1.Subtract(cap2))

	// Executors uneffected by Subtract
	assert.Len(t, cap1.Executors, 2)
	assert.Equal(t, ExecutorType(ExecutorTypeDocker), cap1.Executors[0].ExecutorType)
	assert.Equal(t, ExecutorType(ExecutorTypeWasm), cap1.Executors[1].ExecutorType)

	// JobTypes uneffected by Subtract
	assert.Len(t, cap1.JobTypes, 2)
	assert.Equal(t, BATCH, cap1.JobTypes[0])
	assert.Equal(t, RECURRING, cap1.JobTypes[1])

	// Resources affected by Subtract
	assert.Equal(t, uint64(1), cap1.Resources.CPU.Cores)
	assert.Equal(t, uint64(1000), cap1.Resources.CPU.ClockSpeedHz)
	assert.Equal(t, uint64(1024), cap1.Resources.Memory.Size)
	assert.Equal(t, uint64(1000), cap1.Resources.Memory.ClockSpeedHz)
	assert.Equal(t, uint64(100), cap1.Resources.Disk.Size)
	assert.Len(t, cap1.Resources.GPUs, 1)
	assert.Equal(t, uint64(1), cap1.Resources.GPUs[0].Index)
	assert.Equal(t, GPUVendorNvidia, cap1.Resources.GPUs[0].Vendor)
	assert.Equal(t, "AAAA:BB:CC.C", cap1.Resources.GPUs[0].PCIAddress)
	assert.Equal(t, "RTX4010", cap1.Resources.GPUs[0].Model)
	assert.Equal(t, uint64(8024), cap1.Resources.GPUs[0].TotalVRAM)

	// Libraries uneffected by Subtract
	assert.Len(t, cap1.Libraries, 2)
	assert.Equal(t, "lib1", cap1.Libraries[0].Name)
	assert.Equal(t, "constraint1", cap1.Libraries[0].Constraint)
	assert.Equal(t, "1.0.0", cap1.Libraries[0].Version)
	assert.Equal(t, "lib2", cap1.Libraries[1].Name)
	assert.Equal(t, "constraint2", cap1.Libraries[1].Constraint)
	assert.Equal(t, "2.0.0", cap1.Libraries[1].Version)

	// Localities uneffected by Subtract
	assert.Len(t, cap1.Localities, 2)
	assert.Equal(t, "geographic", cap1.Localities[0].Kind)
	assert.Equal(t, "zone1", cap1.Localities[0].Name)
	assert.Equal(t, "geographic", cap1.Localities[1].Kind)
	assert.Equal(t, "zone2", cap1.Localities[1].Name)

	// Storage affected by Subtract
	assert.Len(t, cap1.Storage, 1)
	assert.Equal(t, "ssd", cap1.Storage[0].Type)
	assert.Equal(t, 100, cap1.Storage[0].Size)

	// Connectivity affected by Subtract
	assert.Equal(t, true, cap1.Connectivity.VPN)
	assert.Len(t, cap1.Connectivity.Ports, 0)

	// Price uneffected by Subtract
	assert.Equal(t, 2, len(cap1.Price))
	assert.Equal(t, "USD", cap1.Price[0].Currency)
	assert.Equal(t, 20, cap1.Price[0].CurrencyPerHour)
	assert.Equal(t, 100, cap1.Price[0].TotalPerJob)
	assert.Equal(t, 0, cap1.Price[0].Preference)
	assert.Equal(t, "EUR", cap1.Price[1].Currency)
	assert.Equal(t, 20, cap1.Price[1].CurrencyPerHour)
	assert.Equal(t, 100, cap1.Price[1].TotalPerJob)
	assert.Equal(t, 0, cap1.Price[1].Preference)

	// Time effected by Subtract
	assert.Equal(t, 100, cap1.Time.MaxTime)

	// KYCs uneffected by Subtract
	assert.Equal(t, 2, len(cap1.KYCs))
	assert.Equal(t, "type1", cap1.KYCs[0].Type)
	assert.Equal(t, "data1", cap1.KYCs[0].Data)
	assert.Equal(t, "type2", cap1.KYCs[1].Type)
	assert.Equal(t, "data2", cap1.KYCs[1].Data)

	require.NoError(t, cap1.Subtract(cap1))
	assert.Equal(t, uint64(0), cap1.Resources.CPU.Cores)
	assert.Equal(t, uint64(0), cap1.Resources.Memory.Size)
	assert.Equal(t, uint64(0), cap1.Resources.Disk.Size)
	assert.Len(t, cap1.Resources.GPUs, 0)
}
