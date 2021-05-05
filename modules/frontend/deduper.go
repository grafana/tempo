package frontend

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
)

const (
	warningTooManySpans = "cannot assign unique span ID, too many spans in the trace"
)

var (
	maxSpanID uint64 = 0xffffffffffffffff
)

func Deduper(logger log.Logger) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return spanIDDeduper{
			next:   next,
			logger: logger,
		}
	})
}

// This is copied over from Jaeger and modified to work for OpenTelemetry Trace data structure
// https://github.com/jaegertracing/jaeger/blob/12bba8c9b91cf4a29d314934bc08f4a80e43c042/model/adjuster/span_id_deduper.go
type spanIDDeduper struct {
	next      Handler
	logger    log.Logger
	trace     *tempopb.Trace
	spansByID map[uint64][]*v1.Span
	maxUsedID uint64
}

// Do implements Handler
func (s spanIDDeduper) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	span, _ := opentracing.StartSpanFromContext(ctx, "frontend.DedupeSpanIDs")
	defer span.Finish()

	resp, err := s.next.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		traceObject := &tempopb.Trace{}
		err = proto.Unmarshal(body, traceObject)
		if err != nil {
			return nil, err
		}

		s.trace = traceObject
		s.dedupe()

		traceBytes, err := proto.Marshal(s.trace)
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(traceBytes)),
			Header:     http.Header{},
		}, nil
	}

	return &http.Response{
		StatusCode: resp.StatusCode,
		Body:       ioutil.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{},
	}, nil
}

func (s *spanIDDeduper) dedupe() {
	s.groupSpansByID()
	s.dedupeSpanIDs()
}

// groupSpansByID groups spans with the same ID returning a map id -> []Span
func (d *spanIDDeduper) groupSpansByID() {
	spansByID := make(map[uint64][]*v1.Span)
	for _, batch := range d.trace.Batches {
		for _, ils := range batch.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				id := binary.BigEndian.Uint64(span.SpanId)
				if spans, ok := spansByID[id]; ok {
					// TODO maybe return an error if more than 2 spans found
					spansByID[id] = append(spans, span)
				} else {
					spansByID[id] = []*v1.Span{span}
				}
			}
		}
	}
	d.spansByID = spansByID
}

func (d *spanIDDeduper) isSharedWithClientSpan(spanID uint64) bool {
	for _, span := range d.spansByID[spanID] {
		if span.GetKind() == v1.Span_SPAN_KIND_CLIENT {
			return true
		}
	}
	return false
}

func (d *spanIDDeduper) dedupeSpanIDs() {
	oldToNewSpanIDs := make(map[uint64]uint64)
	for _, batch := range d.trace.Batches {
		for _, ils := range batch.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				id := binary.BigEndian.Uint64(span.SpanId)
				// only replace span IDs for server-side spans that share the ID with something else
				if span.GetKind() == v1.Span_SPAN_KIND_SERVER && d.isSharedWithClientSpan(id) {
					newID, err := d.makeUniqueSpanID()
					if err != nil {
						// ignore this error condition where we have more than 2^64 unique span IDs
						continue
					}
					oldToNewSpanIDs[id] = newID
					if len(span.ParentSpanId) == 0 {
						span.ParentSpanId = make([]byte, 8)
					}
					binary.BigEndian.PutUint64(span.ParentSpanId, id) // previously shared ID is the new parent
					binary.BigEndian.PutUint64(span.SpanId, newID)
				}
			}
		}
	}
	d.swapParentIDs(oldToNewSpanIDs)
}

// swapParentIDs corrects ParentSpanID of all spans that are children of the server
// spans whose IDs we deduped.
func (d *spanIDDeduper) swapParentIDs(oldToNewSpanIDs map[uint64]uint64) {
	for _, batch := range d.trace.Batches {
		for _, ils := range batch.InstrumentationLibrarySpans {
			for _, span := range ils.Spans {
				if len(span.GetParentSpanId()) > 0 {
					parentSpanId := binary.BigEndian.Uint64(span.GetParentSpanId())
					if parentID, ok := oldToNewSpanIDs[parentSpanId]; ok {
						if binary.BigEndian.Uint64(span.SpanId) != parentID {
							binary.BigEndian.PutUint64(span.SpanId, parentID)
						}
					}
				}
			}
		}
	}
}

// makeUniqueSpanID returns a new ID that is not used in the trace,
// or an error if such ID cannot be generated, which is unlikely,
// given that the whole space of span IDs is 2^64.
func (d *spanIDDeduper) makeUniqueSpanID() (uint64, error) {
	for id := d.maxUsedID + 1; id < maxSpanID; id++ {
		if _, ok := d.spansByID[id]; !ok {
			d.maxUsedID = id
			return id, nil
		}
	}
	return 0, fmt.Errorf(warningTooManySpans)
}
