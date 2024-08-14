package resources

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/shirou/gopsutil/cpu"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ------------ Mocking data ------------ //

var (
	testOnceCalcRes       sync.Once
	testDBReadOnlyCalcRes *gorm.DB
)

var vm1 = types.VirtualMachine{
	VCPUCount:  10,
	MemSizeMib: 1000,
	State:      "running",
}

var availableResTable = types.AvailableResources{
	TotCpuHz: 15000,
	Ram:      11000,
	CpuHz:    100,
	Disk:     2000,
}

var s1 = types.Services{
	ResourceRequirements: 1,
	JobStatus:            "running",
}

var serviceResReqs1 = types.ServiceResourceRequirements{
	BaseDBModel: types.BaseDBModel{
		ID: "1",
	},
	CPU: 3000,
	RAM: 1000,
	HDD: 500,
}

var s2 = types.Services{
	ResourceRequirements: 2,
	JobStatus:            "running",
}

var serviceResReqs2 = types.ServiceResourceRequirements{
	BaseDBModel: types.BaseDBModel{
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

	expectedFreeRes := types.FreeResources{
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
// with []types.Services and types.ServiceResourceRequirements being mocked structs
func TestCalcUsedResourcesConts(t *testing.T) {
	services := []types.Services{s1, s2}
	requirements := map[string]types.ServiceResourceRequirements{
		"1": serviceResReqs1,
		"2": serviceResReqs2,
	}

	result := calcUsedResourcesConts(services, requirements)
	expected := types.FreeResources{
		TotCpuHz: serviceResReqs1.CPU + serviceResReqs2.CPU,
		Ram:      serviceResReqs1.RAM + serviceResReqs2.RAM,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Expected %v, got %v", expected, result)
	}
}

// TestCalcUsedResourcesVMs is an unit test which tests the calcUsedResourcesVMs()
// with []types.VirtualMachine being a mocked struct
func TestCalcUsedResourcesVMs(t *testing.T) {
	vms := []types.VirtualMachine{vm1}

	cpuInfo := mockCpuInfo()
	result := calcUsedResourcesVMs(vms, cpuInfo)

	expected := types.FreeResources{
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
			&types.AvailableResources{},
			&types.ServiceResourceRequirements{},
			&types.VirtualMachine{},
			&types.Services{},
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
