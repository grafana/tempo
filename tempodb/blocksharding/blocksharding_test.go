package blocksharding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/tempodb/backend"
)

func TestPagesPerRequest(t *testing.T) {
	meta := func(size uint64, records uint32) *backend.BlockMeta {
		m := backend.NewBlockMeta("t", uuid.New(), "vParquet5")
		m.Size_, m.TotalRecords = size, records
		return m
	}

	for _, tc := range []struct {
		name            string
		meta            *backend.BlockMeta
		bytesPerRequest int
		want            int
	}{
		{"a block under the target is one request", meta(50, 4), 100, 4},
		{"one page per request when pages are target-sized", meta(400, 4), 100, 1},
		{"small pages are grouped up to the target", meta(400, 8), 100, 2},
		{"a page larger than the target still gets one", meta(1000, 2), 100, 1},
		{"no size means no answer", meta(0, 4), 100, 0},
		{"no records means no answer", meta(400, 0), 100, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, PagesPerRequest(tc.meta, tc.bytesPerRequest))
		})
	}
}
