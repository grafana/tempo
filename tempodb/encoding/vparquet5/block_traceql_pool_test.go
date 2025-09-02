package vparquet5

import (
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo/pkg/parquetquery"
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
	span1.cbSpansetFinal = true
	span1.cbSpanset = getSpanset()
	span1.spanAttrs = append(span1.spanAttrs, attrVal{
		a: traceql.NewAttribute("span-key"),
		s: traceql.NewStaticString("span-value"),
	})
	span1.resourceAttrs = append(span1.resourceAttrs, attrVal{
		a: traceql.NewAttribute("resource-key"),
		s: traceql.NewStaticString("resource-value"),
	})
	span1.traceAttrs = append(span1.traceAttrs, attrVal{
		a: traceql.NewAttribute("trace-key"),
		s: traceql.NewStaticString("trace-value"),
	})
	span1.eventAttrs = append(span1.eventAttrs, attrVal{
		a: traceql.NewAttribute("event-key"),
		s: traceql.NewStaticString("event-value"),
	})
	span1.linkAttrs = append(span1.linkAttrs, attrVal{
		a: traceql.NewAttribute("link-key"),
		s: traceql.NewStaticString("link-value"),
	})
	span1.instrumentationAttrs = append(span1.instrumentationAttrs, attrVal{
		a: traceql.NewAttribute("instrumentation-key"),
		s: traceql.NewStaticString("instrumentation-value"),
	})
	span1.rowNum = parquetquery.RowNumber{1, 100}

	putSpan(span1)

	span2 := getSpan()

	// Verify the reused span is properly cleared
	assert.Nil(t, span2.id, "id should be cleared")
	assert.Zero(t, span2.startTimeUnixNanos, "startTimeUnixNanos should be cleared")
	assert.Zero(t, span2.durationNanos, "durationNanos should be cleared")
	assert.Zero(t, span2.nestedSetParent, "nestedSetParent should be cleared")
	assert.Zero(t, span2.nestedSetLeft, "nestedSetLeft should be cleared")
	assert.Zero(t, span2.nestedSetRight, "nestedSetRight should be cleared")
	assert.False(t, span2.cbSpansetFinal, "cbSpansetFinal should be cleared")
	assert.Nil(t, span2.cbSpanset, "cbSpanset should be cleared")
	assert.Len(t, span2.spanAttrs, 0, "spanAttrs should be cleared")
	assert.Len(t, span2.resourceAttrs, 0, "resourceAttrs should be cleared")
	assert.Len(t, span2.traceAttrs, 0, "traceAttrs should be cleared")
	assert.Len(t, span2.eventAttrs, 0, "eventAttrs should be cleared")
	assert.Len(t, span2.linkAttrs, 0, "linkAttrs should be cleared")
	assert.Len(t, span2.instrumentationAttrs, 0, "instrumentationAttrs should be cleared")
	assert.Equal(t, parquetquery.EmptyRowNumber(), span2.rowNum, "rowNum should be cleared")
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
	if spanset1.ServiceStats == nil {
		spanset1.ServiceStats = make(map[string]traceql.ServiceStats)
	}
	spanset1.ServiceStats["service1"] = traceql.ServiceStats{SpanCount: 10, ErrorCount: 2}
	spanset1.ServiceStats["service2"] = traceql.ServiceStats{SpanCount: 5, ErrorCount: 1}

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
	assert.Len(t, spanset2.ServiceStats, 0, "ServiceStats should be cleared")
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
