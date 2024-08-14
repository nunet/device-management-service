package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// OnboardingParamsRepository represents a repository for CRUD operations on OnboardingConfig entity.
type OnboardingParamsRepository interface {
	GenericEntityRepository[types.OnboardingConfig]
}
