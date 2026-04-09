package util

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/grafana/dskit/user"
	jaeger "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	jaegerTrans "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

const (
	// vultureBlobSize is the exact size in bytes for the Blob01 attribute at each level.
	vultureBlobSize = 100
)

var (
	// maxBatchesPerWrite is used when writing and reading, and needs to match so
	// that we get the expected number of batches on a trace.  A value larger
	// than 10 here results in vulture writing traces that exceed the maximum
	// trace size.
	maxBatchesPerWrite int64 = 10

	// maxBatchesPerWrite is the maximum number of time-delayed writes for a trace.
	maxLongWritesPerTrace int64 = 3
)

// TraceInfo is used to construct synthetic traces and manage the expectations.
type TraceInfo struct {
	timestamp           time.Time
	r                   *rand.Rand
	traceIDHigh         int64
	traceIDLow          int64
	longWritesRemaining int64
	tempoOrgID          string
}

// JaegerClient is an interface used to mock the underlying client in tests.
type JaegerClient interface {
	EmitBatch(ctx context.Context, b *jaeger.Batch) error
}

// NewTraceInfo is used to produce a new TraceInfo.
func NewTraceInfo(timestamp time.Time, tempoOrgID string) *TraceInfo {
	r := newRand(timestamp)

	return &TraceInfo{
		timestamp:           timestamp,
		r:                   r,
		traceIDHigh:         r.Int63(),
		traceIDLow:          r.Int63(),
		longWritesRemaining: r.Int63n(maxLongWritesPerTrace),
		tempoOrgID:          tempoOrgID,
	}
}

// NewTraceInfos creates multiple trace infos with slightly different timestamps to ensure
// different trace seeds are used.
func NewTraceInfos(timestamp time.Time, count int, tempoOrgID string) []*TraceInfo {
	infos := make([]*TraceInfo, 0, count)
	for i := 0; i < count; i++ {
		ts := timestamp.Add(time.Duration(i) * time.Nanosecond)
		infos = append(infos, NewTraceInfo(ts, tempoOrgID))
	}
	return infos
}

func NewTraceInfoWithMaxLongWrites(timestamp time.Time, maxLongWrites int64, tempoOrgID string) *TraceInfo {
	r := newRand(timestamp)

	return &TraceInfo{
		timestamp:           timestamp,
		r:                   r,
		traceIDHigh:         r.Int63(),
		traceIDLow:          r.Int63(),
		longWritesRemaining: maxLongWrites,
		tempoOrgID:          tempoOrgID,
	}
}

func (t *TraceInfo) Ready(now time.Time, writeBackoff, longWriteBackoff time.Duration) bool {
	// Don't use the last time interval to allow the write loop to finish before
	// we try to read it.
	if t.timestamp.After(now.Add(-writeBackoff)) {
		return false
	}

	// Compare a new instance with the same timestamp to know how many longWritesRemaining.
	totalWrites := NewTraceInfo(t.timestamp, t.tempoOrgID).longWritesRemaining
	// We are not ready if not all writes have had a chance to send.
	lastWrite := t.timestamp.Add(time.Duration(totalWrites) * longWriteBackoff)
	return !now.Before(lastWrite.Add(longWriteBackoff))
}

func (t *TraceInfo) Timestamp() time.Time {
	return t.timestamp
}

func (t *TraceInfo) TraceID() ([]byte, error) {
	return HexStringToTraceID(t.HexID())
}

func (t *TraceInfo) HexID() string {
	return fmt.Sprintf("%016x%016x", t.traceIDHigh, t.traceIDLow)
}

func (t *TraceInfo) LongWritesRemaining() int64 {
	return t.longWritesRemaining
}

func (t *TraceInfo) Done() {
	t.longWritesRemaining--
}

func (t *TraceInfo) EmitBatches(ctx context.Context, c JaegerClient) error {
	for i := int64(0); i < t.generateRandomInt(1, maxBatchesPerWrite); i++ {
		ctx := user.InjectOrgID(ctx, t.tempoOrgID)
		ctx, err := user.InjectIntoGRPCRequest(ctx)
		if err != nil {
			return fmt.Errorf("error injecting org id: %w", err)
		}

		err = c.EmitBatch(ctx, t.makeThriftBatch(t.traceIDHigh, t.traceIDLow))
		if err != nil {
			return fmt.Errorf("error pushing batch to Tempo: %w", err)
		}
	}

	return nil
}

