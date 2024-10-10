package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardingConfigGORM struct {
	repositories.GenericEntityRepository[types.OnboardingConfig]
}

func NewOnboardingConfig(db *gorm.DB) repositories.OnboardingConfig {
	return &OnboardingConfigGORM{
		NewGenericEntityRepository[types.OnboardingConfig](db),
	}
}
