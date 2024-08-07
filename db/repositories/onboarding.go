package repositories

import (
	"gitlab.com/nunet/device-management-service/models"
)

// OnboardingParamsRepository represents a repository for CRUD operations on OnboardingConfig entity.
type OnboardingParamsRepository interface {
	GenericEntityRepository[models.OnboardingConfig]
}
