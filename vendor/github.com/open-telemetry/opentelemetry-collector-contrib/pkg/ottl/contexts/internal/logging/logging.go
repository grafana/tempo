// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logging // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"

import (
	"encoding/hex"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap/zapcore"
)

type Slice pcommon.Slice

func (s Slice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	ss := pcommon.Slice(s)
	var err error
	for i := 0; i < ss.Len(); i++ {
		v := ss.At(i)
		switch v.Type() {
		case pcommon.ValueTypeStr:
			encoder.AppendString(v.Str())
		case pcommon.ValueTypeBool:
			encoder.AppendBool(v.Bool())
		case pcommon.ValueTypeInt:
			encoder.AppendInt64(v.Int())
		case pcommon.ValueTypeDouble:
			encoder.AppendFloat64(v.Double())
		case pcommon.ValueTypeMap:
			err = errors.Join(err, encoder.AppendObject(Map(v.Map())))
		case pcommon.ValueTypeSlice:
			err = errors.Join(err, encoder.AppendArray(Slice(v.Slice())))
		case pcommon.ValueTypeBytes:
			encoder.AppendByteString(v.Bytes().AsRaw())
		}
	}
	return err
}

type Map pcommon.Map

func (m Map) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	mm := pcommon.Map(m)
	var err error
	for k, v := range mm.All() {
		switch v.Type() {
		case pcommon.ValueTypeStr:
			encoder.AddString(k, v.Str())
		case pcommon.ValueTypeBool:
			encoder.AddBool(k, v.Bool())
		case pcommon.ValueTypeInt:
			encoder.AddInt64(k, v.Int())
		case pcommon.ValueTypeDouble:
			encoder.AddFloat64(k, v.Double())
		case pcommon.ValueTypeMap:
			err = errors.Join(err, encoder.AddObject(k, Map(v.Map())))
		case pcommon.ValueTypeSlice:
			err = errors.Join(err, encoder.AddArray(k, Slice(v.Slice())))
		case pcommon.ValueTypeBytes:
			encoder.AddByteString(k, v.Bytes().AsRaw())
		}
	}
	return nil
}

type Resource pcommon.Resource

func (r Resource) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	rr := pcommon.Resource(r)
	err := encoder.AddObject("attributes", Map(rr.Attributes()))
	encoder.AddUint32("dropped_attribute_count", rr.DroppedAttributesCount())
	return err
}

type InstrumentationScope pcommon.InstrumentationScope

func (i InstrumentationScope) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	is := pcommon.InstrumentationScope(i)
	err := encoder.AddObject("attributes", Map(is.Attributes()))
	encoder.AddUint32("dropped_attribute_count", is.DroppedAttributesCount())
	encoder.AddString("name", is.Name())
	encoder.AddString("version", is.Version())
	return err
}

type Span ptrace.Span

func (s Span) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	ss := ptrace.Span(s)
	parentSpanID := ss.ParentSpanID()
	spanID := ss.SpanID()
	traceID := ss.TraceID()
	err := encoder.AddObject("attributes", Map(ss.Attributes()))
	encoder.AddUint32("dropped_attribute_count", ss.DroppedAttributesCount())
	encoder.AddUint32("dropped_events_count", ss.DroppedEventsCount())
	encoder.AddUint32("dropped_links_count", ss.DroppedLinksCount())
	encoder.AddUint64("end_time_unix_nano", uint64(ss.EndTimestamp()))
	err = errors.Join(err, encoder.AddArray("events", SpanEventSlice(ss.Events())))
	encoder.AddString("kind", ss.Kind().String())
	err = errors.Join(err, encoder.AddArray("links", SpanLinkSlice(ss.Links())))
	encoder.AddString("name", ss.Name())
	encoder.AddString("parent_span_id", hex.EncodeToString(parentSpanID[:]))
	encoder.AddString("span_id", hex.EncodeToString(spanID[:]))
	encoder.AddUint64("start_time_unix_nano", uint64(ss.StartTimestamp()))
	encoder.AddString("status.code", ss.Status().Code().String())
	encoder.AddString("status.message", ss.Status().Message())
	encoder.AddString("trace_id", hex.EncodeToString(traceID[:]))
	encoder.AddString("trace_state", ss.TraceState().AsRaw())
	return err
}

