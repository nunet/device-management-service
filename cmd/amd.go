package cmd

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

type amdGPU struct {
	index int
}

func (a *amdGPU) name() string {
	pattern := fmt.Sprintf(`GPU\[%d\]\s+: Card series:\s+(.+)`, a.index)
	re := regexp.MustCompile(pattern)

	rocmOutput, err := runShellCmd("rocm-smi --showproductname")
	if err != nil {
		return ""
	}

	match := re.FindStringSubmatch(rocmOutput)
	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func (a *amdGPU) utilizationRate() uint32 {
	pattern := fmt.Sprintf(`GPU\[%d\]\s+: GPU use \(%%\): (\d+)`, a.index)
	re := regexp.MustCompile(pattern)

	rocmOutput, err := runShellCmd("rocm-smi --showuse")
	if err != nil {
		return 0
	}

	match := re.FindStringSubmatch(rocmOutput)
	if len(match) > 1 {
		utilization, err := strconv.ParseInt(match[1], 10, 32)
		if err != nil {
			return 0
		}

		return uint32(utilization)
	}

	return 0
}

func (a *amdGPU) memory() memoryInfo {
	patternTotal := fmt.Sprintf(`GPU\[%d\]\s+: vram Total Memory \(B\): (\d+)`, a.index)
	reTotal := regexp.MustCompile(patternTotal)

	patternUsed := fmt.Sprintf(`GPU\[%d\]\s+: vram Total Used Memory \(B\): (\d+)`, a.index)
	reUsed := regexp.MustCompile(patternUsed)

	rocmOutput, err := runShellCmd("rocm-smi --showmeminfo vram")
	if err != nil {
		return memoryInfo{}
	}

	matchTotal := reTotal.FindStringSubmatch(rocmOutput)
	matchUsed := reUsed.FindStringSubmatch(rocmOutput)

	if len(matchTotal) > 1 && len(matchUsed) > 1 {
		total, err := strconv.ParseInt(matchTotal[1], 10, 64)
		if err != nil {
			total = 0
		}

		used, err := strconv.ParseInt(matchUsed[1], 10, 64)
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

func (a *amdGPU) temperature() float64 {
	pattern := fmt.Sprintf(`GPU\[%d\]\s+: Temperature \(Sensor edge\) \(C\): ([\d\.]+)`, a.index)
	re := regexp.MustCompile(pattern)

	rocmOutput, err := runShellCmd("rocm-smi --showtemp")
	if err != nil {
		return 0
	}

	match := re.FindStringSubmatch(rocmOutput)
	if len(match) > 1 {
		temperature, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0
		}

		return temperature
	}

	return 0
}

func (a *amdGPU) powerUsage() uint32 {
	pattern := fmt.Sprintf(`GPU\[%d\]\s+: Average Graphics Package Power \(W\): ([\d\.]+)`, a.index)
	re := regexp.MustCompile(pattern)

	rocmOutput, err := runShellCmd("rocm-smi --showpower")
	if err != nil {
		return 0
	}

	match := re.FindStringSubmatch(rocmOutput)
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
