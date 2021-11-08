package ingester

import (
	"context"
	"encoding/hex"
	"time"

	cortex_util "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	metricTraceSearchBytesDiscardedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "ingester_trace_search_bytes_discarded_total",
		Help:      "The total number of trace search bytes discarded per tenant.",
	}, []string{"tenant"})
)

type trace struct {
	traceBytes   *tempopb.TraceBytes
	lastAppend   time.Time
	traceID      []byte
	maxBytes     int
	currentBytes int

	// List of flatbuffers
	searchData         [][]byte
	maxSearchBytes     int
	currentSearchBytes int
}

func newTrace(traceID []byte, maxBytes int, maxSearchBytes int) *trace {
	return &trace{
		traceBytes: &tempopb.TraceBytes{
			Traces: make([][]byte, 0, 10), // 10 for luck
		},
		lastAppend:     time.Now(),
		traceID:        traceID,
		maxBytes:       maxBytes,
		maxSearchBytes: maxSearchBytes,
	}
}

func (t *trace) Push(_ context.Context, instanceID string, trace []byte, searchData []byte) error {
	t.lastAppend = time.Now()
	if t.maxBytes != 0 {
		reqSize := len(trace)
		if t.currentBytes+reqSize > t.maxBytes {
			return status.Errorf(codes.FailedPrecondition, "%s max size of trace (%d) exceeded while adding %d bytes to trace %s", overrides.ErrorPrefixTraceTooLarge, t.maxBytes, reqSize, hex.EncodeToString(t.traceID))
		}

		t.currentBytes += reqSize
	}

	t.traceBytes.Traces = append(t.traceBytes.Traces, trace)

	if searchDataSize := len(searchData); searchDataSize > 0 {
		// disable limit when set to 0
		if t.maxSearchBytes == 0 || t.currentSearchBytes+searchDataSize <= t.maxSearchBytes {
			t.searchData = append(t.searchData, searchData)
			t.currentSearchBytes += searchDataSize
		} else {
			// todo: info level since we are not expecting this limit to be hit, but calibrate accordingly in the future
			level.Info(cortex_util.Logger).Log("msg", "size of search data exceeded max search bytes limit", "maxSearchBytes", t.maxSearchBytes, "discardedBytes", searchDataSize)
			metricTraceSearchBytesDiscardedTotal.WithLabelValues(instanceID).Add(float64(searchDataSize))
		}
	}

	return nil
}
