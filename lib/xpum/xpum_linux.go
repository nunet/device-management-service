// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package xpum

/*
#cgo CXXFLAGS: -Iinclude -std=c++11
#cgo CFLAGS: -Iinclude
#cgo LDFLAGS: -ldl

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include "xpum_api.h"
#include "xpum_structs.h"

// ========================
// XPUM Function Pointer Definitions
// ========================

// Macro to define a function pointer type
#define DEFINE_XPUM_FUNC_TYPE(ret, name, args) typedef ret (*name##_fp) args

// Define function pointer types for Intel XPUM functions
DEFINE_XPUM_FUNC_TYPE(xpum_result_t, xpumInit, (void));
DEFINE_XPUM_FUNC_TYPE(xpum_result_t, xpumShutdown, (void));
DEFINE_XPUM_FUNC_TYPE(xpum_result_t, xpumGetDeviceList, (xpum_device_basic_info deviceList[], int *count));
DEFINE_XPUM_FUNC_TYPE(xpum_result_t, xpumGetStats, (xpum_device_id_t deviceId, xpum_device_stats_t dataList[], uint32_t *count, uint64_t *begin, uint64_t *end, uint64_t sessionId));
DEFINE_XPUM_FUNC_TYPE(xpum_result_t, xpumGetDeviceProperties, (xpum_device_id_t deviceId, xpum_device_properties_t *pXpumProperties));

// ========================
// XPUM Function Pointers Struct
// ========================

typedef struct {
    xpumInit_fp xpumInit;
    xpumShutdown_fp xpumShutdown;
    xpumGetDeviceList_fp xpumGetDeviceList;
    xpumGetStats_fp xpumGetStats;
    xpumGetDeviceProperties_fp xpumGetDeviceProperties;
} xpum_functions_t;

// ========================
// Global Variables
// ========================

static void *lib_handle = NULL;
static xpum_functions_t xpum_funcs;

// ========================
// Helper Macros
// ========================

// Macro to load a symbol and assign it to a struct member
#define LOAD_XPUM_SYMBOL(func_name) \
    xpum_funcs.func_name = (func_name##_fp)dlsym(lib_handle, #func_name); \
    if (!xpum_funcs.func_name) { \
        fprintf(stderr, "Error loading symbol %s: %s\n", #func_name, dlerror()); \
        dlclose(lib_handle); \
        lib_handle = NULL; \
        return 0; \
    }

// Macro to define a wrapper function
#define DEFINE_XPUM_WRAPPER(ret_type, wrapper_name, xpum_func, args, ...) \
    ret_type wrapper_name args { \
        if (xpum_funcs.xpum_func) { \
            return xpum_funcs.xpum_func(__VA_ARGS__); \
        } \
        return XPUM_GENERIC_ERROR; \
    }

// ========================
// Library Management Functions
// ========================

// Load the XPUM library and resolve all required symbols
int load_xpum_library() {
    if (lib_handle) {
        fprintf(stderr, "libxpum.so is already loaded.\n");
        return 1;
    }


    lib_handle = dlopen("libxpum.so", RTLD_LAZY);
    if (!lib_handle) {
        return 0;
    }

    // Clear any existing errors
    dlerror();

    // Load all required symbols
    LOAD_XPUM_SYMBOL(xpumInit)
    LOAD_XPUM_SYMBOL(xpumShutdown)
    LOAD_XPUM_SYMBOL(xpumGetDeviceList)
    LOAD_XPUM_SYMBOL(xpumGetStats)
    LOAD_XPUM_SYMBOL(xpumGetDeviceProperties)

    return 1;
}

// Unload the XPUM library and reset function pointers
void unload_xpum_library() {
    if (lib_handle) {
        dlclose(lib_handle);
        lib_handle = NULL;

        // Reset all function pointers to NULL
        xpum_funcs.xpumInit = NULL;
        xpum_funcs.xpumShutdown = NULL;
        xpum_funcs.xpumGetDeviceList = NULL;
        xpum_funcs.xpumGetStats = NULL;
        xpum_funcs.xpumGetDeviceProperties = NULL;
    }
}

// ========================
// Wrapper Functions
// ========================

DEFINE_XPUM_WRAPPER(xpum_result_t, call_xpumInit, xpumInit, (void))
DEFINE_XPUM_WRAPPER(xpum_result_t, call_xpumShutdown, xpumShutdown, (void))
DEFINE_XPUM_WRAPPER(xpum_result_t, call_xpumGetDeviceList, xpumGetDeviceList, (xpum_device_basic_info deviceList[], int *count), deviceList, count)
DEFINE_XPUM_WRAPPER(xpum_result_t, call_xpumGetStats, xpumGetStats, (xpum_device_id_t deviceId, xpum_device_stats_t dataList[], uint32_t *count, uint64_t *begin, uint64_t *end, uint64_t sessionId), deviceId, dataList, count, begin, end, sessionId)
DEFINE_XPUM_WRAPPER(xpum_result_t, call_xpumGetDeviceProperties, xpumGetDeviceProperties, (xpum_device_id_t deviceId, xpum_device_properties_t *pXpumProperties), deviceId, pXpumProperties)
*/
import "C"

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/avast/retry-go"
)

