package provider

import (
	"errors"

	"gitlab.com/nunet/device-management-service/types"
)

// Plan represents a VM or server plan
type Plan struct {
	ID          string
	Name        string
	Description string
	CPU         int
	MemoryMB    int
	DiskGB      int

	GPUCount  int
	GPUModel  string
	GPUVRAMGB int

	Region   string
	PriceUSD float64
}

// Server represents a provisioned server or VM.
type Server struct {
	ID       string
	Name     string
	IP       string
	PlanID   string
	Status   string // e.g., "running", "stopped", "deleted"
	Region   string
	Created  int64 // Unix timestamp
	Metadata map[string]string

	PeerID     string
	ListenAddr string
}

// SelectMatchingPlan based on available plans returns the best plan for the given resources
// helper function
func SelectMatchingPlan(plans []Plan, target types.Resources) (*Plan, error) {
	for _, v := range plans {
		res := convertPlanToResources(v)
		comp, err := res.Compare(target)
		if err != nil {
			continue
		}

		if comp == types.Better || comp == types.Equal {
			return &v, nil
		}
	}

	return nil, errors.New("can't match hardware requirements")
}

// convertPlanToResources returns resources
func convertPlanToResources(plan Plan) types.Resources {
	var gpus types.GPUs
	if plan.GPUCount > 0 {
		for i := 0; i < plan.GPUCount; i++ {
			gpus = append(gpus, types.GPU{
				Index:  i,
				Vendor: types.ParseGPUVendor(plan.GPUModel),
				Model:  plan.GPUModel,
				VRAM:   types.ConvertGBToBytes(uint64(plan.GPUVRAMGB)),
			})
		}
	}

	return types.Resources{
		CPU: types.CPU{
			Cores: float32(plan.CPU),
		},
		RAM: types.RAM{
			Size: types.ConvertMibToBytes(uint64(plan.MemoryMB)),
		},
		Disk: types.Disk{
			Size: types.ConvertGBToBytes(uint64(plan.DiskGB)),
		},
		GPUs: gpus,
	}
}
