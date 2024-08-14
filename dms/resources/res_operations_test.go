package resources

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ------------ Mocking data ------------ //

var (
	testOnceResOperations       sync.Once
	testDBReadOnlyResOperations *gorm.DB
)

var mockFreeRes1 = types.FreeResources{
	TotCpuHz: 6000,
	Ram:      5000,
	Disk:     600,
}

var mockFreeRes2 = types.FreeResources{
	TotCpuHz: 1000,
	Ram:      1000,
	Disk:     500,
}

var mockAvailableRes1 = types.AvailableResources{
	TotCpuHz: 2000,
	Ram:      1500,
	Disk:     800,
	CpuHz:    100,
	Vcpu:     5,
}

// ------------ Tests ------------ //

// TestSubtractFromAvailableRes is an unit test which tests subtractFromAvailableRes()
// in which communicates with the in-memory DB with mock data.
func TestSubtractFromAvailableRes(t *testing.T) {
	tests := []struct {
		resUsage types.FreeResources
		want     types.FreeResources
		wantErr  bool
	}{
		{
			// If AvailableResources is less than the resources usage, should return error
			resUsage: mockFreeRes1,
			want:     types.FreeResources{},
			wantErr:  true,
		},
		{
			// Normal case, subtracting AvailableResources from resources usage
			resUsage: mockFreeRes2,
			want:     types.FreeResources{TotCpuHz: 1000, Ram: 500, Disk: 300, Vcpu: 10},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		err := setupTestDBResOperations()
		if err != nil {
			t.Fatalf("failed to setup test db: %v", err)
		}

		got, err := subtractFromAvailableRes(testDBReadOnlyResOperations, tt.resUsage)
		if (err != nil) != tt.wantErr {
			t.Errorf("subtractFromAvailableRes() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("subtractFromAvailableRes() = %v, want %v", got, tt.want)
		}
	}
}

// TestSubResourcesUsage tests subResourcesUsage() with table tests
func TestSubResourcesUsage(t *testing.T) {
	tests := []struct {
		r1, r2, want types.FreeResources
		wantErr      bool
	}{
		{
			// Normal case, subtraction resulting in positive values
			r1:      mockFreeRes1,
			r2:      mockFreeRes2,
			want:    types.FreeResources{TotCpuHz: 5000, Ram: 4000, Disk: 100},
			wantErr: false,
		},
		{
			// Subtraction of resources resulting in negative values should return error
			r1:      mockFreeRes2,
			r2:      mockFreeRes1,
			want:    types.FreeResources{TotCpuHz: -5000, Ram: -4000, Disk: -100},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := subResourcesUsage(tt.r1, tt.r2)
		if (err != nil) != tt.wantErr {
			t.Errorf("SubResourcesUsage() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SubResourcesUsage() = %v, want %v", got, tt.want)
		}
	}
}

// TestAddResourcesUsage tests addResourcesUsage() with table tests (mocked structs)
func TestAddResourcesUsage(t *testing.T) {
	tests := []struct {
		r1, r2, want types.FreeResources
	}{
		{
			// When adding to an empty struct, the final value must not change
			r1:   types.FreeResources{},
			r2:   mockFreeRes1,
			want: mockFreeRes1,
		},
		{
			// Normal addition case
			r1:   mockFreeRes1,
			r2:   mockFreeRes2,
			want: types.FreeResources{TotCpuHz: 7000, Ram: 6000, Disk: 1100},
		},
	}

	for _, tt := range tests {
		got := addResourcesUsage(tt.r1, tt.r2)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Expected ResourcesUsage to be %+v, but got %+v", tt.want, got)
		}
	}
}

// TestResultsInNegativeValuesFreeRes tests resultsInNegativeValuesFreeRes()
// with table tests (mocked structs)
func TestResultsInNegativeValuesFreeRes(t *testing.T) {
	tests := []struct {
		r1, r2  types.FreeResources
		wantErr bool
	}{
		{
			// subtraction results in positive value
			r1:      mockFreeRes1,
			r2:      mockFreeRes2,
			wantErr: false,
		},
		{
			// subtraction results in negative value
			r1:      mockFreeRes2,
			r2:      mockFreeRes1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		err := resultsInNegativeValuesFreeRes(&tt.r1, &tt.r2)
		if (err != nil) != tt.wantErr {
			t.Errorf("resultsInNegativeValuesFreeRes() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

// TestResultsInNegativeValuesAvailableRes tests resultsInNegativeValuesAvailableRes()
// with table tests (mocked structs)
func TestResultsInNegativeValuesAvailableRes(t *testing.T) {
	tests := []struct {
		r1      types.AvailableResources
		r2      types.FreeResources
		wantErr bool
	}{
		{
			// subtraction results in positive value
			r1:      mockAvailableRes1,
			r2:      mockFreeRes2,
			wantErr: false,
		},
		{
			// subtraction results in negative value, function must return error
			r1:      mockAvailableRes1,
			r2:      mockFreeRes1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		err := resultsInNegativeValuesAvailableRes(tt.r1, tt.r2)
		if (err != nil) != tt.wantErr {
			t.Errorf("resultsInNegativeValuesAvailableRes() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

// ------------ Settting up in-memory DB ------------ //

// setupTestDBResOperations creates and configures an in-memory DB with fake data
// for the tests related to the math operations between resources's values.
func setupTestDBResOperations() error {
	var err error
	testOnceResOperations.Do(func() {
		dbName := fmt.Sprintf("file:testDBReadOnlyResOperations?mode=memory&cache=shared")

		// Create a new in-memory SQLite database
		testDB, errLocal := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
		if errLocal != nil {
			return
		}

		errLocal = testDB.AutoMigrate(
			&types.AvailableResources{},
		)

		if errLocal != nil {
			err = errLocal
			return
		}

		errLocal = insertTestDBResOperations(testDB)
		if errLocal != nil {
			err = errLocal
			return
		}

		testDBReadOnlyResOperations = testDB
	})
	return err
}

// insertTestDBResOperations inserts fake data into the in-memory DB for tests
func insertTestDBResOperations(db *gorm.DB) error {
	if err := db.Create(&mockAvailableRes1).Error; err != nil {
		return err
	}

	return nil
}