type SpanEventSlice ptrace.SpanEventSlice

func (s SpanEventSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	ses := ptrace.SpanEventSlice(s)
	var err error
	for i := 0; i < ses.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(SpanEvent(ses.At(i))))
	}
	return err
}

type SpanEvent ptrace.SpanEvent

func (s SpanEvent) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	se := ptrace.SpanEvent(s)
	err := encoder.AddObject("attributes", Map(se.Attributes()))
	encoder.AddUint32("dropped_attribute_count", se.DroppedAttributesCount())
	encoder.AddString("name", se.Name())
	encoder.AddUint64("time_unix_nano", uint64(se.Timestamp()))
	return err
}

type SpanLinkSlice ptrace.SpanLinkSlice

func (s SpanLinkSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	sls := ptrace.SpanLinkSlice(s)
	var err error
	for i := 0; i < sls.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(SpanLink(sls.At(i))))
	}
	return err
}

type SpanLink ptrace.SpanLink

func (s SpanLink) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	sl := ptrace.SpanLink(s)
	spanID := sl.SpanID()
	traceID := sl.TraceID()
	err := encoder.AddObject("attributes", Map(sl.Attributes()))
	encoder.AddUint32("dropped_attribute_count", sl.DroppedAttributesCount())
	encoder.AddUint32("flags", sl.Flags())
	encoder.AddString("span_id", hex.EncodeToString(spanID[:]))
	encoder.AddString("trace_id", hex.EncodeToString(traceID[:]))
	encoder.AddString("trace_state", sl.TraceState().AsRaw())
	return err
}

type Metric pmetric.Metric

func (m Metric) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	mm := pmetric.Metric(m)
	encoder.AddString("description", mm.Description())
	encoder.AddString("name", mm.Name())
	encoder.AddString("unit", mm.Unit())
	encoder.AddString("type", mm.Type().String())

	var err error
	switch mm.Type() {
	case pmetric.MetricTypeSum:
		encoder.AddString("aggregation_temporality", mm.Sum().AggregationTemporality().String())
		encoder.AddBool("is_monotonic", mm.Sum().IsMonotonic())
		err = encoder.AddArray("datapoints", NumberDataPointSlice(mm.Sum().DataPoints()))
	case pmetric.MetricTypeGauge:
		err = encoder.AddArray("datapoints", NumberDataPointSlice(mm.Gauge().DataPoints()))
	case pmetric.MetricTypeHistogram:
		encoder.AddString("aggregation_temporality", mm.Histogram().AggregationTemporality().String())
		err = encoder.AddArray("datapoints", HistogramDataPointSlice(mm.Histogram().DataPoints()))
	case pmetric.MetricTypeExponentialHistogram:
		encoder.AddString("aggregation_temporality", mm.ExponentialHistogram().AggregationTemporality().String())
		err = encoder.AddArray("datapoints", ExponentialHistogramDataPointSlice(mm.ExponentialHistogram().DataPoints()))
	case pmetric.MetricTypeSummary:
		err = encoder.AddArray("datapoints", SummaryDataPointSlice(mm.Summary().DataPoints()))
	}

	return err
}

type NumberDataPointSlice pmetric.NumberDataPointSlice

func (n NumberDataPointSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	ndps := pmetric.NumberDataPointSlice(n)
	var err error
	for i := 0; i < ndps.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(NumberDataPoint(ndps.At(i))))
	}
	return err
}

type NumberDataPoint pmetric.NumberDataPoint

