// +build linux
// +build !arm64

package cpu

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
)

var cpuMap = map[string]string{
	"vendor_id":  "vendor_id",
	"model name": "model_name",
	"cpu cores":  "cpu_cores",
	"siblings":   "cpu_logical_processors",
	"cpu MHz\t":  "mhz",
	"cache size": "cache_size",
	"cpu family": "family",
	"model\t":    "model",
	"stepping":   "stepping",
}

// Values that need to be multiplied by the number of physical processors
var perPhysicalProcValues = []string{
	"cpu_cores",
	"cpu_logical_processors",
}

func getCpuInfo() (cpuInfo map[string]string, err error) {
	lines, err := readProcFile()
	if err != nil {
		return
	}

	cpuInfo = make(map[string]string)
	// Implementation of a set that holds the physical IDs
	physicalProcIDs := make(map[string]struct{})

	for _, line := range lines {
		pair := regexp.MustCompile("\t: ").Split(line, 2)

		if pair[0] == "physical id" {
			physicalProcIDs[pair[1]] = struct{}{}
		}

		key, ok := cpuMap[pair[0]]
		if ok {
			cpuInfo[key] = pair[1]
		}
	}

	// Multiply the values that are "per physical processor" by the number of physical procs
	for _, field := range perPhysicalProcValues {
		if value, ok := cpuInfo[field]; ok {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			cpuInfo[field] = strconv.Itoa(intValue * len(physicalProcIDs))
		}
	}

	return
}

func readProcFile() (lines []string, err error) {
	file, err := os.Open("/proc/cpuinfo")

	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanner.Err() != nil {
		err = scanner.Err()
		return
	}

	return
}