const (
	StatsMemoryUsed                      = C.XPUM_STATS_MEMORY_USED
	DevicePropertyMemoryPhysicalSizeByte = C.XPUM_DEVICE_PROPERTY_MEMORY_PHYSICAL_SIZE_BYTE
)

// DeviceBasicInfo is the Go representation of the C struct xpum_device_basic_info.
type DeviceBasicInfo struct {
	DeviceID      int32
	FunctionType  int32
	UUID          string
	DeviceName    string
	PCIDeviceID   string
	PCIBDFAddress string
	VendorName    string
	DRMDevice     string
}

// DeviceStatsData is the Go representation of the C struct xpum_device_stats_data.
type DeviceStatsData struct {
	MetricsType int32
	IsCounter   bool
	Value       uint64
	Accumulated uint64
	Min         uint64
	Avg         uint64
	Max         uint64
	Scale       uint32
}

// DeviceStats is the Go representation of the C struct xpum_device_stats.
type DeviceStats struct {
	DeviceID   int32
	IsTileData bool
	TileID     int32
	Count      int32
	DataList   []DeviceStatsData
}

// DeviceProperty represents a single device property
type DeviceProperty struct {
	Name  int32 // xpum_device_property_name_t
	Value string
}

// InitIntel initializes the Intel XPUM library.
func InitIntel() (bool, error) {
	// TODO: Set the log level based on the dms log level
	os.Setenv("SPDLOG_LEVEL", "off")
	if C.load_xpum_library() == 0 {
		return false, fmt.Errorf("load Intel XPUM library")
	}
	return C.call_xpumInit() == C.XPUM_OK, nil
}

// ShutdownIntel shuts down the Intel XPUM library.
func ShutdownIntel() bool {
	return C.call_xpumShutdown() == C.XPUM_OK
}

// GetDeviceList retrieves the list of devices and converts them to Go structs.
func GetDeviceList() ([]DeviceBasicInfo, error) {
	const maxDevices = 32
	var count C.int = maxDevices
	var deviceList [maxDevices]C.xpum_device_basic_info

	// Call the C wrapper function
	ret := C.call_xpumGetDeviceList((*C.xpum_device_basic_info)(unsafe.Pointer(&deviceList[0])), &count)
	if ret != C.XPUM_OK {
		return nil, fmt.Errorf("get device list: %v", ret)
	}

	// Convert C array to Go slice of DeviceBasicInfo
	goDevices := make([]DeviceBasicInfo, int(count))
	for i := 0; i < int(count); i++ {
		goDevices[i] = DeviceBasicInfo{
			DeviceID:      int32(deviceList[i].deviceId),
			FunctionType:  int32(deviceList[i].functionType),
			UUID:          C.GoString(&deviceList[i].uuid[0]),
			DeviceName:    C.GoString(&deviceList[i].deviceName[0]),
			PCIDeviceID:   C.GoString(&deviceList[i].PCIDeviceId[0]),
			PCIBDFAddress: C.GoString(&deviceList[i].PCIBDFAddress[0]),
			VendorName:    C.GoString(&deviceList[i].VendorName[0]),
			DRMDevice:     C.GoString(&deviceList[i].drmDevice[0]),
		}
	}

	return goDevices, nil
}

