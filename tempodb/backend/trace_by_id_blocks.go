package backend

import (
	"fmt"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
)

// TraceByIDBlocksFromMetas dedupes dedicated columns because a tenant's blocks usually share them.
func TraceByIDBlocksFromMetas(metas []*BlockMeta) (*tempopb.TraceByIDBlocks, error) {
	blocks := &tempopb.TraceByIDBlocks{
		Blocks: make([]*tempopb.TraceByIDBlock, 0, len(metas)),
	}
	columnsIdx := map[uint64]uint32{}

	for _, m := range metas {
		var idx uint32
		if len(m.DedicatedColumns) > 0 {
			hash := m.DedicatedColumns.Hash()
			var ok bool
			if idx, ok = columnsIdx[hash]; !ok {
				cols, err := m.DedicatedColumns.ToTempopb()
				if err != nil {
					return nil, fmt.Errorf("error converting dedicated columns for block %s: %w", m.BlockID, err)
				}
				blocks.DedicatedColumns = append(blocks.DedicatedColumns, &tempopb.TraceByIDDedicatedColumns{Columns: cols})

				idx = uint32(len(blocks.DedicatedColumns))
				columnsIdx[hash] = idx
			}
		}

		blocks.Blocks = append(blocks.Blocks, &tempopb.TraceByIDBlock{
			BlockID:               m.BlockID[:],
			Version:               m.Version,
			StartTimeUnixNano:     unixNano(m.StartTime),
			Size_:                 m.Size_,
			CompactionLevel:       m.CompactionLevel,
			BloomShardCount:       m.BloomShardCount,
			FooterSize:            m.FooterSize,
			DedicatedColumnsIndex: idx,
		})
	}

	return blocks, nil
}

// MetasFromTraceByIDBlocks is the inverse of TraceByIDBlocksFromMetas.
func MetasFromTraceByIDBlocks(blocks *tempopb.TraceByIDBlocks, tenantID string) ([]*BlockMeta, error) {
	columns := make([]DedicatedColumns, 0, len(blocks.DedicatedColumns))
	for _, c := range blocks.DedicatedColumns {
		cols, err := DedicatedColumnsFromTempopb(c.Columns)
		if err != nil {
			return nil, err
		}
		columns = append(columns, cols)
	}

	metas := make([]*BlockMeta, 0, len(blocks.Blocks))
	for _, b := range blocks.Blocks {
		// unsent BlockMeta fields stay zero, add them to TraceByIDBlock in proto if trace by id path needs them.
		m := &BlockMeta{
			Version:         b.Version,
			TenantID:        tenantID,
			StartTime:       fromUnixNano(b.StartTimeUnixNano),
			Size_:           b.Size_,
			CompactionLevel: b.CompactionLevel,
			BloomShardCount: b.BloomShardCount,
			FooterSize:      b.FooterSize,
		}
		if err := m.BlockID.Unmarshal(b.BlockID); err != nil {
			return nil, fmt.Errorf("invalid block id: %w", err)
		}
		if b.DedicatedColumnsIndex > uint32(len(columns)) {
			return nil, fmt.Errorf("invalid dedicated columns index %d for block %s", b.DedicatedColumnsIndex, m.BlockID)
		}
		if b.DedicatedColumnsIndex > 0 {
			m.DedicatedColumns = columns[b.DedicatedColumnsIndex-1]
		}
		metas = append(metas, m)
	}

	return metas, nil
}

// unixNano maps the zero time to 0 because its UnixNano overflows int64.
func unixNano(t time.Time) uint64 {
	if t.IsZero() {
		return 0
	}
	return uint64(t.UnixNano())
}

func fromUnixNano(ns uint64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(ns)).UTC()
}
