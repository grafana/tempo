package fs

import (
	"os"
	"testing"
)

const devNullCount = 500

func openDevNullFiles(b *testing.B, n int) []*os.File {
	var arr []*os.File
	for i := 0; i < n; i++ {
		fh, err := os.Open("/dev/null")
		if err != nil {
			b.Fatalf("Cannot open /dev/null.")
		}
		arr = append(arr, fh)
	}
	return arr

}
func closeDevNullFiles(b *testing.B, arr []*os.File) {
	for i := range arr {
		err := arr[i].Close()
		if err != nil {
			b.Fatalf("Cannot close /dev/null.")
		}
	}
}

func BenchmarkReadDirNames(b *testing.B) {
	arr := openDevNullFiles(b, devNullCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		names, err := ReadDirNames("/proc/self/fd")
		if err != nil {
			b.Fatalf("ReadDirNames failed: %v", err)
		}
		count := len(names)
		if count < devNullCount || count > devNullCount+10 {
			b.Fatalf("ReadDirNames failed: count=%d", count)
		}
	}
	b.StopTimer()

	closeDevNullFiles(b, arr)
}

func BenchmarkReadDirCount(b *testing.B) {
	arr := openDevNullFiles(b, devNullCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count, err := ReadDirCount("/proc/self/fd")
		if err != nil {
			b.Fatalf("ReadDirCount failed: %v", err)
		}
		if count < devNullCount || count > devNullCount+10 {
			b.Fatalf("ReadDirCount failed: count=%d", count)
		}
	}
	b.StopTimer()

	closeDevNullFiles(b, arr)
}

func TestReadDirNames(t *testing.T) {
	names, err := ReadDirNames("/proc/self/fd")
	if err != nil {
		t.Fatalf("ReadDirNames failed: %v", err)
	}
	count, err := ReadDirCount("/proc/self/fd")
	if err != nil {
		t.Fatalf("ReadDirCount failed: %v", err)
	}
	if len(names) != count {
		t.Fatalf("ReadDirNames and ReadDirCount give inconsitent results: %d != %d", len(names), count)
	}

}
