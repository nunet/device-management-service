package resources

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func GetFreeResource(ctx context.Context) (*models.FreeResources, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("URL", "/telemetry/free"))

	err := CalcFreeResAndUpdateDB()
	if err != nil {
		return nil, fmt.Errorf("could not calculate free resources and update database: %w", err)
	}

	var free models.FreeResources
	res := db.DB.WithContext(ctx).Find(&free)
	if res.Error != nil {
		return nil, fmt.Errorf("could not find free resources table: %w", res.Error)
	} else if res.RowsAffected == 0 {
		return nil, fmt.Errorf("no rows were affected")
	}
	return &free, nil
}

func updateDBFreeResources(freeRes models.FreeResources) error {
	freeRes.ID = "1" // Enforce unique record for a given machine

	var freeResourcesModel models.FreeResources
	if res := db.DB.Find(&freeResourcesModel); res.RowsAffected == 0 {
		result := db.DB.Create(&freeRes)
		if result.Error != nil {
			return result.Error
		}
	} else {
		result := db.DB.Model(&models.FreeResources{}).Where("id = ?", 1).Select("*").Updates(&freeRes)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func getServiceResourcesRequirements(gormDB *gorm.DB) (map[string]models.ServiceResourceRequirements, error) {
	var serviceResRequirements []models.ServiceResourceRequirements
	result := gormDB.Find(&serviceResRequirements)
	if result.Error != nil {
		return nil, fmt.Errorf("unable to query resource requirements - %v", result.Error)
	}

	mappedServicesResRequirements := make(map[string]models.ServiceResourceRequirements)
	for _, rr := range serviceResRequirements {
		mappedServicesResRequirements[rr.ID] = rr
	}

	return mappedServicesResRequirements, nil
}

func GetFreeResources() (models.FreeResources, error) {
	var freeResource models.FreeResources
	if res := db.DB.Find(&freeResource); res.RowsAffected == 0 {
		return freeResource, res.Error
	}
	return freeResource, nil
}

func GetAvailableResources(gormDB *gorm.DB) (models.AvailableResources, error) {
	var availableRes models.AvailableResources
	if res := gormDB.Find(&availableRes); res.RowsAffected == 0 {
		return availableRes, res.Error
	}
	return availableRes, nil
}