// EmitAllBatches sends all the batches that would normally be sent at some
// interval when using EmitBatches.
func (t *TraceInfo) EmitAllBatches(c JaegerClient) error {
	err := t.EmitBatches(context.Background(), c)
	if err != nil {
		return err
	}

	for t.LongWritesRemaining() > 0 {
		t.Done()

		err := t.EmitBatches(context.Background(), c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TraceInfo) generateRandomInt(min, max int64) int64 {
	min++
	number := min + t.r.Int63n(max-min)
	return number
}

func (t *TraceInfo) makeThriftBatch(traceIDHigh, traceIDLow int64) *jaeger.Batch {
	var spans []*jaeger.Span
	count := t.generateRandomInt(1, 5)
	lastSpanID, nextSpanID := int64(0), int64(0)

	// Each span has the previous span as parent, creating a tree with a single branch per batch.
	for i := int64(0); i < count; i++ {
		nextSpanID = t.r.Int63()

		spanTags := t.generateRandomTagsWithPrefix("vulture")
		spanTags = append(spanTags, t.generateFixedAttributesWithPrefix("vulture")...)
		spanTags = append(spanTags, t.generateSpanWellKnownAttributes()...)

		spans = append(spans, &jaeger.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        nextSpanID,
			ParentSpanId:  lastSpanID,
			OperationName: fmt.Sprintf("vulture-%d", t.generateRandomInt(0, 100)),
			References:    nil,
			Flags:         0,
			StartTime:     t.timestamp.UnixMicro(),
			Duration:      t.generateRandomInt(0, 100),
			Tags:          spanTags,
			Logs:          t.generateRandomLogs(),
		})

		lastSpanID = nextSpanID
	}

	processTags := t.generateRandomTagsWithPrefix("vulture-process")
	processTags = append(processTags, t.generateFixedAttributesWithPrefix("vulture-process")...)
	processTags = append(processTags, t.generateResourceWellKnownAttributes()...)

	process := &jaeger.Process{
		ServiceName: "tempo-vulture",
		Tags:        processTags,
	}

	return &jaeger.Batch{Process: process, Spans: spans}
}

func (t *TraceInfo) generateRandomString() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, t.generateRandomInt(5, 20))
	for i := range s {
		s[i] = letters[t.r.Intn(len(letters))]
	}
	return string(s)
}

// generateRandomBlob returns a string of exactly size bytes of random data (same character set as other attributes).
func (t *TraceInfo) generateRandomBlob(size int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, size)
	for i := 0; i < size; i++ {
		s[i] = letters[t.r.Intn(len(letters))]
	}
	return string(s)
}

// generateFixedAttributesWithPrefix returns the fixed attributes with a prefix. Keys are lowercase with a hyphen before the numeric suffix (e.g. string-01, int-01, blob-01).
func (t *TraceInfo) generateFixedAttributesWithPrefix(prefix string) []*jaeger.Tag {
	var (
		numStrings = t.r.Intn(6)
		numBlobs   = t.r.Intn(2)
		numInts    = t.r.Intn(6)
		tags       = make([]*jaeger.Tag, 0, numStrings+numBlobs+numInts)
	)
	for i := 0; i < numStrings; i++ {
		tags = append(tags, &jaeger.Tag{Key: fmt.Sprintf("%s-string-%02d", prefix, i+1), VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())})
	}
	for i := 0; i < numBlobs; i++ {
		tags = append(tags, &jaeger.Tag{Key: fmt.Sprintf("%s-blob-%02d", prefix, i+1), VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomBlob(vultureBlobSize))})
	}
	for i := 0; i < numInts; i++ {
		tags = append(tags, &jaeger.Tag{Key: fmt.Sprintf("%s-int-%02d", prefix, i+1), VType: jaeger.TagType_LONG, VLong: int64Ptr(t.generateRandomInt(1, 1000000))})
	}
	return tags
}

func stringPtr(s string) *string { return &s }

func int64Ptr(n int64) *int64 { return &n }

func (t *TraceInfo) generateResourceWellKnownAttributes() []*jaeger.Tag {
	if t.r.Intn(2) == 0 {
		return nil
	}
	return []*jaeger.Tag{
		{Key: "cluster", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "namespace", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "pod", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "container", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "k8s.namespace.name", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "k8s.cluster.name", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "k8s.pod.name", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "k8s.container.name", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
	}
}

func (t *TraceInfo) generateSpanWellKnownAttributes() []*jaeger.Tag {
	if t.r.Intn(2) == 0 {
		return nil
	}
	return []*jaeger.Tag{
		{Key: "http.method", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "http.url", VType: jaeger.TagType_STRING, VStr: stringPtr(t.generateRandomString())},
		{Key: "http.status_code", VType: jaeger.TagType_LONG, VLong: int64Ptr(t.generateRandomInt(1, 500))},
	}
}

func (t *TraceInfo) generateRandomTagsWithPrefix(prefix string) []*jaeger.Tag {
	var tags []*jaeger.Tag
	count := t.generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		value := t.generateRandomString()
		tags = append(tags, &jaeger.Tag{
			Key:  fmt.Sprintf("%s-%d", prefix, i),
			VStr: &value,
		})
	}
	return tags
}

