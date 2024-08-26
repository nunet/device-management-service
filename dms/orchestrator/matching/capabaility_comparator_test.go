package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestCapabilityComparator(t *testing.T) {
	cp := types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 4, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 32},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.SSD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080, 9091},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
			{
				Currency:        "EUR",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{
			{
				Type: "KYC1",
				Data: "data1",
			},
			{
				Type: "KYC2",
				Data: "data2",
			},
		},
	}

	sp := types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 2, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 32},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.HDD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{
			{
				Type: "KYC1",
				Data: "data1",
			},
		},
	}

	result := CapabilityComparator(cp, sp)
	expectedValue := types.Better
	assert.Equal(t, expectedValue, result)

	sp = types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 4, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 16},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.SSD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080, 9091},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
			{
				Currency:        "EUR",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{
			{
				Type: "KYC1",
				Data: "data1",
			},
			{
				Type: "KYC2",
				Data: "data2",
			},
		},
	}

	result = CapabilityComparator(cp, sp)
	expectedValue = types.Equal
	assert.Equal(t, expectedValue, result)

	sp = types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 6, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 32},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.SSD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080, 9091},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
			{
				Currency:        "EUR",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{
			{
				Type: "KYC1",
				Data: "data1",
			},
			{
				Type: "KYC2",
				Data: "data2",
			},
		},
	}

	result = CapabilityComparator(cp, sp)
	expectedValue = types.Worse
	assert.Equal(t, expectedValue, result)

	sp = types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 4, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 32},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.SSD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080, 9091},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
			{
				Currency:        "EUR",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{},
	}

	result = CapabilityComparator(cp, sp)
	expectedValue = types.Better
	assert.Equal(t, expectedValue, result)

	sp = types.Capability{
		Executors: types.Executors{
			{
				ExecutorType: types.ExecutorTypeDocker,
			},
		},
		JobTypes: types.JobTypes{types.LONGRUNNING},
		Resources: types.ExecutionResources{
			CPU:    types.CPU{Cores: 4, Architecture: "arm", ClockSpeedHz: 1},
			Memory: types.RAM{Size: 32},
			Disk:   types.Disk{Size: 64},
			GPUs: []types.GPU{
				{
					Index:      0,
					Vendor:     types.GPUVendorNvidia,
					PCIAddress: "AAAA:BB:CC.C",
					Model:      "A100",
					TotalVRAM:  16,
				},
			},
		},
		Libraries: []types.Library{
			{
				Name:       "TensorFlow",
				Constraint: "=",
				Version:    "2.4.0",
			},
			{
				Name:       "PyTorch",
				Constraint: "=",
				Version:    "1.7.0",
			},
		},
		Localities: []types.Locality{
			{
				Kind: "geographic",
				Name: "EU",
			},
		},
		Storage: []types.Storage{
			{
				Type:   types.SSD_STORAGE_TYPE,
				Size:   256,
				Amount: 1,
			},
		},
		Connectivity: types.Connectivity{
			VPN:   true,
			Ports: []int{80, 8080, 9091},
		},
		Price: []types.PriceInformation{
			{
				Currency:        "USD",
				CurrencyPerHour: 2,
				TotalPerJob:     100,
				Preference:      0,
			},
			{
				Currency:        "EUR",
				CurrencyPerHour: 5,
				TotalPerJob:     100,
				Preference:      0,
			},
		},
		Time: types.TimeInformation{
			Units:      "seconds",
			MaxTime:    60 * 60 * 60,
			Preference: 0,
		},
		KYCs: []types.KYC{
			{
				Type: "KYC1",
				Data: "data1",
			},
		},
	}
	cp.KYCs = []types.KYC{}

	result = CapabilityComparator(cp, sp)
	expectedValue = types.Error
	assert.Equal(t, expectedValue, result)
}
