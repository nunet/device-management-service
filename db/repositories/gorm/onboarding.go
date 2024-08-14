package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardingParamsRepositoryGORM struct {
	repositories.GenericEntityRepository[types.OnboardingConfig]
}

func NewOnboardingParamsRepository(db *gorm.DB) repositories.OnboardingParamsRepository {
	return &OnboardingParamsRepositoryGORM{
		NewGenericEntityRepository[types.OnboardingConfig](db),
	}
}
