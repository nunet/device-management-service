// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package amdsmi

/*
#cgo CXXFLAGS: -Iinclude -std=c++11
#cgo CFLAGS: -Iinclude
#cgo LDFLAGS: -ldl

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include "amdsmi.h"

// Define constants for Go to use
const int GO_AMDSMI_PROCESSOR_TYPE_UNKNOWN = AMDSMI_PROCESSOR_TYPE_UNKNOWN;
const int GO_AMDSMI_PROCESSOR_TYPE_AMD_GPU = AMDSMI_PROCESSOR_TYPE_AMD_GPU;
const int GO_AMDSMI_PROCESSOR_TYPE_AMD_CPU = AMDSMI_PROCESSOR_TYPE_AMD_CPU;
const int GO_AMDSMI_PROCESSOR_TYPE_NON_AMD_GPU = AMDSMI_PROCESSOR_TYPE_NON_AMD_GPU;
const int GO_AMDSMI_PROCESSOR_TYPE_NON_AMD_CPU = AMDSMI_PROCESSOR_TYPE_NON_AMD_CPU;
const int GO_AMDSMI_PROCESSOR_TYPE_AMD_CPU_CORE = AMDSMI_PROCESSOR_TYPE_AMD_CPU_CORE;
const int GO_AMDSMI_PROCESSOR_TYPE_AMD_APU = AMDSMI_PROCESSOR_TYPE_AMD_APU;
const int GO_AMDSMI_GPU_UUID_SIZE = AMDSMI_GPU_UUID_SIZE;

// ========================
// AMD SMI Function Pointer Definitions
// ========================

// Macro to define a function pointer type
#define DEFINE_AMDSMI_FUNC_TYPE(ret, name, args) typedef ret (*name##_fp) args

// Define function pointer types for AMD SMI functions
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_init, (uint64_t flags));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_shut_down, (void));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_socket_handles, (uint32_t *socket_count, amdsmi_socket_handle* socket_handles));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_socket_info, (amdsmi_socket_handle socket_handle, size_t len, char *name));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_processor_handles, (amdsmi_socket_handle socket_handle, uint32_t* processor_count, amdsmi_processor_handle* processor_handles));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_processor_type, (amdsmi_processor_handle processor_handle, processor_type_t* processor_type));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_gpu_board_info, (amdsmi_processor_handle processor_handle, amdsmi_board_info_t *board_info));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_gpu_id, (amdsmi_processor_handle processor_handle, uint16_t *id));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_gpu_device_uuid, (amdsmi_processor_handle processor_handle, unsigned int *uuid_length, char *uuid));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_gpu_vram_usage, (amdsmi_processor_handle processor_handle, amdsmi_vram_usage_t *vram_info));
DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_get_gpu_bdf_id, (amdsmi_processor_handle processor_handle, uint64_t *bdf_id));

// ========================
// AMD SMI Function Pointers Struct
// ========================

typedef struct {
    amdsmi_init_fp amdsmi_init;
    amdsmi_shut_down_fp amdsmi_shut_down;
    amdsmi_get_socket_handles_fp amdsmi_get_socket_handles;
    amdsmi_get_socket_info_fp amdsmi_get_socket_info;
    amdsmi_get_processor_handles_fp amdsmi_get_processor_handles;
    amdsmi_get_processor_type_fp amdsmi_get_processor_type;
    amdsmi_get_gpu_board_info_fp amdsmi_get_gpu_board_info;
    amdsmi_get_gpu_id_fp amdsmi_get_gpu_id;
    amdsmi_get_gpu_device_uuid_fp amdsmi_get_gpu_device_uuid;
    amdsmi_get_gpu_vram_usage_fp amdsmi_get_gpu_vram_usage;
    amdsmi_get_gpu_bdf_id_fp amdsmi_get_gpu_bdf_id;
} amdsmi_functions_t;

// ========================
// Global Variables
// ========================

static void *lib_handle = NULL;
static amdsmi_functions_t amdsmi_funcs;

// ========================
// Helper Macros
// ========================

// Macro to load a symbol and assign it to a struct member
#define LOAD_AMDSMI_SYMBOL(func_name) \
    amdsmi_funcs.func_name = (func_name##_fp)dlsym(lib_handle, #func_name); \
    if (!amdsmi_funcs.func_name) { \
        fprintf(stderr, "Error loading symbol %s: %s\n", #func_name, dlerror()); \
        dlclose(lib_handle); \
        lib_handle = NULL; \
        return 0; \
    }

// Macro to define a wrapper function
#define DEFINE_AMDSMI_WRAPPER(ret_type, wrapper_name, amdsmi_func, args, ...) \
    ret_type wrapper_name args { \
        if (amdsmi_funcs.amdsmi_func) { \
            return amdsmi_funcs.amdsmi_func(__VA_ARGS__); \
        } \
        return AMDSMI_STATUS_INVAL; \
    }

// ========================
// Library Management Functions
// ========================

// Load the AMD SMI library and resolve all required symbols
int load_amdsmi_library() {
    if (lib_handle) {
        return 1;
    }

    lib_handle = dlopen("libamd_smi.so", RTLD_LAZY);
    if (!lib_handle) {
        return 0;
    }

    // Clear any existing errors
    dlerror();

    // Load all required symbols
    LOAD_AMDSMI_SYMBOL(amdsmi_init)
    LOAD_AMDSMI_SYMBOL(amdsmi_shut_down)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_socket_handles)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_socket_info)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_processor_handles)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_processor_type)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_gpu_board_info)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_gpu_id)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_gpu_device_uuid)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_gpu_vram_usage)
    LOAD_AMDSMI_SYMBOL(amdsmi_get_gpu_bdf_id)

    return 1;
}

// Unload the AMD SMI library and reset function pointers
void unload_amdsmi_library() {
    if (lib_handle) {
        dlclose(lib_handle);
        lib_handle = NULL;

        // Reset all function pointers to NULL
        amdsmi_funcs.amdsmi_init = NULL;
        amdsmi_funcs.amdsmi_shut_down = NULL;
        amdsmi_funcs.amdsmi_get_socket_handles = NULL;
        amdsmi_funcs.amdsmi_get_socket_info = NULL;
        amdsmi_funcs.amdsmi_get_processor_handles = NULL;
        amdsmi_funcs.amdsmi_get_processor_type = NULL;
        amdsmi_funcs.amdsmi_get_gpu_board_info = NULL;
        amdsmi_funcs.amdsmi_get_gpu_id = NULL;
        amdsmi_funcs.amdsmi_get_gpu_device_uuid = NULL;
        amdsmi_funcs.amdsmi_get_gpu_vram_usage = NULL;
        amdsmi_funcs.amdsmi_get_gpu_bdf_id = NULL;
    }
}

// ========================
// Wrapper Functions
// ========================

// Define wrappers for each AMD SMI function
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_init, amdsmi_init, (uint64_t flags), flags)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_shut_down, amdsmi_shut_down, (void))
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_socket_handles, amdsmi_get_socket_handles, (uint32_t *socket_count, amdsmi_socket_handle* socket_handles), socket_count, socket_handles)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_socket_info, amdsmi_get_socket_info, (amdsmi_socket_handle socket_handle, size_t len, char *name), socket_handle, len, name)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_processor_handles, amdsmi_get_processor_handles, (amdsmi_socket_handle socket_handle, uint32_t* processor_count, amdsmi_processor_handle* processor_handles), socket_handle, processor_count, processor_handles)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_processor_type, amdsmi_get_processor_type, (amdsmi_processor_handle processor_handle, processor_type_t* processor_type), processor_handle, processor_type)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_gpu_board_info, amdsmi_get_gpu_board_info, (amdsmi_processor_handle processor_handle, amdsmi_board_info_t *board_info), processor_handle, board_info)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_gpu_id, amdsmi_get_gpu_id, (amdsmi_processor_handle processor_handle, uint16_t *id), processor_handle, id)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_gpu_device_uuid, amdsmi_get_gpu_device_uuid, (amdsmi_processor_handle processor_handle, unsigned int *uuid_length, char *uuid), processor_handle, uuid_length, uuid)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_gpu_vram_usage, amdsmi_get_gpu_vram_usage, (amdsmi_processor_handle processor_handle, amdsmi_vram_usage_t *vram_info), processor_handle, vram_info)
DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_get_gpu_bdf_id, amdsmi_get_gpu_bdf_id, (amdsmi_processor_handle processor_handle, uint64_t *bdf_id), processor_handle, bdf_id)
*/
import "C"