func (t *TraceInfo) generateRandomLogs() []*jaeger.Log {
	var logs []*jaeger.Log
	count := t.generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		logs = append(logs, &jaeger.Log{
			Timestamp: t.timestamp.UnixMicro(),
			Fields:    append(t.generateRandomTagsWithPrefix("vulture-event"), t.generateFixedAttributesWithPrefix("vulture-event")...),
		})
	}

	return logs
}

func (t *TraceInfo) ConstructTraceFromEpoch() (*tempopb.Trace, error) {
	trace := &tempopb.Trace{}

	// Create a new trace from our timestamp to ensure a fresh rand.Rand is used for consistency.
	info := NewTraceInfo(t.timestamp, t.tempoOrgID)

	addBatches := func(t *TraceInfo, trace *tempopb.Trace) error {
		for i := int64(0); i < t.generateRandomInt(1, maxBatchesPerWrite); i++ {
			batch := t.makeThriftBatch(t.traceIDHigh, t.traceIDLow)
			internalTrace, err := jaegerTrans.ThriftToTraces(batch)
			if err != nil {
				return err
			}
			conv, err := (&ptrace.ProtoMarshaler{}).MarshalTraces(internalTrace)
			if err != nil {
				return err
			}

			t := tempopb.Trace{}
			err = t.Unmarshal(conv)
			if err != nil {
				return err
			}

			// Due to the several transforms above, some manual mangling is required to
			// get the parentSpanID to match.  In the case of an empty []byte in place
			// for the ParentSpanId, we set to nil here to ensure that the final result
			// matches the json.Unmarshal value when tempo is queried.
			for _, b := range t.ResourceSpans {
				for _, l := range b.ScopeSpans {
					for _, s := range l.Spans {
						if len(s.GetParentSpanId()) == 0 {
							s.ParentSpanId = nil
						}
					}
				}
			}

			trace.ResourceSpans = append(trace.ResourceSpans, t.ResourceSpans...)
		}

		return nil
	}

	err := addBatches(info, trace)
	if err != nil {
		return nil, err
	}

	for info.longWritesRemaining > 0 {
		info.Done()
		err := addBatches(info, trace)
		if err != nil {
			return nil, err
		}
	}

	return trace, nil
}

// RandomAttrFromTrace returns a random attribute from the trace for use in search validation.
// Integer attributes are never chosen: they are not unique enough for search.
func RandomAttrFromTrace(t *tempopb.Trace) *v1common.KeyValue {
	r := newRand(time.Now())

	if len(t.ResourceSpans) == 0 {
		return nil
	}
	batch := randFrom(r, t.ResourceSpans)

	// maybe choose resource attribute
	res := batch.Resource
	if len(res.Attributes) > 0 && r.Int()%2 == 1 {
		attr := randFrom(r, res.Attributes)
		// skip service.name because service names have low cardinality and produce queries with
		// too many results in tempo-vulture
		if attr.Key != "service.name" {
			if attr.Value == nil {
				return attr
			}
			if _, ok := attr.Value.Value.(*v1common.AnyValue_IntValue); !ok {
				return attr
			}
		}
	}

	if len(batch.ScopeSpans) == 0 {
		return nil
	}
	ss := randFrom(r, batch.ScopeSpans)

	if len(ss.Spans) == 0 {
		return nil
	}
	span := randFrom(r, ss.Spans)

	if len(span.Attributes) == 0 {
		return nil
	}

	// Pick only from non-integer attributes (integers are not unique enough for search).
	nonIntAttrs := make([]*v1common.KeyValue, 0, len(span.Attributes))
	for _, a := range span.Attributes {
		if a.Value == nil {
			nonIntAttrs = append(nonIntAttrs, a)
		} else if _, ok := a.Value.Value.(*v1common.AnyValue_IntValue); !ok {
			nonIntAttrs = append(nonIntAttrs, a)
		}
	}
	if len(nonIntAttrs) == 0 {
		return nil
	}
	return randFrom(r, nonIntAttrs)
}

func randFrom[T any](r *rand.Rand, s []T) T {
	return s[r.Intn(len(s))]
}

func newRand(t time.Time) *rand.Rand {
	return rand.New(rand.NewSource(t.UnixNano())) // nolint:gosec // G404: Use of weak random number generator
}
