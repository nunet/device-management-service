// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || amd)

package amdsmi

import "fmt"

// StatusCode is a Go type to represent amdsmi_status_t from C.
type StatusCode uint32

const (
	// General success and error codes

	StatusSuccess           StatusCode = 0  // AMDSMI_STATUS_SUCCESS
	StatusInval             StatusCode = 1  // AMDSMI_STATUS_INVAL
	StatusNotSupported      StatusCode = 2  // AMDSMI_STATUS_NOT_SUPPORTED
	StatusNotYetImplemented StatusCode = 3  // AMDSMI_STATUS_NOT_YET_IMPLEMENTED
	StatusFailLoadModule    StatusCode = 4  // AMDSMI_STATUS_FAIL_LOAD_MODULE
	StatusFailLoadSymbol    StatusCode = 5  // AMDSMI_STATUS_FAIL_LOAD_SYMBOL
	StatusDRMError          StatusCode = 6  // AMDSMI_STATUS_DRM_ERROR
	StatusAPIFailed         StatusCode = 7  // AMDSMI_STATUS_API_FAILED
	StatusTimeout           StatusCode = 8  // AMDSMI_STATUS_TIMEOUT
	StatusRetry             StatusCode = 9  // AMDSMI_STATUS_RETRY
	StatusNoPerm            StatusCode = 10 // AMDSMI_STATUS_NO_PERM
	StatusInterrupt         StatusCode = 11 // AMDSMI_STATUS_INTERRUPT
	StatusIO                StatusCode = 12 // AMDSMI_STATUS_IO
	StatusAddressFault      StatusCode = 13 // AMDSMI_STATUS_ADDRESS_FAULT
	StatusFileError         StatusCode = 14 // AMDSMI_STATUS_FILE_ERROR
	StatusOutOfResources    StatusCode = 15 // AMDSMI_STATUS_OUT_OF_RESOURCES
	StatusInternalException StatusCode = 16 // AMDSMI_STATUS_INTERNAL_EXCEPTION
	StatusInputOutOfBounds  StatusCode = 17 // AMDSMI_STATUS_INPUT_OUT_OF_BOUNDS
	StatusInitError         StatusCode = 18 // AMDSMI_STATUS_INIT_ERROR
	StatusRefcountOverflow  StatusCode = 19 // AMDSMI_STATUS_REFCOUNT_OVERFLOW

	// Device related errors (starting from 30)

	StatusBusy            StatusCode = 30 // AMDSMI_STATUS_BUSY
	StatusNotFound        StatusCode = 31 // AMDSMI_STATUS_NOT_FOUND
	StatusNotInit         StatusCode = 32 // AMDSMI_STATUS_NOT_INIT
	StatusNoSlot          StatusCode = 33 // AMDSMI_STATUS_NO_SLOT
	StatusDriverNotLoaded StatusCode = 34 // AMDSMI_STATUS_DRIVER_NOT_LOADED

	// Data and size errors (starting from 40)

	StatusNoData           StatusCode = 40 // AMDSMI_STATUS_NO_DATA
	StatusInsufficientSize StatusCode = 41 // AMDSMI_STATUS_INSUFFICIENT_SIZE
	StatusUnexpectedSize   StatusCode = 42 // AMDSMI_STATUS_UNEXPECTED_SIZE
	StatusUnexpectedData   StatusCode = 43 // AMDSMI_STATUS_UNEXPECTED_DATA

	// General errors with specific values

	StatusMapError     StatusCode = 0xFFFFFFFE // AMDSMI_STATUS_MAP_ERROR
	StatusUnknownError StatusCode = 0xFFFFFFFF // AMDSMI_STATUS_UNKNOWN_ERROR
)

// String returns the string representation of the StatusCode
func (code StatusCode) String() string {
	switch code {
	case StatusSuccess:
		return "Success"
	case StatusInval:
		return "Inval"
	case StatusNotSupported:
		return "NotSupported"
	case StatusNotYetImplemented:
		return "NotYetImplemented"
	case StatusFailLoadModule:
		return "FailLoadModule"
	case StatusFailLoadSymbol:
		return "FailLoadSymbol"
	case StatusDRMError:
		return "DRMError"
	case StatusAPIFailed:
		return "APIFailed"
	case StatusTimeout:
		return "Timeout"
	case StatusRetry:
		return "Retry"
	case StatusNoPerm:
		return "NoPerm"
	case StatusInterrupt:
		return "Interrupt"
	case StatusIO:
		return "IO"
	case StatusAddressFault:
		return "AddressFault"
	case StatusFileError:
		return "FileError"
	case StatusOutOfResources:
		return "OutOfResources"
	case StatusInternalException:
		return "InternalException"
	case StatusInputOutOfBounds:
		return "InputOutOfBounds"
	case StatusInitError:
		return "InitError"
	case StatusRefcountOverflow:
		return "RefcountOverflow"

	// Device related errors
	case StatusBusy:
		return "Busy"
	case StatusNotFound:
		return "NotFound"
	case StatusNotInit:
		return "NotInit"
	case StatusNoSlot:
		return "NoSlot"
	case StatusDriverNotLoaded:
		return "DriverNotLoaded"

	// Data and size errors
	case StatusNoData:
		return "NoData"
	case StatusInsufficientSize:
		return "InsufficientSize"
	case StatusUnexpectedSize:
		return "UnexpectedSize"
	case StatusUnexpectedData:
		return "UnexpectedData"

	// General errors
	case StatusMapError:
		return "MapError"
	case StatusUnknownError:
		return "UnknownError"

	default:
		return fmt.Sprintf("AMDSMIStatusCode(%d)", code)
	}
}

// Status a wrapper around StatusCode and a message
type Status struct {
	Code    StatusCode
	Message string
}

// Error returns the error message if the status is not success
func (s Status) Error() error {
	if s.Code == StatusSuccess {
		return nil
	}

	return fmt.Errorf("%s %s(%d)", s.Message, s.Code.String(), s.Code)
}
