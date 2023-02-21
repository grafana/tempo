package memory

import (
	"strconv"
	"syscall"
	"unsafe"
)

type MEMORYSTATUSEX struct {
	dwLength               uint32 // size of this structure
	dwMemoryLoad           uint32 // number 0-100 estimating %age of memory in use
	ulTotalPhys            uint64 // amount of physical memory
	ulAvailPhys            uint64 // amount of physical memory that can be used w/o flush to disk
	ulTotalPageFile        uint64 // current commit limit for system or process
	ulAvailPageFile        uint64 // amount of memory current process can commit
	ulTotalVirtual         uint64 // size of user-mode portion of VA space
	ulAvailVirtual         uint64 // amount of unreserved/uncommitted memory in ulTotalVirtual
	ulAvailExtendedVirtual uint64 // reserved (always zero)
}

func getMemoryInfo() (memoryInfo map[string]string, err error) {
	memoryInfo = make(map[string]string)

	mem, _, _, err := getMemoryInfoByte()
	if err == nil {
		memoryInfo["total"] = strconv.FormatUint(mem, 10)
	}
	return
}

func getMemoryInfoByte() (mem uint64, swap uint64, warnings []string, err error) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var getMem = mod.NewProc("GlobalMemoryStatusEx")

	var mem_struct MEMORYSTATUSEX

	mem_struct.dwLength = uint32(unsafe.Sizeof(mem_struct))

	status, _, err := getMem.Call(uintptr(unsafe.Pointer(&mem_struct)))
	if status != 0 {
		mem = mem_struct.ulTotalPhys
		err = nil
	}
	return
}
