package repositories

import (
	"gitlab.com/nunet/device-management-service/types"
)

// OnboardingConfig represents a repository for CRUD operations on OnboardingConfig entity.
type OnboardingConfig interface {
	GenericEntityRepository[types.OnboardingConfig]
}
