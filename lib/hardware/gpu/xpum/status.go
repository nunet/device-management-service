// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package xpum

import "fmt"

// ResultCode is a Go type to represent xpum_result_t.
type ResultCode uint32

const (
	ResultOk                                              ResultCode = iota // XPUM_OK = 0
	ResultGenericError                                                      // XPUM_GENERIC_ERROR = 1
	ResultBufferTooSmall                                                    // XPUM_BUFFER_TOO_SMALL = 2
	ResultDeviceNotFound                                                    // XPUM_RESULT_DEVICE_NOT_FOUND = 3
	ResultTileNotFound                                                      // XPUM_RESULT_TILE_NOT_FOUND = 4
	ResultGroupNotFound                                                     // XPUM_RESULT_GROUP_NOT_FOUND = 5
	ResultPolicyTypeInvalid                                                 // XPUM_RESULT_POLICY_TYPE_INVALID = 6
	ResultPolicyActionTypeInvalid                                           // XPUM_RESULT_POLICY_ACTION_TYPE_INVALID = 7
	ResultPolicyConditionTypeInvalid                                        // XPUM_RESULT_POLICY_CONDITION_TYPE_INVALID = 8
	ResultPolicyTypeActionNotSupport                                        // XPUM_RESULT_POLICY_TYPE_ACTION_NOT_SUPPORT = 9
	ResultPolicyTypeConditionNotSupport                                     // XPUM_RESULT_POLICY_TYPE_CONDITION_NOT_SUPPORT = 10
	ResultPolicyInvalidThreshold                                            // XPUM_RESULT_POLICY_INVALID_THRESHOLD = 11
	ResultPolicyInvalidFrequency                                            // XPUM_RESULT_POLICY_INVALID_FREQUENCY = 12
	ResultPolicyNotExist                                                    // XPUM_RESULT_POLICY_NOT_EXIST = 13
	ResultDiagnosticTaskNotComplete                                         // XPUM_RESULT_DIAGNOSTIC_TASK_NOT_COMPLETE = 14
	ResultDiagnosticTaskNotFound                                            // XPUM_RESULT_DIAGNOSTIC_TASK_NOT_FOUND = 15
	GroupDeviceDuplicated                                                   // XPUM_GROUP_DEVICE_DUPLICATED = 16
	GroupChangeNotAllowed                                                   // XPUM_GROUP_CHANGE_NOT_ALLOWED = 17
	NotInitialized                                                          // XPUM_NOT_INITIALIZED = 18
	DumpRawDataTaskNotExist                                                 // XPUM_DUMP_RAW_DATA_TASK_NOT_EXIST = 19
	DumpRawDataIllegalDumpFilePath                                          // XPUM_DUMP_RAW_DATA_ILLEGAL_DUMP_FILE_PATH = 20
	ResultUnknownAgentConfigKey                                             // XPUM_RESULT_UNKNOWN_AGENT_CONFIG_KEY = 21
	UpdateFirmwareImageFileNotFound                                         // XPUM_UPDATE_FIRMWARE_IMAGE_FILE_NOT_FOUND = 22
	UpdateFirmwareUnsupportedAmc                                            // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_AMC = 23
	UpdateFirmwareUnsupportedAmcSingle                                      // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_AMC_SINGLE = 24
	UpdateFirmwareUnsupportedGfxAll                                         // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_GFX_ALL = 25
	UpdateFirmwareModelInconsistence                                        // XPUM_UPDATE_FIRMWARE_MODEL_INCONSISTENCE = 26
	UpdateFirmwareIgscNotFound                                              // XPUM_UPDATE_FIRMWARE_IGSC_NOT_FOUND = 27
	UpdateFirmwareTaskRunning                                               // XPUM_UPDATE_FIRMWARE_TASK_RUNNING = 28
	UpdateFirmwareInvalidFwImage                                            // XPUM_UPDATE_FIRMWARE_INVALID_FW_IMAGE = 29
	UpdateFirmwareFwImageNotCompatibleWithDevice                            // XPUM_UPDATE_FIRMWARE_FW_IMAGE_NOT_COMPATIBLE_WITH_DEVICE = 30
	ResultDumpMetricsTypeNotSupport                                         // XPUM_RESULT_DUMP_METRICS_TYPE_NOT_SUPPORT = 31
	MetricNotSupported                                                      // XPUM_METRIC_NOT_SUPPORTED = 32
	MetricNotEnabled                                                        // XPUM_METRIC_NOT_ENABLED = 33
	ResultHealthInvalidType                                                 // XPUM_RESULT_HEALTH_INVALID_TYPE = 34
	ResultHealthInvalidConfigType                                           // XPUM_RESULT_HEALTH_INVALID_CONIG_TYPE = 35
	ResultHealthInvalidThreshold                                            // XPUM_RESULT_HEALTH_INVALID_THRESHOLD = 36
	ResultDiagnosticInvalidLevel                                            // XPUM_RESULT_DIAGNOSTIC_INVALID_LEVEL = 37
	ResultDiagnosticInvalidTaskType                                         // XPUM_RESULT_DIAGNOSTIC_INVALID_TASK_TYPE = 38
	ResultAgentSetInvalidValue                                              // XPUM_RESULT_AGENT_SET_INVALID_VALUE = 39
	LevelZeroInitializationError                                            // XPUM_LEVEL_ZERO_INITIALIZATION_ERROR = 40
	UnsupportedSessionID                                                    // XPUM_UNSUPPORTED_SESSIONID = 41
	ResultMemoryEccLibNotSupport                                            // XPUM_RESULT_MEMORY_ECC_LIB_NOT_SUPPORT = 42
	UpdateFirmwareUnsupportedGfxData                                        // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_GFX_DATA = 43
	UpdateFirmwareUnsupportedPsc                                            // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_PSC = 44
	UpdateFirmwareUnsupportedPscIgsc                                        // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_PSC_IGSC = 45
	UpdateFirmwareUnsupportedGfxCodeData                                    // XPUM_UPDATE_FIRMWARE_UNSUPPORTED_GFX_CODE_DATA = 46
	IntervalInvalid                                                         // XPUM_INTERVAL_INVALID = 47
	ResultFileDup                                                           // XPUM_RESULT_FILE_DUP = 48
	ResultInvalidDir                                                        // XPUM_RESULT_INVALID_DIR = 49
	ResultFwMgmtNotInit                                                     // XPUM_RESULT_FW_MGMT_NOT_INIT = 50
	VgpuInvalidLmem                                                         // XPUM_VGPU_INVALID_LMEM = 51
	VgpuInvalidNumVfs                                                       // XPUM_VGPU_INVALID_NUMVFS = 52
	VgpuDirtyPf                                                             // XPUM_VGPU_DIRTY_PF = 53
	VgpuVfUnsupportedOperation                                              // XPUM_VGPU_VF_UNSUPPORTED_OPERATION = 54
	VgpuCreateVfFailed                                                      // XPUM_VGPU_CREATE_VF_FAILED = 55
	VgpuRemoveVfFailed                                                      // XPUM_VGPU_REMOVE_VF_FAILED = 56
	VgpuNoConfigFile                                                        // XPUM_VGPU_NO_CONFIG_FILE = 57
	VgpuSysfsError                                                          // XPUM_VGPU_SYSFS_ERROR = 58
	VgpuUnsupportedDeviceModel                                              // XPUM_VGPU_UNSUPPORTED_DEVICE_MODEL = 59
	ResultResetFail                                                         // XPUM_RESULT_RESET_FAIL = 60
	APIUnsupported                                                          // XPUM_API_UNSUPPORTED = 61
	PrecheckInvalidSinceTime                                                // XPUM_PRECHECK_INVALID_SINCETIME = 62
	PprNotFound                                                             // XPUM_PPR_NOT_FOUND = 63
	UpdateFirmwareGfxDataImageVersionLowerOrEqualToDevice                   // XPUM_UPDATE_FIRMWARE_GFX_DATA_IMAGE_VERSION_LOWER_OR_EQUAL_TO_DEVICE = 64
	ResultUnsupportedDevice                                                 // XPUM_RESULT_UNSUPPORTED_DEVICE = 65
	GroupLimitReached                                                       // XPUM_GROUP_LIMIT_REACHED = 66
)

