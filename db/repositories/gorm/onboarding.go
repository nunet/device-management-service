package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardingParamsGORM struct {
	repositories.GenericEntityRepository[types.OnboardingConfig]
}

func NewOnboardingParams(db *gorm.DB) repositories.OnboardingParams {
	return &OnboardingParamsGORM{
		NewGenericEntityRepository[types.OnboardingConfig](db),
	}
}
