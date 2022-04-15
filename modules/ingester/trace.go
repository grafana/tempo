package ingester

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/util/log"
)

var (
	metricTraceSearchBytesDiscardedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_trace_search_bytes_discarded_total",
		Help:      "The total number of trace search bytes discarded per tenant.",
	}, []string{"tenant"})
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

	// List of flatbuffers
	searchData         [][]byte
	maxSearchBytes     int
	currentSearchBytes int
}

func newTrace(traceID []byte, maxBytes int, maxSearchBytes int) *liveTrace {
	return &liveTrace{
		batches:        make([][]byte, 0, 10), // 10 for luck
		lastAppend:     time.Now(),
		traceID:        traceID,
		maxBytes:       maxBytes,
		maxSearchBytes: maxSearchBytes,
		decoder:        model.MustNewSegmentDecoder(model.CurrentEncoding),
	}
}

func (t *liveTrace) Push(_ context.Context, instanceID string, trace []byte, searchData []byte) error {
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

	if searchDataSize := len(searchData); searchDataSize > 0 {
		// disable limit when set to 0
		if t.maxSearchBytes == 0 || t.currentSearchBytes+searchDataSize <= t.maxSearchBytes {
			t.searchData = append(t.searchData, searchData)
			t.currentSearchBytes += searchDataSize
		} else {
			// todo: info level since we are not expecting this limit to be hit, but calibrate accordingly in the future
			level.Info(log.Logger).Log("msg", "size of search data exceeded max search bytes limit", "maxSearchBytes", t.maxSearchBytes, "discardedBytes", searchDataSize)
			metricTraceSearchBytesDiscardedTotal.WithLabelValues(instanceID).Add(float64(searchDataSize))
		}
	}

	return nil
}
