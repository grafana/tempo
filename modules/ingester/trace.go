package ingester

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/model"
)

type liveTrace struct {
	batches    [][]byte
	lastAppend time.Time
	traceID    []byte
	start      uint32
	end        uint32
	decoder    model.SegmentDecoder
}

func newTrace(traceID []byte) *liveTrace {
	return &liveTrace{
		batches:    make([][]byte, 0, 10), // 10 for luck
		lastAppend: time.Now(),
		traceID:    traceID,
		decoder:    model.MustNewSegmentDecoder(model.CurrentEncoding),
	}
}

func (t *liveTrace) Push(_ context.Context, instanceID string, trace []byte) error {
	t.lastAppend = time.Now()

	start, end, err := t.decoder.FastRange(trace)
	if err != nil {
		return fmt.Errorf("failed to get range while adding segment: %w", err)
	}
	t.batches = append(t.batches, trace)
	if t.start == 0 || start < t.start {
		t.start = start
	}
	if t.end == 0 || end > t.end {
		t.end = end
	}

	return nil
}

func (t *liveTrace) Size() uint64 {
	size := uint64(0)
	for _, batch := range t.batches {
		size += uint64(len(batch))
	}
	return size
}
