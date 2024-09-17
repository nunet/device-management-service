package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// OnboardingParams represents a repository for CRUD operations on OnboardingConfig entity.
type OnboardingParams interface {
	GenericEntityRepository[types.OnboardingConfig]
}
