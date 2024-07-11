package resources

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gitlab.com/nunet/device-management-service/models"

	"github.com/shirou/gopsutil/cpu"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ------------ Mocking data ------------ //

var (
	testOnceCalcRes       sync.Once
	testDBReadOnlyCalcRes *gorm.DB
)

var vm1 = models.VirtualMachine{
	VCPUCount:  10,
	MemSizeMib: 1000,
	State:      "running",
}

var availableResTable = models.AvailableResources{
	TotCpuHz: 15000,
	Ram:      11000,
	CpuHz:    100,
	Disk:     2000,
}

var s1 = models.Services{
	ResourceRequirements: 1,
	JobStatus:            "running",
}

var serviceResReqs1 = models.ServiceResourceRequirements{
	BaseDBModel: models.BaseDBModel{
		ID: "1",
	},
	CPU: 3000,
	RAM: 1000,
	HDD: 500,
}

var s2 = models.Services{
	ResourceRequirements: 2,
	JobStatus:            "running",
}

var serviceResReqs2 = models.ServiceResourceRequirements{
	BaseDBModel: models.BaseDBModel{
		ID: "2",
	},
	CPU: 4000,
	RAM: 2000,
}

func mockCpuInfo() []cpu.InfoStat {
	return []cpu.InfoStat{
		{Mhz: 100},
	}
}

// ------------ Tests ------------ //

// TestCalcFreeResources is an integration test which tests
// the whole calcFreeResources() function with all its called
// functions. It uses an in-memory DB with mocked data
func TestCalcFreeResources(t *testing.T) {
	err := setupTestDBCalcFree()
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	cpuInfo := mockCpuInfo()

	freeRes, err := calcFreeResources(testDBReadOnlyCalcRes, cpuInfo)
	if err != nil {
		t.Fatalf("calcFreeResources failed with error: %v", err)
	}

	expectedFreeRes := models.FreeResources{
		TotCpuHz: 7000,
		Ram:      7000,
		Disk:     2000,
		Vcpu:     70,
	}

	if !reflect.DeepEqual(freeRes, expectedFreeRes) {
		t.Fatalf("expected %v but got %v", expectedFreeRes, freeRes)
	}
}

// TestCalcUsedResourcesConts is an unit test which tests the calcUsedResourcesConts()
// with []models.Services and models.ServiceResourceRequirements being mocked structs
func TestCalcUsedResourcesConts(t *testing.T) {
	services := []models.Services{s1, s2}
	requirements := map[string]models.ServiceResourceRequirements{
		"1": serviceResReqs1,
		"2": serviceResReqs2,
	}

	result := calcUsedResourcesConts(services, requirements)
	expected := models.FreeResources{
		TotCpuHz: serviceResReqs1.CPU + serviceResReqs2.CPU,
		Ram:      serviceResReqs1.RAM + serviceResReqs2.RAM,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Expected %v, got %v", expected, result)
	}
}

// TestCalcUsedResourcesVMs is an unit test which tests the calcUsedResourcesVMs()
// with []models.VirtualMachine being a mocked struct
func TestCalcUsedResourcesVMs(t *testing.T) {
	vms := []models.VirtualMachine{vm1}

	cpuInfo := mockCpuInfo()
	result := calcUsedResourcesVMs(vms, cpuInfo)

	expected := models.FreeResources{
		Ram:      vm1.MemSizeMib,
		TotCpuHz: vm1.VCPUCount * int(cpuInfo[0].Mhz),
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Expected %v, got %v", expected, result)
	}
}

// ------------ Settting up in-memory DB ------------ //

// setupTestDBCalcFree creates and configures an in-memory DB with fake data
// for the tests related to the calcFreeResources.
func setupTestDBCalcFree() error {
	var err error
	testOnceCalcRes.Do(func() {
		dbName := fmt.Sprintf("file:testDBReadOnlyCalcRes?mode=memory&cache=shared")

		// Create a new in-memory SQLite database
		testDB, errLocal := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
		if errLocal != nil {
			return
		}

		errLocal = testDB.AutoMigrate(
			&models.AvailableResources{},
			&models.ServiceResourceRequirements{},
			&models.VirtualMachine{},
			&models.Services{},
		)
		if errLocal != nil {
			err = errLocal
			return
		}

		errLocal = insertTestDBCalcRes(testDB)
		if errLocal != nil {
			err = errLocal
			return
		}

		testDBReadOnlyCalcRes = testDB
	})
	return err
}

// insertTestDBCalcRes inserts fake data into the in-memory DB for tests
func insertTestDBCalcRes(db *gorm.DB) error {
	if err := db.Create(&vm1).Error; err != nil {
		return err
	}

	if err := db.Create(&availableResTable).Error; err != nil {
		return err
	}

	if err := db.Create(&s1).Error; err != nil {
		return err
	}
	if err := db.Create(&serviceResReqs1).Error; err != nil {
		return err
	}
	if err := db.Create(&s2).Error; err != nil {
		return err
	}
	if err := db.Create(&serviceResReqs2).Error; err != nil {
		return err
	}

	return nil
}