// String returns the string representation of the ResultCode without the 'Result' prefix.
func (code ResultCode) String() string {
	switch code {
	case ResultOk:
		return "Ok"
	case ResultGenericError:
		return "GenericError"
	case ResultBufferTooSmall:
		return "BufferTooSmall"
	case ResultDeviceNotFound:
		return "DeviceNotFound"
	case ResultTileNotFound:
		return "TileNotFound"
	case ResultGroupNotFound:
		return "GroupNotFound"
	case ResultPolicyTypeInvalid:
		return "PolicyTypeInvalid"
	case ResultPolicyActionTypeInvalid:
		return "PolicyActionTypeInvalid"
	case ResultPolicyConditionTypeInvalid:
		return "PolicyConditionTypeInvalid"
	case ResultPolicyTypeActionNotSupport:
		return "PolicyTypeActionNotSupport"
	case ResultPolicyTypeConditionNotSupport:
		return "PolicyTypeConditionNotSupport"
	case ResultPolicyInvalidThreshold:
		return "PolicyInvalidThreshold"
	case ResultPolicyInvalidFrequency:
		return "PolicyInvalidFrequency"
	case ResultPolicyNotExist:
		return "PolicyNotExist"
	case ResultDiagnosticTaskNotComplete:
		return "DiagnosticTaskNotComplete"
	case ResultDiagnosticTaskNotFound:
		return "DiagnosticTaskNotFound"
	case GroupDeviceDuplicated:
		return "GroupDeviceDuplicated"
	case GroupChangeNotAllowed:
		return "GroupChangeNotAllowed"
	case NotInitialized:
		return "NotInitialized"
	case DumpRawDataTaskNotExist:
		return "DumpRawDataTaskNotExist"
	case DumpRawDataIllegalDumpFilePath:
		return "DumpRawDataIllegalDumpFilePath"
	case ResultUnknownAgentConfigKey:
		return "UnknownAgentConfigKey"
	case UpdateFirmwareImageFileNotFound:
		return "UpdateFirmwareImageFileNotFound"
	case UpdateFirmwareUnsupportedAmc:
		return "UpdateFirmwareUnsupportedAmc"
	case UpdateFirmwareUnsupportedAmcSingle:
		return "UpdateFirmwareUnsupportedAmcSingle"
	case UpdateFirmwareUnsupportedGfxAll:
		return "UpdateFirmwareUnsupportedGfxAll"
	case UpdateFirmwareModelInconsistence:
		return "UpdateFirmwareModelInconsistence"
	case UpdateFirmwareIgscNotFound:
		return "UpdateFirmwareIgscNotFound"
	case UpdateFirmwareTaskRunning:
		return "UpdateFirmwareTaskRunning"
	case UpdateFirmwareInvalidFwImage:
		return "UpdateFirmwareInvalidFwImage"
	case UpdateFirmwareFwImageNotCompatibleWithDevice:
		return "UpdateFirmwareFwImageNotCompatibleWithDevice"
	case ResultDumpMetricsTypeNotSupport:
		return "DumpMetricsTypeNotSupport"
	case MetricNotSupported:
		return "MetricNotSupported"
	case MetricNotEnabled:
		return "MetricNotEnabled"
	case ResultHealthInvalidType:
		return "HealthInvalidType"
	case ResultHealthInvalidConfigType:
		return "HealthInvalidConfigType"
	case ResultHealthInvalidThreshold:
		return "HealthInvalidThreshold"
	case ResultDiagnosticInvalidLevel:
		return "DiagnosticInvalidLevel"
	case ResultDiagnosticInvalidTaskType:
		return "DiagnosticInvalidTaskType"
	case ResultAgentSetInvalidValue:
		return "AgentSetInvalidValue"
	case LevelZeroInitializationError:
		return "LevelZeroInitializationError"
	case UnsupportedSessionID:
		return "UnsupportedSessionId"
	case ResultMemoryEccLibNotSupport:
		return "MemoryEccLibNotSupport"
	case UpdateFirmwareUnsupportedGfxData:
		return "UpdateFirmwareUnsupportedGfxData"
	case UpdateFirmwareUnsupportedPsc:
		return "UpdateFirmwareUnsupportedPsc"
	case UpdateFirmwareUnsupportedPscIgsc:
		return "UpdateFirmwareUnsupportedPscIgsc"
	case UpdateFirmwareUnsupportedGfxCodeData:
		return "UpdateFirmwareUnsupportedGfxCodeData"
	case IntervalInvalid:
		return "IntervalInvalid"
	case ResultFileDup:
		return "FileDup"
	case ResultInvalidDir:
		return "InvalidDir"
	case ResultFwMgmtNotInit:
		return "FwMgmtNotInit"
	case VgpuInvalidLmem:
		return "VgpuInvalidLmem"
	case VgpuInvalidNumVfs:
		return "VgpuInvalidNumVfs"
	case VgpuDirtyPf:
		return "VgpuDirtyPf"
	case VgpuVfUnsupportedOperation:
		return "VgpuVfUnsupportedOperation"
	case VgpuCreateVfFailed:
		return "VgpuCreateVfFailed"
	case VgpuRemoveVfFailed:
		return "VgpuRemoveVfFailed"
	case VgpuNoConfigFile:
		return "VgpuNoConfigFile"
	case VgpuSysfsError:
		return "VgpuSysfsError"
	case VgpuUnsupportedDeviceModel:
		return "VgpuUnsupportedDeviceModel"
	case ResultResetFail:
		return "ResetFail"
	case APIUnsupported:
		return "ApiUnsupported"
	case PrecheckInvalidSinceTime:
		return "PrecheckInvalidSinceTime"
	case PprNotFound:
		return "PprNotFound"
	case UpdateFirmwareGfxDataImageVersionLowerOrEqualToDevice:
		return "UpdateFirmwareGfxDataImageVersionLowerOrEqualToDevice"
	case ResultUnsupportedDevice:
		return "UnsupportedDevice"
	case GroupLimitReached:
		return "GroupLimitReached"
	default:
		return fmt.Sprintf("ResultCode(%d)", code)
	}
}

// Result is a wrapper around ResultCode and a message.
type Result struct {
	Code    ResultCode
	Message string
}

// Error returns the error message if the ResultCode is not ResultOk.
func (r Result) Error() error {
	if r.Code == ResultOk {
		return nil
	}

	return fmt.Errorf("%s %s(%d)", r.Message, r.Code.String(), r.Code)
}
