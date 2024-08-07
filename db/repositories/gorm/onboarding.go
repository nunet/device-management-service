package repositories_gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/models"
)

type OnboardingParamsRepositoryGORM struct {
	repositories.GenericEntityRepository[models.OnboardingConfig]
}

func NewOnboardingParamsRepository(db *gorm.DB) repositories.OnboardingParamsRepository {
	return &OnboardingParamsRepositoryGORM{
		NewGenericEntityRepository[models.OnboardingConfig](db),
	}
}
