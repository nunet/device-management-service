package backend

import (
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/types"
)

type Resources struct{}

func (r *Resources) GetTotalProvisioned() *types.Provisioned {
	return onboarding.GetTotalProvisioned()
}
