package util

import "bytes"

// CRC32Equal compares CRC32 checksums.
func CRC32Equal(b []byte, c uint32) bool {
	return bytes.Equal(b, []byte{byte(0xff & (c >> 24)), byte(0xff & (c >> 16)), byte(0xff & (c >> 8)), byte(0xff & c)})
}