func (n NumberDataPoint) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	ndp := pmetric.NumberDataPoint(n)
	err := encoder.AddObject("attributes", Map(ndp.Attributes()))
	err = errors.Join(err, encoder.AddArray("exemplars", ExemplarSlice(ndp.Exemplars())))
	encoder.AddUint32("flags", uint32(ndp.Flags()))
	encoder.AddUint64("start_time_unix_nano", uint64(ndp.StartTimestamp()))
	encoder.AddUint64("time_unix_nano", uint64(ndp.Timestamp()))
	if ndp.ValueType() == pmetric.NumberDataPointValueTypeInt {
		encoder.AddInt64("value_int", ndp.IntValue())
	}
	if ndp.ValueType() == pmetric.NumberDataPointValueTypeDouble {
		encoder.AddFloat64("value_double", ndp.DoubleValue())
	}

	return err
}

type HistogramDataPointSlice pmetric.HistogramDataPointSlice

func (h HistogramDataPointSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	hdps := pmetric.HistogramDataPointSlice(h)
	var err error
	for i := 0; i < hdps.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(HistogramDataPoint(hdps.At(i))))
	}
	return err
}

type HistogramDataPoint pmetric.HistogramDataPoint

func (h HistogramDataPoint) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	hdp := pmetric.HistogramDataPoint(h)
	err := encoder.AddObject("attributes", Map(hdp.Attributes()))
	err = errors.Join(err, encoder.AddArray("bucket_counts", UInt64Slice(hdp.BucketCounts())))
	encoder.AddUint64("count", hdp.Count())
	err = errors.Join(err, encoder.AddArray("exemplars", ExemplarSlice(hdp.Exemplars())))
	err = errors.Join(err, encoder.AddArray("explicit_bounds", Float64Slice(hdp.ExplicitBounds())))
	encoder.AddUint32("flags", uint32(hdp.Flags()))
	encoder.AddFloat64("max", hdp.Max())
	encoder.AddFloat64("min", hdp.Min())
	encoder.AddUint64("start_time_unix_nano", uint64(hdp.StartTimestamp()))
	encoder.AddFloat64("sum", hdp.Sum())
	encoder.AddUint64("time_unix_nano", uint64(hdp.Timestamp()))

	return err
}

type ExponentialHistogramDataPointSlice pmetric.ExponentialHistogramDataPointSlice

func (e ExponentialHistogramDataPointSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	ehdps := pmetric.ExponentialHistogramDataPointSlice(e)
	var err error
	for i := 0; i < ehdps.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(ExponentialHistogramDataPoint(ehdps.At(i))))
	}
	return err
}

type ExponentialHistogramDataPoint pmetric.ExponentialHistogramDataPoint

func (e ExponentialHistogramDataPoint) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	ehdp := pmetric.ExponentialHistogramDataPoint(e)
	err := encoder.AddObject("attributes", Map(ehdp.Attributes()))
	encoder.AddUint64("count", ehdp.Count())
	err = errors.Join(err, encoder.AddArray("exemplars", ExemplarSlice(ehdp.Exemplars())))
	encoder.AddUint32("flags", uint32(ehdp.Flags()))
	encoder.AddFloat64("max", ehdp.Max())
	encoder.AddFloat64("min", ehdp.Min())
	err = errors.Join(err, encoder.AddObject("negative", ExponentialHistogramDataPointBuckets(ehdp.Negative())))
	err = errors.Join(err, encoder.AddObject("positive", ExponentialHistogramDataPointBuckets(ehdp.Positive())))
	encoder.AddInt32("scale", ehdp.Scale())
	encoder.AddUint64("start_time_unix_nano", uint64(ehdp.StartTimestamp()))
	encoder.AddFloat64("sum", ehdp.Sum())
	encoder.AddUint64("time_unix_nano", uint64(ehdp.Timestamp()))
	encoder.AddUint64("zero_count", ehdp.ZeroCount())
	encoder.AddFloat64("zero_threshold", ehdp.ZeroThreshold())
	return err
}