/*
===============================================================================
Adding a New AMD SMI Function to the Cgo Block
===============================================================================

To add a new AMD SMI function, follow these steps:

1. **Define the Function Pointer Type:**
   - Use `DEFINE_AMDSMI_FUNC_TYPE` to create a typedef for the new function.
   - Example:
     `DEFINE_AMDSMI_FUNC_TYPE(amdsmi_status_t, amdsmi_new_function, (int arg1, float arg2));`

2. **Add to the Struct:**
   - Add the new function pointer to `amdsmi_functions_t`.
   - Example:
     `amdsmi_new_function_fp amdsmi_new_function;`

3. **Load the Symbol:**
   - In `load_amdsmi_library`, load the new symbol using `LOAD_AMDSMI_SYMBOL`.
   - Example:
     `LOAD_AMDSMI_SYMBOL(amdsmi_new_function)`

4. **Create a Wrapper Function:**
   - Define a wrapper using `DEFINE_AMDSMI_WRAPPER`.
   - Example:
     `DEFINE_AMDSMI_WRAPPER(amdsmi_status_t, call_amdsmi_new_function, amdsmi_new_function, (int arg1, float arg2), arg1, arg2)`

5. **Rebuild and Test:**
   - Rebuild and verify the new function works as expected.

===============================================================================
*/

