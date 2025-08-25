package vparquet4

import (
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/traceql"
)

func TestSpanPoolRelease(t *testing.T) {
	// Get a span from the pool and populate it
	span1 := getSpan()
	span1.id = []byte("test-id")
	span1.startTimeUnixNanos = 12345
	span1.durationNanos = 67890
	span1.nestedSetParent = 1
	span1.nestedSetLeft = 2
	span1.nestedSetRight = 3

	putSpan(span1)

	span2 := getSpan()

	// Verify the reused span is properly cleared
	assert.Nil(t, span2.id, "id should be cleared")
	assert.Zero(t, span2.startTimeUnixNanos, "startTimeUnixNanos should be cleared")
	assert.Zero(t, span2.durationNanos, "durationNanos should be cleared")
	assert.Zero(t, span2.nestedSetParent, "nestedSetParent should be cleared")
	assert.Zero(t, span2.nestedSetLeft, "nestedSetLeft should be cleared")
	assert.Zero(t, span2.nestedSetRight, "nestedSetRight should be cleared")
}

func TestSpansetPoolRelease(t *testing.T) {
	// Get a spanset from the pool and populate it
	spanset1 := getSpanset()
	spanset1.TraceID = []byte("trace-id")
	spanset1.RootSpanName = "root"
	spanset1.RootServiceName = "service"
	spanset1.StartTimeUnixNanos = 12345
	spanset1.DurationNanos = 67890
	spanset1.Scalar = traceql.NewStaticString("scalar")
	spanset1.Attributes = append(spanset1.Attributes, &traceql.SpansetAttribute{Name: "test", Val: traceql.NewStaticString("value")})
	spanset1.Spans = append(spanset1.Spans, getSpan())

	putSpanset(spanset1)

	spanset2 := getSpanset()

	// Verify the reused spanset is properly cleared
	assert.Nil(t, spanset2.TraceID, "TraceID should be cleared")
	assert.Empty(t, spanset2.RootSpanName, "RootSpanName should be cleared")
	assert.Empty(t, spanset2.RootServiceName, "RootServiceName should be cleared")
	assert.Zero(t, spanset2.StartTimeUnixNanos, "StartTimeUnixNanos should be cleared")
	assert.Zero(t, spanset2.DurationNanos, "DurationNanos should be cleared")
	assert.Equal(t, traceql.TypeNil, spanset2.Scalar.Type, "Scalar should be cleared")
	assert.Len(t, spanset2.Attributes, 0, "Attributes should be cleared")
	assert.Len(t, spanset2.Spans, 0, "Spans should be cleared")
}

func TestEventPoolRelease(t *testing.T) {
	// Get an event from the pool and populate it
	event1 := getEvent()
	event1.attrs = append(event1.attrs, attrVal{
		a: traceql.NewAttribute("key1"),
		s: traceql.NewStaticString("value1"),
	})
	event1.attrs = append(event1.attrs, attrVal{
		a: traceql.NewAttribute("key2"),
		s: traceql.NewStaticString("value2"),
	})

	putEvent(event1)

	event2 := getEvent()

	// Verify the reused event is properly cleared
	assert.Len(t, event2.attrs, 0, "attrs should be cleared")
}

func TestLinkPoolRelease(t *testing.T) {
	// Get a link from the pool and populate it
	link1 := getLink()
	link1.attrs = append(link1.attrs, attrVal{
		a: traceql.NewAttribute("key1"),
		s: traceql.NewStaticString("value1"),
	})
	link1.attrs = append(link1.attrs, attrVal{
		a: traceql.NewAttribute("key2"),
		s: traceql.NewStaticString("value2"),
	})

	putLink(link1)

	link2 := getLink()

	// Verify the reused link is properly cleared
	assert.Len(t, link2.attrs, 0, "attrs should be cleared")
}

func TestRowPoolRelease(t *testing.T) {
	pool := newRowPool(5)

	// Get a row from the pool and populate it
	row1 := pool.Get()
	row1 = append(row1, parquet.ValueOf("value1"))
	row1 = append(row1, parquet.ValueOf(42))
	row1 = append(row1, parquet.ValueOf(true))

	pool.Put(row1)

	row2 := pool.Get()

	// Verify the reused row is properly cleared
	assert.Len(t, row2, 0, "row should be empty after release")
	assert.Equal(t, 5, cap(row2), "capacity should be preserved")
}
