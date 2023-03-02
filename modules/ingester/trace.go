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

	// byte limits
	maxBytes     int
	currentBytes int
}

func newTrace(traceID []byte, maxBytes int) *liveTrace {
	return &liveTrace{
		batches:    make([][]byte, 0, 10), // 10 for luck
		lastAppend: time.Now(),
		traceID:    traceID,
		maxBytes:   maxBytes,
		decoder:    model.MustNewSegmentDecoder(model.CurrentEncoding),
	}
}

func (t *liveTrace) Push(_ context.Context, instanceID string, trace []byte) error {
	t.lastAppend = time.Now()
	if t.maxBytes != 0 {
		reqSize := len(trace)
		if t.currentBytes+reqSize > t.maxBytes {
			return newTraceTooLargeError(t.traceID, instanceID, t.maxBytes, reqSize)
		}

		t.currentBytes += reqSize
	}

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
