package backend

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkTenantIndexRetainedColumns measures live metadata after index writes,
// separate from temporary allocations and compression buffers.
func BenchmarkTenantIndexRetainedColumns(b *testing.B) {
	for _, layouts := range []int{10, 4096} {
		b.Run(fmt.Sprintf("layouts%d", layouts), func(b *testing.B) {
			payload, err := tenantIndexWriteFixture(4096, layouts).marshalPb()
			require.NoError(b, err)
			for range 2 {
				var warm TenantIndex
				require.NoError(b, warm.unmarshalPb(payload))
			}
			for b.Loop() {
				runtime.GC()
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				var live [32]*TenantIndex
				for i := range live {
					live[i] = new(TenantIndex)
					require.NoError(b, live[i].unmarshalPb(payload))
					_, err = live[i].marshalPb()
					require.NoError(b, err)
				}
				runtime.GC()
				runtime.GC()
				runtime.ReadMemStats(&after)
				b.ReportMetric(float64(int64(after.HeapAlloc)-int64(before.HeapAlloc))/float64(len(live)), "retained-B/index")
				runtime.KeepAlive(live)
				runtime.KeepAlive(payload)
			}
		})
	}
}
