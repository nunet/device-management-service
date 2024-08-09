package cmd

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type intelGPU struct {
	index int
}

// name returns the name of the Intel GPU.
func (i *intelGPU) name() string {
	pattern := fmt.Sprintf(`Device Name:\s+(.+)`)
	re := regexp.MustCompile(pattern)

	xpuOutput, err := runShellCmd(fmt.Sprintf("xpu-smi discovery -d %d", i.index))
	if err != nil {
		return ""
	}

	match := re.FindStringSubmatch(xpuOutput)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	return ""
}

// utilizationRate returns the utilization rate of the Intel GPU.
func (i *intelGPU) utilizationRate() uint32 {
	pattern := fmt.Sprintf(`GPU Utilization \(%%\)\s+\|\s+(\d+)`)
	re := regexp.MustCompile(pattern)

	xpuOutput, err := runShellCmd(fmt.Sprintf("xpu-smi stats -d %d", i.index))
	if err != nil {
		return 0
	}

	match := re.FindStringSubmatch(xpuOutput)
	if len(match) > 1 {
		utilization, err := strconv.ParseInt(match[1], 10, 32)
		if err != nil {
			return 0
		}

		return uint32(utilization)
	}

	return 0
}

// memory returns the memory information of the Intel GPU.
func (i *intelGPU) memory() memoryInfo {
	patternTotal := fmt.Sprintf(`Memory Physical Size:\s+([^\s]+)\s+MiB`)
	reTotal := regexp.MustCompile(patternTotal)

	patternUsed := fmt.Sprintf(`GPU Memory Used \(MiB\)\s+\|\s+(\d+)`)
	reUsed := regexp.MustCompile(patternUsed)

	xpuOutput, err := runShellCmd(fmt.Sprintf("xpu-smi discovery -d %d", i.index))
	if err != nil {
		return memoryInfo{}
	}

	matchTotal := reTotal.FindStringSubmatch(xpuOutput)
	matchUsed := reUsed.FindStringSubmatch(xpuOutput)

	if len(matchTotal) > 1 && len(matchUsed) > 1 {
		total, err := strconv.ParseFloat(matchTotal[1], 64)
		if err != nil {
			total = 0
		}

		used, err := strconv.ParseFloat(matchUsed[1], 64)
		if err != nil {
			used = 0
		}

		free := (total - used)

		return memoryInfo{
			used:  uint64(used),
			free:  uint64(free),
			total: uint64(total),
		}
	}

	return memoryInfo{}
}

// powerUsage returns the power usage of the Intel GPU.
func (i *intelGPU) powerUsage() uint32 {
	pattern := fmt.Sprintf(`GPU Power \(W\)\s+\|\s+([\d\.]+)`)
	re := regexp.MustCompile(pattern)

	xpuOutput, err := runShellCmd(fmt.Sprintf("xpu-smi stats -d %d", i.index))
	if err != nil {
		return 0
	}

	match := re.FindStringSubmatch(xpuOutput)
	if len(match) > 1 {
		powerFloat, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0
		}

		power := uint32(math.Round(powerFloat))

		return power
	}

	return 0
}