import (
	"fmt"
	"unsafe"
)

// ProcessorType is a Go type to represent processor_type_t from C.
type ProcessorType uint32

var (
	ProcessorTypeUnknown    = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_UNKNOWN)
	ProcessorTypeAMDGPU     = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_AMD_GPU)
	ProcessorTypeAMDCPU     = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_AMD_CPU)
	ProcessorTypeNonAMDGPU  = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_NON_AMD_GPU)
	ProcessorTypeNonAMDCPU  = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_NON_AMD_CPU)
	ProcessorTypeAMDCPUCore = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_AMD_CPU_CORE)
	ProcessorTypeAMDAPU     = ProcessorType(C.GO_AMDSMI_PROCESSOR_TYPE_AMD_APU)

	GPUUUIDSize = uint(C.GO_AMDSMI_GPU_UUID_SIZE)
)

// Init initializes the AMD SMI library with GPUs.
func Init() (Status, error) {
	if C.load_amdsmi_library() == 0 {
		return Status{}, fmt.Errorf("failed to load AMD SMI library")
	}

	ret := C.call_amdsmi_init(0)
	return Status{
		Code: StatusCode(ret),
	}, nil
}

// Shutdown shuts down the AMD SMI library.
func Shutdown() Status {
	ret := C.call_amdsmi_shut_down()
	C.unload_amdsmi_library()
	return Status{
		Code: StatusCode(ret),
	}
}

// GetSocketHandles returns the socket handles of the GPUs.
func GetSocketHandles() ([]SocketHandle, Status) {
	var socketCount C.uint32_t

	ret := C.call_amdsmi_get_socket_handles(&socketCount, nil)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return nil, Status{Code: StatusCode(ret), Message: "get socket count"}
	}

	if socketCount == 0 {
		return nil, Status{Code: StatusSuccess, Message: "no socket found"}
	}

	sockets := make([]C.amdsmi_socket_handle, socketCount)
	ret = C.call_amdsmi_get_socket_handles(&socketCount, (*C.amdsmi_socket_handle)(unsafe.Pointer(&sockets[0])))
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return nil, Status{Code: StatusCode(ret), Message: "get socket handles"}
	}

	goSockets := make([]SocketHandle, socketCount)
	for i, socket := range sockets {
		goSockets[i] = SocketHandle{handle: unsafe.Pointer(socket)}
	}

	return goSockets, Status{Code: StatusSuccess}
}

// GetSocketName retrieves the socket name for a given socket handle.
func GetSocketName(socketHandle SocketHandle, maxLen int) (string, Status) {
	name := make([]C.char, maxLen)

	ret := C.call_amdsmi_get_socket_info(C.amdsmi_socket_handle(socketHandle.handle), C.size_t(maxLen), (*C.char)(unsafe.Pointer(&name[0])))
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return "", Status{Code: StatusCode(ret), Message: "get socket info"}
	}

	socketInfo := C.GoString(&name[0])
	return socketInfo, Status{Code: StatusSuccess}
}

