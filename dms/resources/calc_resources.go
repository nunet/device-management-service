package resources

import (
	"fmt"

	"github.com/shirou/gopsutil/cpu"
	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/models"
	"gorm.io/gorm"
)

// CalcFreeResAndUpdateDB calculates the FreeResources based on the AvailableResources
// and all processes started by DMS, and updates the FreeResources DB's table accordingly
func CalcFreeResAndUpdateDB() error {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return err
	}

	freeRes, err := calcFreeResources(db.DB, cpuInfo)
	if err != nil {
		return err
	}

	err = updateDBFreeResources(freeRes)
	if err != nil {
		return err
	}

	return nil
}

// calcFreeResources returns the subtraction between onboarded resources (AvailableResources)
// by the user and the sum of resources usage for every services, virtual machines, plugins
// and any other process started by DMS on the user's machine.
func calcFreeResources(gormDB *gorm.DB, cpuInfo []cpu.InfoStat) (models.FreeResources, error) {
	vms, err := queryRunningVMs(gormDB)
	if err != nil {
		return models.FreeResources{},
			fmt.Errorf("Error querying running VMs: %w", err)
	}

	conts, err := queryRunningConts(gormDB)
	if err != nil {
		return models.FreeResources{},
			fmt.Errorf("Error querying running containers: %w", err)
	}

	servicesReqs, err := getServiceResourcesRequirements(gormDB)
	if err != nil {
		return models.FreeResources{},
			fmt.Errorf("Error querying ServiceResourceRequirements table: %w", err)
	}

	resUsageVMs := calcUsedResourcesVMs(vms, cpuInfo)
	resUsageDcContainers := calcUsedResourcesConts(conts, servicesReqs)

	totalResourcesUsage := addResourcesUsage(resUsageDcContainers, resUsageVMs)

	freeResources, err := subtractFromAvailableRes(gormDB, totalResourcesUsage)
	if err != nil {
		return freeResources,
			fmt.Errorf("Couldn't subtract from AvailableResources, Error: %w", err)
	}

	return freeResources, nil
}

// calcUsedResourcesVMs returns the sum of resource usage between all running Firecracker
// virtual machines started by DMS.
func calcUsedResourcesVMs(vms []models.VirtualMachine, cpuInfo []cpu.InfoStat) models.FreeResources {
	var resourcesUsage models.FreeResources
	if len(vms) == 0 {
		return resourcesUsage
	}
	var totVCPU, totalMemSizeMib int
	for i := 0; i < len(vms); i++ {
		totVCPU += vms[i].VCPUCount
		totalMemSizeMib += vms[i].MemSizeMib
	}
	resourcesUsage.Ram = totalMemSizeMib
	resourcesUsage.TotCpuHz = totVCPU * int(cpuInfo[0].Mhz) // CPU in MHz
	return resourcesUsage
}

// calcUsedResourcesConts returns the sum of resource usage between all running Docker
// containers started by DMS.
func calcUsedResourcesConts(
	services []models.Services, requirements map[string]models.ServiceResourceRequirements,
) models.FreeResources {

	var resourcesUsage models.FreeResources
	if len(services) == 0 {
		return resourcesUsage
	}

	for _, service := range services {
		idx := fmt.Sprint(service.ResourceRequirements)
		resourcesReq := requirements[idx]

		resourcesUsage.TotCpuHz += resourcesReq.CPU
		resourcesUsage.Ram += resourcesReq.RAM
	}

	resourcesUsage.TotCpuHz += 1
	resourcesUsage.Ram += 1

	return resourcesUsage
}

// queryRunningVMs returns a list of running Firecracker's virtual machines
func queryRunningVMs(gormDB *gorm.DB) ([]models.VirtualMachine, error) {
	var vm []models.VirtualMachine
	result := gormDB.Where("state = ?", "running").Find(&vm)
	if result.Error != nil {
		return nil, fmt.Errorf("unable to query running vms - %v", result.Error)
	}
	return vm, nil

}

// queryRunningConts returns a list of running Docker Containers
func queryRunningConts(gormDB *gorm.DB) ([]models.Services, error) {
	var services []models.Services
	result := gormDB.Where("job_status = ?", "running").Find(&services)
	if result.Error != nil {
		return nil, fmt.Errorf("unable to query running containers - %v", result.Error)
	}

	return services, nil
}
