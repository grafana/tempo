package common

import (
	"encoding/binary"
	"testing"
)

func BenchmarkIDMap(b *testing.B) {
	count := 100_000

	for i := 0; i < b.N; i++ {

		m := NewIDMap[int64](count)
		tid := make([]byte, 16)

		for j := 0; j < count; j++ {
			binary.BigEndian.PutUint64(tid, uint64(j))
			m.Set(tid, int64(j))
		}
	}
}

/*func BenchmarkIDMap2(b *testing.B) {
	for i := 0; i < b.N; i++ {

		m := NewIDMap2[int64]()
		tid := make([]byte, 16)
		for j := 0; j < 100_000; j++ {
			binary.BigEndian.PutUint64(tid, uint64(j))
			m.Set(tid, int64(j))
		}
	}
}*/
