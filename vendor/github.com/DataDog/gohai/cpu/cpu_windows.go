package cpu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var getCpuInfo = GetCpuInfo

// Values that need to be multiplied by the number of physical processors
var perPhysicalProcValues = []string{
	"cpu_cores",
	"cpu_logical_processors",
}

const ERROR_INSUFFICIENT_BUFFER syscall.Errno = 122
const registryHive = "HARDWARE\\DESCRIPTION\\System\\CentralProcessor\\0"

type CACHE_DESCRIPTOR struct {
	Level         uint8
	Associativity uint8
	LineSize      uint16
	Size          uint32
	cacheType     uint32
}
type SYSTEM_LOGICAL_PROCESSOR_INFORMATION struct {
	ProcessorMask uintptr
	Relationship  int // enum (int)
	// in the Windows header, this is a union of a byte, a DWORD,
	// and a CACHE_DESCRIPTOR structure
	dataunion [16]byte
}

//.const SYSTEM_LOGICAL_PROCESSOR_INFORMATION_SIZE = 32

type GROUP_AFFINITY struct {
	Mask     uintptr
	Group    uint16
	Reserved [3]uint16
}
type NUMA_NODE_RELATIONSHIP struct {
	NodeNumber uint32
	Reserved   [20]uint8
	GroupMask  GROUP_AFFINITY
}
type CACHE_RELATIONSHIP struct {
	Level         uint8
	Associativity uint8
	LineSize      uint16
	CacheSize     uint32
	CacheType     int // enum in C
	Reserved      [20]uint8
	GroupMask     GROUP_AFFINITY
}

type PROCESSOR_GROUP_INFO struct {
	MaximumProcessorCount uint8
	ActiveProcessorCount  uint8
	Reserved              [38]uint8
	ActiveProcessorMask   uintptr
}
type GROUP_RELATIONSHIP struct {
	MaximumGroupCount uint16
	ActiveGroupCount  uint16
	Reserved          [20]uint8
	// variable size array of PROCESSOR_GROUP_INFO
}
type PROCESSOR_RELATIONSHIP struct {
	Flags           uint8
	EfficiencyClass uint8
	wReserved       [20]uint8
	GroupCount      uint16
	// what follows is an array of zero or more GROUP_AFFINITY structures
}

type SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX struct {
	Relationship int
	Size         uint32
	// what follows is a C union of
	// PROCESSOR_RELATIONSHIP,
	// NUMA_NODE_RELATIONSHIP,
	// CACHE_RELATIONSHIP,
	// GROUP_RELATIONSHIP
}

const RelationProcessorCore = 0
const RelationNumaNode = 1
const RelationCache = 2
const RelationProcessorPackage = 3
const RelationGroup = 4

type SYSTEM_INFO struct {
	wProcessorArchitecture  uint16
	wReserved               uint16
	dwPageSize              uint32
	lpMinApplicationAddress *uint32
	lpMaxApplicationAddress *uint32
	dwActiveProcessorMask   uintptr
	dwNumberOfProcessors    uint32
	dwProcessorType         uint32
	dwAllocationGranularity uint32
	wProcessorLevel         uint16
	wProcessorRevision      uint16
}

type CPU_INFO struct {
	numaNodeCount       int    // number of NUMA nodes
	pkgcount            int    // number of packages (physical CPUS)
	corecount           int    // total number of cores
	logicalcount        int    // number of logical CPUS
	l1CacheSize         uint32 // layer 1 cache size
	l2CacheSize         uint32 // layer 2 cache size
	l3CacheSize         uint32 // layer 3 cache size
	relationGroups      int    // number of cpu relation groups
	maxProcsInGroups    int    // max number of processors
	activeProcsInGroups int    // active processors

}

func countBits(num uint64) (count int) {
	count = 0
	for num > 0 {
		if (num & 0x1) == 1 {
			count++
		}
		num >>= 1
	}
	return
}

func getSystemInfo() (si SYSTEM_INFO) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var gsi = mod.NewProc("GetSystemInfo")

	gsi.Call(uintptr(unsafe.Pointer(&si)))
	return
}

// GetCpuInfo returns map of interesting bits of information about the CPU
func GetCpuInfo() (cpuInfo map[string]string, err error) {

	cpuInfo = make(map[string]string)

	cpus, _ := computeCoresAndProcessors()
	si := getSystemInfo()

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		registryHive,
		registry.QUERY_VALUE)
	defer k.Close()
	dw, _, err := k.GetIntegerValue("~MHz")
	cpuInfo["mhz"] = strconv.Itoa(int(dw))

	s, _, err := k.GetStringValue("ProcessorNameString")
	cpuInfo["model_name"] = s

	cpuInfo["cpu_pkgs"] = strconv.Itoa(cpus.pkgcount)
	cpuInfo["cpu_numa_nodes"] = strconv.Itoa(cpus.numaNodeCount)
	cpuInfo["cpu_cores"] = strconv.Itoa(cpus.corecount)
	cpuInfo["cpu_logical_processors"] = strconv.Itoa(cpus.logicalcount)

	s, _, err = k.GetStringValue("VendorIdentifier")
	cpuInfo["vendor_id"] = s

	s, _, err = k.GetStringValue("Identifier")
	cpuInfo["family"] = extract(s, "Family")

	cpuInfo["model"] = strconv.Itoa(int((si.wProcessorRevision >> 8) & 0xFF))
	cpuInfo["stepping"] = strconv.Itoa(int(si.wProcessorRevision & 0xFF))

	cpuInfo["cache_size_l1"] = strconv.Itoa(int(cpus.l1CacheSize))
	cpuInfo["cache_size_l2"] = strconv.Itoa(int(cpus.l2CacheSize))
	cpuInfo["cache_size_l3"] = strconv.Itoa(int(cpus.l3CacheSize))

	return
}

func extract(caption, field string) string {
	re := regexp.MustCompile(fmt.Sprintf("%s [0-9]* ", field))
	return strings.Split(re.FindStringSubmatch(caption)[0], " ")[1]
}
