package cpu

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

const SYSTEM_LOGICAL_PROCESSOR_INFORMATION_SIZE = 24

func getSystemLogicalProcessorInformationSize() int {
	return SYSTEM_LOGICAL_PROCESSOR_INFORMATION_SIZE
}
func byteArrayToProcessorStruct(data []byte) (info SYSTEM_LOGICAL_PROCESSOR_INFORMATION) {
	info.ProcessorMask = uintptr(binary.LittleEndian.Uint32(data))
	info.Relationship = int(binary.LittleEndian.Uint32(data[4:]))
	copy(info.dataunion[0:16], data[8:24])
	return
}

func computeCoresAndProcessors() (cpuInfo CPU_INFO, err error) {
	var mod = syscall.NewLazyDLL("kernel32.dll")
	var getProcInfo = mod.NewProc("GetLogicalProcessorInformation")
	var buflen uint32 = 0
	err = syscall.Errno(0)
	// first, figure out how much we need
	status, _, err := getProcInfo.Call(uintptr(0),
		uintptr(unsafe.Pointer(&buflen)))
	if status == 0 {
		if err != ERROR_INSUFFICIENT_BUFFER {
			// only error we're expecing here is insufficient buffer
			// anything else is a failure
			return
		}
	} else {
		// this shouldn't happen. Errno won't be set (because the function)
		// succeeded.  So just return something to indicate we've failed
		err = syscall.Errno(2)
		return
	}
	buf := make([]byte, buflen)
	status, _, err = getProcInfo.Call(uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&buflen)))
	if status == 0 {
		return
	}
	// walk through each of the buffers

	for i := 0; uint32(i) < buflen; i += getSystemLogicalProcessorInformationSize() {
		info := byteArrayToProcessorStruct(buf[i : i+getSystemLogicalProcessorInformationSize()])

		switch info.Relationship {
		case RelationNumaNode:
			cpuInfo.numaNodeCount++

		case RelationProcessorCore:
			cpuInfo.corecount++
			cpuInfo.logicalcount += countBits(uint64(info.ProcessorMask))

		case RelationProcessorPackage:
			cpuInfo.pkgcount++
		}
	}
	return
}
