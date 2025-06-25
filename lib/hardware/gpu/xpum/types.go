// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package xpum

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