// GetProcessorHandles retrieves all processor handles for a given socket.
func GetProcessorHandles(socket SocketHandle) ([]ProcessorHandle, Status) {
	var processorCount C.uint32_t
	ret := C.call_amdsmi_get_processor_handles(C.amdsmi_socket_handle(socket.handle), &processorCount, nil)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return nil, Status{Code: StatusCode(ret), Message: "get processor count"}
	}

	processors := make([]C.amdsmi_processor_handle, processorCount)
	ret = C.call_amdsmi_get_processor_handles(C.amdsmi_socket_handle(socket.handle), &processorCount, (*C.amdsmi_processor_handle)(unsafe.Pointer(&processors[0])))
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return nil, Status{Code: StatusCode(ret), Message: "get processor handles"}
	}

	goProcessors := make([]ProcessorHandle, processorCount)
	for i, processor := range processors {
		goProcessors[i] = ProcessorHandle{handle: unsafe.Pointer(processor)}
	}

	return goProcessors, Status{Code: StatusSuccess}
}

// GetProcessorType retrieves the type of a given processor.
func GetProcessorType(processor ProcessorHandle) (ProcessorType, Status) {
	var processorType C.processor_type_t
	ret := C.call_amdsmi_get_processor_type(C.amdsmi_processor_handle(processor.handle), &processorType)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return 0, Status{Code: StatusCode(ret), Message: "get processor type"}
	}

	return ProcessorType(processorType), Status{Code: StatusSuccess}
}

// GetGPUBoardInfo retrieves the board information for a given GPU processor handle.
func GetGPUBoardInfo(processor ProcessorHandle) (BoardInfo, Status) {
	var boardInfo C.amdsmi_board_info_t

	ret := C.call_amdsmi_get_gpu_board_info(C.amdsmi_processor_handle(processor.handle), &boardInfo)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return BoardInfo{}, Status{Code: StatusCode(ret), Message: "get GPU board info"}
	}

	goBoardInfo := BoardInfo{
		ModelNumber:      C.GoString(&boardInfo.model_number[0]),
		ProductSerial:    C.GoString(&boardInfo.product_serial[0]),
		FruID:            C.GoString(&boardInfo.fru_id[0]),
		ProductName:      C.GoString(&boardInfo.product_name[0]),
		ManufacturerName: C.GoString(&boardInfo.manufacturer_name[0]),
	}
	return goBoardInfo, Status{Code: StatusSuccess}
}

// GetGPUID retrieves the GPU ID for a given processor handle.
func GetGPUID(processor ProcessorHandle) (uint32, Status) {
	var gpuID C.uint16_t
	ret := C.call_amdsmi_get_gpu_id(C.amdsmi_processor_handle(processor.handle), &gpuID)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return 0, Status{Code: StatusCode(ret), Message: "get GPU ID"}
	}

	return uint32(gpuID), Status{Code: StatusSuccess}
}

// GetGPUBDFID retrieves the GPU BDF ID for a given processor handle.
func GetGPUBDFID(processor ProcessorHandle) (uint64, Status) {
	var bdfID C.uint64_t
	ret := C.call_amdsmi_get_gpu_bdf_id(C.amdsmi_processor_handle(processor.handle), &bdfID)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return 0, Status{Code: StatusCode(ret), Message: "get GPU BDF ID"}
	}

	return uint64(bdfID), Status{Code: StatusSuccess}
}

// GetGPUUUID retrieves the GPU UUID for a given processor handle.
func GetGPUUUID(processor ProcessorHandle) (string, Status) {
	var uuid [38]C.char
	var length C.uint = 38
	ret := C.call_amdsmi_get_gpu_device_uuid(C.amdsmi_processor_handle(processor.handle), &length, (*C.char)(unsafe.Pointer(&uuid)))
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return "", Status{Code: StatusCode(ret), Message: "get GPU UUID"}
	}

	return C.GoString(&uuid[0]), Status{Code: StatusSuccess}
}

// GetGPUVRAM retrieves the GPU VRAM stats for a given processor handle.
func GetGPUVRAM(processor ProcessorHandle) (VRAM, Status) {
	var vramUsage C.amdsmi_vram_usage_t
	ret := C.call_amdsmi_get_gpu_vram_usage(C.amdsmi_processor_handle(processor.handle), &vramUsage)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		return VRAM{}, Status{Code: StatusCode(ret), Message: "get GPU VRAM usage"}
	}

	goVRAM := VRAM{
		Total: uint32(vramUsage.vram_total),
		Used:  uint32(vramUsage.vram_used),
	}
	return goVRAM, Status{Code: StatusSuccess}
}
