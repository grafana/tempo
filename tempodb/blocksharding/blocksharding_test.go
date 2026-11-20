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

func TestJobBytes(t *testing.T) {
	for _, tc := range []struct {
		name         string
		blockSize    uint64
		startPage    uint32
		pages        uint32
		totalRecords uint32
		want         uint64
	}{
		{"part of the block", 1000, 2, 2, 10, 200},
		{"whole block", 1000, 0, 10, 10, 1000},
		{"last job covers less than requested", 1000, 8, 4, 10, 200},
		{"job past the end of the block", 1000, 12, 2, 10, 0},
		{"bytes are not rounded down per row group", 1000, 0, 3, 7, 428},
		// not a realistic block, it guards that blockSize*rowGroups is not computed in one step, which overflows a uint64
		{"big block does not overflow", 100 << 40, 0, 400_000, 1_000_000, 40 << 40},
		{"no total records", 1000, 0, 2, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, JobBytes(tc.blockSize, tc.startPage, tc.pages, tc.totalRecords))
		})
	}
}
