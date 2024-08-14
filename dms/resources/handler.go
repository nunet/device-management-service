package resources

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func GetFreeResource(ctx context.Context) (*types.FreeResources, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("URL", "/telemetry/free"))

	err := CalcFreeResAndUpdateDB()
	if err != nil {
		return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	}

	var free types.FreeResources
	res := db.DB.WithContext(ctx).Find(&free)
	if res.Error != nil {
		return nil, fmt.Errorf("could not find free resources table: %w", res.Error)
	} else if res.RowsAffected == 0 {
		return nil, fmt.Errorf("no rows were affected")
	}
	return &free, nil
}

func updateDBFreeResources(freeRes types.FreeResources) error {
	freeRes.ID = "1" // Enforce unique record for a given machine

	var freeResourcesModel types.FreeResources
	if res := db.DB.Find(&freeResourcesModel); res.RowsAffected == 0 {
		result := db.DB.Create(&freeRes)
		if result.Error != nil {
			return result.Error
		}
	} else {
		result := db.DB.Model(&types.FreeResources{}).Where("id = ?", 1).Select("*").Updates(&freeRes)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func getServiceResourcesRequirements(gormDB *gorm.DB) (map[string]types.ServiceResourceRequirements, error) {
	var serviceResRequirements []types.ServiceResourceRequirements
	result := gormDB.Find(&serviceResRequirements)
	if result.Error != nil {
		return nil, fmt.Errorf("unable to query resource requirements - %v", result.Error)
	}

	mappedServicesResRequirements := make(map[string]types.ServiceResourceRequirements)
	for _, rr := range serviceResRequirements {
		mappedServicesResRequirements[rr.ID] = rr
	}

	return mappedServicesResRequirements, nil
}

func GetFreeResources() (types.FreeResources, error) {
	var freeResource types.FreeResources
	if res := db.DB.Find(&freeResource); res.RowsAffected == 0 {
		return freeResource, res.Error
	}
	return freeResource, nil
}

func GetAvailableResources(gormDB *gorm.DB) (types.AvailableResources, error) {
	var availableRes types.AvailableResources
	if res := gormDB.Find(&availableRes); res.RowsAffected == 0 {
		return availableRes, res.Error
	}
	return availableRes, nil
}
