package backend

import (
	"fmt"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
)

// TraceByIDBlocksFromMetas converts the blocks a trace by id job must search into their tempopb form.
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
			EndTimeUnixNano:       unixNano(m.EndTime),
			TotalObjects:          m.TotalObjects,
			Size_:                 m.Size_,
			CompactionLevel:       m.CompactionLevel,
			IndexPageSize:         m.IndexPageSize,
			TotalRecords:          m.TotalRecords,
			BloomShardCount:       m.BloomShardCount,
			FooterSize:            m.FooterSize,
			ReplicationFactor:     m.ReplicationFactor,
			DedicatedColumnsIndex: idx,
		})
	}

	return blocks, nil
}

// MetasFromTraceByIDBlocks converts the blocks sent with a trace by id job back into block metas of the tenant.
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
		m := &BlockMeta{
			Version:           b.Version,
			TenantID:          tenantID,
			StartTime:         fromUnixNano(b.StartTimeUnixNano),
			EndTime:           fromUnixNano(b.EndTimeUnixNano),
			TotalObjects:      b.TotalObjects,
			Size_:             b.Size_,
			CompactionLevel:   b.CompactionLevel,
			IndexPageSize:     b.IndexPageSize,
			TotalRecords:      b.TotalRecords,
			BloomShardCount:   b.BloomShardCount,
			FooterSize:        b.FooterSize,
			ReplicationFactor: b.ReplicationFactor,
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

// unixNano keeps the zero time as 0, its UnixNano is out of the int64 range.
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
