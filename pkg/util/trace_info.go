package util

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	jaegerTrans "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/model/otlp"

	"github.com/grafana/tempo/pkg/tempopb"
	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

var (
	// maxBatchesPerWrite is used when writing and reading, and needs to match so
	// that we get the expected number of batches on a trace.  A value larger
	// than 10 here results in vulture writing traces that exceed the maximum
	// trace size.
	maxBatchesPerWrite int64 = 10

	//maxBatchesPerWrite is the maximum number of time-delayed writes for a trace.
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

func (t *TraceInfo) Ready(now time.Time, writeBackoff, longWriteBackoff time.Duration) bool {

	// Don't use the last time interval to allow the write loop to finish before
	// we try to read it.
	if t.timestamp.After(now.Add(-writeBackoff)) {
		return false
	}

	// Compare a new instance with the same timstamp to know how many longWritesRemaining.
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

func (t *TraceInfo) EmitBatches(c *jaeger_grpc.Reporter) error {
	for i := int64(0); i < t.generateRandomInt(1, maxBatchesPerWrite); i++ {
		ctx := user.InjectOrgID(context.Background(), t.tempoOrgID)
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
func (t *TraceInfo) EmitAllBatches(c *jaeger_grpc.Reporter) error {
	err := t.EmitBatches(c)
	if err != nil {
		return err
	}

	for t.LongWritesRemaining() > 0 {
		t.Done()

		err := t.EmitBatches(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TraceInfo) generateRandomInt(min int64, max int64) int64 {
	min++
	number := min + t.r.Int63n(max-min)
	return number
}

func (t *TraceInfo) makeThriftBatch(TraceIDHigh int64, TraceIDLow int64) *thrift.Batch {
	var spans []*thrift.Span
	count := t.generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		spans = append(spans, &thrift.Span{
			TraceIdLow:    TraceIDLow,
			TraceIdHigh:   TraceIDHigh,
			SpanId:        t.r.Int63(),
			ParentSpanId:  0,
			OperationName: fmt.Sprintf("vulture-%d", t.generateRandomInt(0, 100)),
			References:    nil,
			Flags:         0,
			StartTime:     unixMicro(t.timestamp),
			Duration:      t.generateRandomInt(0, 100),
			Tags:          t.generateRandomTags(),
			Logs:          t.generateRandomLogs(),
		})
	}

	process := &thrift.Process{
		ServiceName: "tempo-vulture",
	}

	return &thrift.Batch{Process: process, Spans: spans}
}

func (t *TraceInfo) generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, t.generateRandomInt(5, 20))
	for i := range s {
		s[i] = letters[t.r.Intn(len(letters))]
	}
	return string(s)
}

func (t *TraceInfo) generateRandomTags() []*thrift.Tag {
	var tags []*thrift.Tag
	count := t.generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		value := t.generateRandomString()
		tags = append(tags, &thrift.Tag{
			Key:  fmt.Sprintf("vulture-%d", i),
			VStr: &value,
		})
	}
	return tags
}

func (t *TraceInfo) generateRandomLogs() []*thrift.Log {
	var logs []*thrift.Log
	count := t.generateRandomInt(1, 5)
	for i := int64(0); i < count; i++ {
		logs = append(logs, &thrift.Log{
			Timestamp: unixMicro(t.timestamp),
			Fields:    t.generateRandomTags(),
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
			internalTrace := jaegerTrans.ThriftBatchToInternalTraces(batch)
			conv, err := otlp.NewProtobufTracesMarshaler().MarshalTraces(internalTrace)
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
			for _, b := range t.Batches {
				for _, l := range b.InstrumentationLibrarySpans {
					for _, s := range l.Spans {
						if len(s.GetParentSpanId()) == 0 {
							s.ParentSpanId = nil
						}
					}
				}
			}

			trace.Batches = append(trace.Batches, t.Batches...)
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

func RandomAttrFromTrace(t *tempopb.Trace) *v1common.KeyValue {
	r := newRand(time.Now())

	if len(t.Batches) == 0 {
		return nil
	}
	iBatch := r.Intn(len(t.Batches))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans) == 0 {
		return nil
	}
	iSpans := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans) == 0 {
		return nil
	}
	iSpan := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans))

	if len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes) == 0 {
		return nil
	}
	iAttr := r.Intn(len(t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes))

	return t.Batches[iBatch].InstrumentationLibrarySpans[iSpans].Spans[iSpan].Attributes[iAttr]
}

func newRand(t time.Time) *rand.Rand {
	return rand.New(rand.NewSource(t.Unix()))
}

// todo: Switch to time.UnixMicro() once Google Cloud Functions supports a go runtime > 1.16
func unixMicro(t time.Time) int64 {
	return t.UnixNano() / int64(time.Microsecond)
}
