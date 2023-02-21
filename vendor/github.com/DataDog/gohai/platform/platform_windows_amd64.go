package platform

import (
	"encoding/binary"
	"unsafe"
)

type WKSTA_INFO_100 struct {
	wki100_platform_id  uint32
	wki100_computername string
	wki100_langroup     string
	wki100_ver_major    uint32
	wki100_ver_minor    uint32
}

type SERVER_INFO_101 struct {
	sv101_platform_id   uint32
	sv101_name          string
	sv101_version_major uint32
	sv101_version_minor uint32
	sv101_type          uint32
	sv101_comment       string
}

func byteArrayToWksaInfo(data []byte) (info WKSTA_INFO_100) {
	info.wki100_platform_id = binary.LittleEndian.Uint32(data)

	// the specified return type of wki100_platform_id is uint32. However,
	// due to 64 bit packing, we actually have to skip 8 bytes.

	// if necessary, convert the pointer to a c-string into a GO string.
	// Not using at this time.  However, leaving as  a placeholder, to
	// show why we're skipping 8 bytes of the buffer here...

	//addr := (*byte)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint64(data[8:]))))
	//info.wki100_computername = addr

	// ... and again here for the lan group name.
	//stringptr = (*[]byte)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint64(data[16:]))))
	//info.wki100_langroup = convert_windows_string(stringptr)

	info.wki100_ver_major = binary.LittleEndian.Uint32(data[24:])
	info.wki100_ver_minor = binary.LittleEndian.Uint32(data[28:])
	return
}
func platGetVersion(outdata *byte) (maj uint64, min uint64, err error) {
	var info WKSTA_INFO_100
	var dataptr []byte
	dataptr = (*[32]byte)(unsafe.Pointer(outdata))[:]

	info = byteArrayToWksaInfo(dataptr)
	maj = uint64(info.wki100_ver_major)
	min = uint64(info.wki100_ver_minor)
	return
}

func platGetServerInfo(data *byte) (si101 SERVER_INFO_101) {
	var outdata []byte
	outdata = (*[40]byte)(unsafe.Pointer(data))[:]
	si101.sv101_platform_id = binary.LittleEndian.Uint32(outdata)

	// due to 64 bit packing, skip 8 bytes to get to the name string
	//stringptr := *(*[]uint16)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint64(outdata[8:]))))
	//si101.sv101_name = convert_windows_string(stringptr)

	si101.sv101_version_major = binary.LittleEndian.Uint32(outdata[16:])
	si101.sv101_version_minor = binary.LittleEndian.Uint32(outdata[20:])
	si101.sv101_type = binary.LittleEndian.Uint32(outdata[24:])

	// again skip 4 more for byte packing, so start at 32
	//stringptr = (*[]uint16)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint32(outdata[32:]))))
	//si101.sv101_comment = convert_windows_string(*stringptr)
	return
}
