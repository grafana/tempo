// +build linux
// +build arm64

package cpu

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// The Linux kernel does not include much useful information in /proc/cpuinfo
// for arm64, so we must dig further into the /sys tree and build a more
// accurate representation of the contained data, rather than relying on the
// simple analysis in cpu/cpu_linux_default.go.

// nodeNRegex recognizes directories named `nodeNN`
var nodeNRegex = regexp.MustCompile("^node[0-9]+$")

func getCpuInfo() (cpuInfo map[string]string, err error) {
	cpuInfo = make(map[string]string)

	procCpu, err := readProcCpuInfo()
	if err != nil {
		return nil, err
	}

	// we blithely assume that many of the CPU characteristics are the same for
	// all CPUs, so we can just use the first.
	firstCpu := procCpu[0]

	// determine vendor and model from CPU implementer / part
	if cpuVariantStr, ok := firstCpu["CPU implementer"]; ok {
		if cpuVariant, err := strconv.ParseUint(cpuVariantStr, 0, 64); err == nil {
			if cpuPartStr, ok := firstCpu["CPU part"]; ok {
				if cpuPart, err := strconv.ParseUint(cpuPartStr, 0, 64); err == nil {
					cpuInfo["model"] = cpuPartStr
					if impl, ok := hwVariant[cpuVariant]; ok {
						cpuInfo["vendor_id"] = impl.name
						if modelName, ok := impl.parts[cpuPart]; ok {
							cpuInfo["model_name"] = modelName
						} else {
							cpuInfo["model_name"] = cpuPartStr
						}
					} else {
						cpuInfo["vendor_id"] = cpuVariantStr
						cpuInfo["model_name"] = cpuPartStr
					}
				}
			}
		}
	}

	// ARM does not define a family
	cpuInfo["family"] = "none"

	// 'lscpu' represents the stepping as an rXpY string
	if cpuVariantStr, ok := firstCpu["CPU variant"]; ok {
		if cpuVariant, err := strconv.ParseUint(cpuVariantStr, 0, 64); err == nil {
			if cpuRevisionStr, ok := firstCpu["CPU revision"]; ok {
				if cpuRevision, err := strconv.ParseUint(cpuRevisionStr, 0, 64); err == nil {
					cpuInfo["stepping"] = fmt.Sprintf("r%dp%d", cpuVariant, cpuRevision)
				}
			}
		}
	}

	// Iterate over each processor and fetch additional information from /sys/devices/system/cpu
	cores := map[uint64]struct{}{}
	packages := map[uint64]struct{}{}
	cacheSizes := map[uint64]uint64{}
	for _, stanza := range procCpu {
		procID, err := strconv.ParseUint(stanza["processor"], 0, 64)
		if err != nil {
			continue
		}

		if coreID, ok := sysCpuInt(fmt.Sprintf("cpu%d/topology/core_id", procID)); ok {
			cores[coreID] = struct{}{}
		}

		if pkgID, ok := sysCpuInt(fmt.Sprintf("cpu%d/topology/physical_package_id", procID)); ok {
			packages[pkgID] = struct{}{}
		}

		// iterate over each cache this CPU can use
		i := 0
		for {
			if sharedList, ok := sysCpuList(fmt.Sprintf("cpu%d/cache/index%d/shared_cpu_list", procID, i)); ok {
				// we are scanning CPUs in order, so only count this cache if it's not shared with a
				// CPU that has already been scanned
				shared := false
				for sharedProcID := range sharedList {
					if sharedProcID < procID {
						shared = true
						break
					}
				}

				if !shared {
					if level, ok := sysCpuInt(fmt.Sprintf("cpu%d/cache/index%d/level", procID, i)); ok {
						if size, ok := sysCpuSize(fmt.Sprintf("cpu%d/cache/index%d/size", procID, i)); ok {
							cacheSizes[level] += size
						}
					}
				}
			} else {
				break
			}
			i++
		}
	}
	cpuInfo["cpu_pkgs"] = strconv.Itoa(len(packages))
	cpuInfo["cpu_cores"] = strconv.Itoa(len(cores))
	cpuInfo["cpu_logical_processors"] = strconv.Itoa(len(procCpu))
	cpuInfo["cache_size_l1"] = strconv.FormatUint(cacheSizes[1], 10)
	cpuInfo["cache_size_l2"] = strconv.FormatUint(cacheSizes[2], 10)
	cpuInfo["cache_size_l3"] = strconv.FormatUint(cacheSizes[3], 10)

	// cache_size uses the format '9216 KB'
	cpuInfo["cache_size"] = fmt.Sprintf("%d KB", (cacheSizes[1]+cacheSizes[2]+cacheSizes[3])/1024)

	// Count the number of NUMA nodes in /sys/devices/system/node
	nodes := 0
	if dirents, err := os.ReadDir("/sys/devices/system/node"); err == nil {
		for _, dirent := range dirents {
			if dirent.IsDir() && nodeNRegex.MatchString(dirent.Name()) {
				nodes++
			}
		}
	}
	cpuInfo["cpu_numa_nodes"] = strconv.Itoa(nodes)

	// ARM does not make the clock speed available
	// cpuInfo["mhz"]

	return cpuInfo, nil
}
