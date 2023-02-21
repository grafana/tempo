package cpu

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/gohai/utils"
)

// Cpu holds metadata about the host CPU
type Cpu struct {
	// VendorId the CPU vendor ID
	VendorId string
	// ModelName the CPU model
	ModelName string
	// CpuCores the number of cores for the CPU
	CpuCores uint64
	// CpuLogicalProcessors the number of logical core for the CPU
	CpuLogicalProcessors uint64
	// Mhz the frequency for the CPU (Not available on ARM)
	Mhz float64
	// CacheSizeBytes the cache size for the CPU (Linux only)
	CacheSizeBytes uint64
	// Family the CPU family
	Family string
	// Model the CPU model name
	Model string
	// Stepping the CPU stepping
	Stepping string

	// CpuPkgs the CPU pkg count (Windows only)
	CpuPkgs uint64
	// CpuNumaNodes the CPU numa node count (Windows only)
	CpuNumaNodes uint64
	// CacheSizeL1Bytes the CPU L1 cache size (Windows only)
	CacheSizeL1Bytes uint64
	// CacheSizeL2Bytes the CPU L2 cache size (Windows only)
	CacheSizeL2Bytes uint64
	// CacheSizeL3 the CPU L3 cache size (Windows only)
	CacheSizeL3Bytes uint64
}

const name = "cpu"

func (self *Cpu) Name() string {
	return name
}

func (self *Cpu) Collect() (result interface{}, err error) {
	result, err = getCpuInfo()
	return
}

// Get returns a Cpu struct already initialized, a list of warnings and an error. The method will try to collect as much
// metadata as possible, an error is returned if nothing could be collected. The list of warnings contains errors if
// some metadata could not be collected.
func Get() (*Cpu, []string, error) {
	cpuInfo, err := getCpuInfo()
	if err != nil {
		return nil, nil, err
	}

	warnings := []string{}
	c := &Cpu{}

	c.VendorId = utils.GetString(cpuInfo, "vendor_id")
	c.ModelName = utils.GetString(cpuInfo, "model_name")
	c.Family = utils.GetString(cpuInfo, "family")
	c.Model = utils.GetString(cpuInfo, "model")
	c.Stepping = utils.GetString(cpuInfo, "stepping")

	// We serialize int to string in the windows version of 'GetCpuInfo' and back to int here. This is less than
	// ideal but we don't want to break backward compatibility for now. The entire gohai project needs a rework but
	// for now we simply adding typed field to avoid using maps of interface..
	c.CpuPkgs = utils.GetUint64(cpuInfo, "cpu_pkgs", &warnings)
	c.CpuNumaNodes = utils.GetUint64(cpuInfo, "cpu_numa_nodes", &warnings)
	c.CacheSizeL1Bytes = utils.GetUint64(cpuInfo, "cache_size_l1", &warnings)
	c.CacheSizeL2Bytes = utils.GetUint64(cpuInfo, "cache_size_l2", &warnings)
	c.CacheSizeL3Bytes = utils.GetUint64(cpuInfo, "cache_size_l3", &warnings)

	c.CpuCores = utils.GetUint64(cpuInfo, "cpu_cores", &warnings)
	c.CpuLogicalProcessors = utils.GetUint64(cpuInfo, "cpu_logical_processors", &warnings)
	c.Mhz = utils.GetFloat64(cpuInfo, "mhz", &warnings)

	// cache_size uses the format '9216 KB'
	cacheSizeString := strings.Split(utils.GetString(cpuInfo, "cache_size"), " ")[0]
	cacheSizeBytes, err := strconv.ParseUint(cacheSizeString, 10, 64)
	if err == nil {
		c.CacheSizeBytes = cacheSizeBytes * 1024
	} else {
		warnings = append(warnings, fmt.Sprintf("could not collect cache size: %s", err))
	}

	return c, warnings, nil
}
