package resources

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"

	"gorm.io/gorm"
)

// negativeValueError is a type struct used to return a custom error for the
// resultsInNegativeValues functions
type negativeValueError struct {
	fieldName string
	r1        int
	r2        int
}

func (e *negativeValueError) Error() string {
	return fmt.Sprintf("Error: %s subtraction results in negative values. (%d - %d)", e.fieldName, e.r1, e.r2)
}

// Note: Despite FreeResources model's main goal being represent the machine's free resources,
// we're using its model to represent general resources usage for all operations between
// resources-related structs (a REFACTORING is needed, we need to simplify the resource-relate types.

// addResourcesUsage returns the sum between each field of two FreeResources
// structs. Use it when increasing resources usage based on two resources-usage
// structs.
func addResourcesUsage(r1, r2 types.FreeResources) types.FreeResources {
	return types.FreeResources{
		TotCpuHz: r1.TotCpuHz + r2.TotCpuHz,
		Ram:      r1.Ram + r2.Ram,
		Disk:     r1.Disk + r2.Disk,
	}
}

// subResourcesUsage returns the result of subtracting each field of the second FreeResources
// struct from the corresponding field of the first. It is used when decreasing resources usage
// based on two resources-usage structs.
func subResourcesUsage(r1, r2 types.FreeResources) (types.FreeResources, error) {
	// We don't need this function if when refactoring the resources types.
	// we use unint instead of int
	if err := resultsInNegativeValuesFreeRes(&r1, &r2); err != nil {
		return types.FreeResources{
			TotCpuHz: r1.TotCpuHz - r2.TotCpuHz,
			Ram:      r1.Ram - r2.Ram,
			Disk:     r1.Disk - r2.Disk,
		}, fmt.Errorf("Subtraction of resources would result in negative values, Error: %w", err)
	}

	return types.FreeResources{
		TotCpuHz: r1.TotCpuHz - r2.TotCpuHz,
		Ram:      r1.Ram - r2.Ram,
		Disk:     r1.Disk - r2.Disk,
	}, nil
}

// resultsInNegativeValuesFreeRes checks if any subtraction operation between the
// fields of two FreeResources structs results in a negative value. It returns an error if so,
// since resource values cannot be negative.
func resultsInNegativeValuesFreeRes(r1, r2 *types.FreeResources) error {
	// We don't need this function if when refactoring the resources types.
	// we use unint instead of int
	if r1.TotCpuHz-r2.TotCpuHz < 0 {
		return &negativeValueError{fieldName: "TotCpuHz", r1: r1.TotCpuHz, r2: r2.TotCpuHz}
	}
	if r1.Ram-r2.Ram < 0 {
		return &negativeValueError{fieldName: "Ram", r1: r1.Ram, r2: r2.Ram}
	}
	if r1.Disk-r2.Disk < 0 {
		return &negativeValueError{fieldName: "Disk", r1: int(r1.Disk), r2: int(r2.Disk)}
	}
	return nil
}

// resultsInNegativeValuesAvailableRes checks if any subtraction operation between the
// fields of two different resource-related structs (AvailableResources and FreeResources) results in a negative value.
// It returns an error if so, since resource values cannot be negative.
func resultsInNegativeValuesAvailableRes(r1 types.AvailableResources, r2 types.FreeResources) error {
	// Basically duplicate function just because we have different models
	// We will remove that when simplifying resources structs
	if r1.TotCpuHz-r2.TotCpuHz < 0 {
		return &negativeValueError{fieldName: "TotCpuHz", r1: r1.TotCpuHz, r2: r2.TotCpuHz}
	}
	if r1.Ram-r2.Ram < 0 {
		return &negativeValueError{fieldName: "Ram", r1: r1.Ram, r2: r2.Ram}
	}
	if r1.Disk-r2.Disk < 0 {
		return &negativeValueError{fieldName: "Disk", r1: int(r1.Disk), r2: int(r2.Disk)}
	}
	return nil
}

// subtractFromAvailableRes returns the difference between AvailableResources usage and
// FreeResources struct.
func subtractFromAvailableRes(gormDB *gorm.DB, resourcesUsage types.FreeResources,
) (types.FreeResources, error) {
	var freeRes types.FreeResources

	availableRes, err := GetAvailableResources(gormDB)
	if err != nil {
		return freeRes,
			fmt.Errorf("Couldn't query AvailableResources for subtraction, Error: %w", err)
	}

	if err := resultsInNegativeValuesAvailableRes(availableRes, resourcesUsage); err != nil {
		return freeRes,
			fmt.Errorf("Subtraction of resources results in negative values, Error: %w", err)
	}

	freeRes.TotCpuHz = availableRes.TotCpuHz - resourcesUsage.TotCpuHz
	freeRes.Vcpu = freeRes.TotCpuHz / int(availableRes.CpuHz)
	freeRes.Ram = availableRes.Ram - resourcesUsage.Ram
	freeRes.Disk = float64(availableRes.Disk) - resourcesUsage.Disk
	freeRes.NTXPricePerMinute = availableRes.NTXPricePerMinute

	return freeRes, nil
}
