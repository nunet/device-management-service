package cmd

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

const (
	sensorNVML nvml.TemperatureSensors = iota
)

type nvidiaGPU struct {
	index int
}

// helper function
func (n *nvidiaGPU) getDevice() (nvml.Device, error) {
	device, ret := nvml.DeviceGetHandleByIndex(n.index)
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device (index %d) handle: %s", n.index, nvml.ErrorString(ret))
	}
	return device, nil
}

func (n *nvidiaGPU) name() string {
	device, err := n.getDevice()
	if err != nil {
		return ""
	}

	name, ret := device.GetName()
	if ret != nvml.SUCCESS {
		return ""
	}

	return name
}

func (n *nvidiaGPU) utilizationRate() uint32 {
	device, err := n.getDevice()
	if err != nil {
		return 0
	}

	utilization, ret := device.GetUtilizationRates()
	if ret != nvml.SUCCESS {
		return 0
	}

	return utilization.Gpu
}

func (n *nvidiaGPU) memory() memoryInfo {
	device, err := n.getDevice()
	if err != nil {
		return memoryInfo{}
	}

	memoryNVML, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return memoryInfo{}
	}

	memory := memoryInfo{
		used:  memoryNVML.Used,
		free:  memoryNVML.Free,
		total: memoryNVML.Total,
	}

	return memory
}

func (n *nvidiaGPU) temperature() float64 {
	device, err := n.getDevice()
	if err != nil {
		return 0
	}

	temp, ret := device.GetTemperature(sensorNVML)
	if ret != nvml.SUCCESS {
		return 0
	}

	return float64(temp)
}

func (n *nvidiaGPU) powerUsage() uint32 {
	device, err := n.getDevice()
	if err != nil {
		return 0
	}

	power, ret := device.GetPowerUsage()
	if ret != nvml.SUCCESS {
		return 0
	}

	return power
}