type ExponentialHistogramDataPointBuckets pmetric.ExponentialHistogramDataPointBuckets

func (e ExponentialHistogramDataPointBuckets) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	b := pmetric.ExponentialHistogramDataPointBuckets(e)
	err := encoder.AddArray("bucket_counts", UInt64Slice(b.BucketCounts()))
	encoder.AddInt32("offset", b.Offset())
	return err
}

type SummaryDataPointSlice pmetric.SummaryDataPointSlice

func (s SummaryDataPointSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	sdps := pmetric.SummaryDataPointSlice(s)
	var err error
	for i := 0; i < sdps.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(SummaryDataPoint(sdps.At(i))))
	}
	return err
}

type SummaryDataPoint pmetric.SummaryDataPoint

func (s SummaryDataPoint) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	sdp := pmetric.SummaryDataPoint(s)
	err := encoder.AddObject("attributes", Map(sdp.Attributes()))
	encoder.AddUint64("count", sdp.Count())
	encoder.AddUint32("flags", uint32(sdp.Flags()))
	encoder.AddUint64("start_time_unix_nano", uint64(sdp.StartTimestamp()))
	encoder.AddFloat64("sum", sdp.Sum())
	encoder.AddUint64("time_unix_nano", uint64(sdp.Timestamp()))
	err = errors.Join(err, encoder.AddArray("quantile_values", SummaryDataPointValueAtQuantileSlice(sdp.QuantileValues())))

	return err
}

type SummaryDataPointValueAtQuantileSlice pmetric.SummaryDataPointValueAtQuantileSlice

func (s SummaryDataPointValueAtQuantileSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	qs := pmetric.SummaryDataPointValueAtQuantileSlice(s)
	var err error
	for i := 0; i < qs.Len(); i++ {
		err = errors.Join(err, encoder.AppendObject(SummaryDataPointValueAtQuantile(qs.At(i))))
	}
	return nil
}

type SummaryDataPointValueAtQuantile pmetric.SummaryDataPointValueAtQuantile

func (s SummaryDataPointValueAtQuantile) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	q := pmetric.SummaryDataPointValueAtQuantile(s)
	encoder.AddFloat64("value", q.Value())
	encoder.AddFloat64("quantile", q.Quantile())
	return nil
}

type UInt64Slice pcommon.UInt64Slice

func (u UInt64Slice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	uis := pcommon.UInt64Slice(u)
	for i := 0; i < uis.Len(); i++ {
		encoder.AppendUint64(uis.At(i))
	}
	return nil
}

type Float64Slice pcommon.Float64Slice

func (f Float64Slice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	fs := pcommon.Float64Slice(f)
	for i := 0; i < fs.Len(); i++ {
		encoder.AppendFloat64(fs.At(i))
	}
	return nil
}

type ExemplarSlice pmetric.ExemplarSlice

func (e ExemplarSlice) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	es := pmetric.ExemplarSlice(e)
	var err error
	for i := 0; i < es.Len(); i++ {
		ee := es.At(i)
		err = errors.Join(err, encoder.AppendObject(Exemplar(ee)))
	}
	return err
}

type Exemplar pmetric.Exemplar

func (e Exemplar) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	ee := pmetric.Exemplar(e)
	spanID := ee.SpanID()
	traceID := ee.TraceID()
	err := encoder.AddObject("filtered_attributes", Map(ee.FilteredAttributes()))
	encoder.AddString("span_id", hex.EncodeToString(spanID[:]))
	encoder.AddUint64("time_unix_nano", uint64(ee.Timestamp()))
	encoder.AddString("trace_id", hex.EncodeToString(traceID[:]))
	if ee.ValueType() == pmetric.ExemplarValueTypeInt {
		encoder.AddInt64("value_int", ee.IntValue())
	}
	if ee.ValueType() == pmetric.ExemplarValueTypeDouble {
		encoder.AddFloat64("value_double", ee.DoubleValue())
	}
	return err
}