// GetDeviceStats retrieves device statistics for the specified device ID.
func GetDeviceStats(deviceID int32, sessionID uint64) ([]DeviceStats, error) {
	const maxStats = 100
	var count C.uint32_t = maxStats
	var begin, end C.uint64_t
	var statsList [maxStats]C.xpum_device_stats_t

	// We're retrying here because the XPUM library may not have initialized properly
	// Only the calls to xpumGetStats returns no data, so we retry this call
	// TODO: Add a check to see if the XPUM library is initialized by calling a c function ( if it exists )
	var goStats []DeviceStats
	if err := retry.Do(
		func() error {
			// Call the C wrapper function
			ret := C.call_xpumGetStats(C.xpum_device_id_t(deviceID), (*C.xpum_device_stats_t)(unsafe.Pointer(&statsList[0])), &count, &begin, &end, C.uint64_t(sessionID))
			if ret != C.XPUM_OK {
				return fmt.Errorf("get device stats: %v", ret)
			}

			// Convert C array to Go slice of DeviceStats
			goStats = make([]DeviceStats, int(count))
			hasData := false
			for i := 0; i < int(count); i++ {
				dataCount := int(statsList[i].count)
				if dataCount != 0 {
					hasData = true
				}
				goDataList := make([]DeviceStatsData, dataCount)
				for j := 0; j < dataCount; j++ {
					goDataList[j] = DeviceStatsData{
						MetricsType: int32(statsList[i].dataList[j].metricsType),
						IsCounter:   bool(statsList[i].dataList[j].isCounter),
						Value:       uint64(statsList[i].dataList[j].value),
						Accumulated: uint64(statsList[i].dataList[j].accumulated),
						Min:         uint64(statsList[i].dataList[j].min),
						Avg:         uint64(statsList[i].dataList[j].avg),
						Max:         uint64(statsList[i].dataList[j].max),
						Scale:       uint32(statsList[i].dataList[j].scale),
					}
				}

				goStats[i] = DeviceStats{
					DeviceID:   int32(statsList[i].deviceId),
					IsTileData: bool(statsList[i].isTileData),
					TileID:     int32(statsList[i].tileId),
					Count:      int32(statsList[i].count),
					DataList:   goDataList,
				}
			}

			if !hasData {
				return fmt.Errorf("no data for devices. It could be an xpum init issue. Retrying")
			}

			return nil
		},
		retry.Delay(1*time.Second),
		retry.Attempts(3),
	); err != nil {
		return nil, fmt.Errorf("get device stats: %w", err)
	}

	return goStats, nil
}

func GetDeviceProperties(deviceID int32) ([]DeviceProperty, error) {
	const xpumMaxNumProperties = 100

	// Prepare the properties struct
	cProperties := C.xpum_device_properties_t{
		deviceId:    C.xpum_device_id_t(deviceID),
		propertyLen: C.int(xpumMaxNumProperties),
	}

	// Call the C wrapper function
	ret := C.call_xpumGetDeviceProperties(C.xpum_device_id_t(deviceID), &cProperties)
	if ret != C.XPUM_OK {
		return nil, fmt.Errorf("get device properties: %v", ret)
	}

	numProperties := int(cProperties.propertyLen)
	goProperties := make([]DeviceProperty, numProperties)

	for i := 0; i < numProperties; i++ {
		cProp := cProperties.properties[i]
		goProperties[i] = DeviceProperty{
			Name:  int32(cProp.name),
			Value: C.GoString(&cProp.value[0]),
		}
	}
	return goProperties, nil
}
