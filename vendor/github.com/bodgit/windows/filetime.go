// Package windows is a collection of types native to Windows platforms but
// are useful on non-Windows platforms.
package windows

// Taken from golang.org/x/sys/windows

const offset int64 = 116444736000000000

// Filetime mirrors the Windows FILETIME structure which represents time
// as the number of 100-nanosecond intervals that have elapsed since
// 00:00:00 UTC, January 1, 1601. This code is taken from the
// golang.org/x/sys/windows package where it's not available for non-Windows
// platforms however various file formats and protocols pass this structure
// about so it's useful to have it available for interoperability purposes.
type Filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

// Nanoseconds returns Filetime ft in nanoseconds
// since Epoch (00:00:00 UTC, January 1, 1970).
func (ft *Filetime) Nanoseconds() int64 {
	// 100-nanosecond intervals since January 1, 1601
	nsec := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime)
	// change starting time to the Epoch (00:00:00 UTC, January 1, 1970)
	nsec -= offset
	// convert into nanoseconds
	nsec *= 100

	return nsec
}

// NsecToFiletime converts nanoseconds to the equivalent Filetime type.
func NsecToFiletime(nsec int64) (ft Filetime) {
	// convert into 100-nanosecond
	nsec /= 100
	// change starting time to January 1, 1601
	nsec += offset
	// split into high / low
	ft.LowDateTime = uint32(nsec & 0xffffffff)
	ft.HighDateTime = uint32(nsec >> 32 & 0xffffffff)

	return ft
}
