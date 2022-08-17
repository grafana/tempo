package vparquet

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
)

func (b *backendBlock) Fetch(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {

	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 32 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 512*1024, 64)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "opening parquet file")
	}

	iter, err := createIterator(ctx, req.Conditions, pf)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "creating condition iter")
	}

	return traceql.FetchSpansResponse{
		Results: &spansetIterator{
			iter: iter,
		},
	}, nil
}

func createIterator(ctx context.Context, conditions []traceql.Condition, pf *parquet.File) (parquetquery.Iterator, error) {

	makeIter := func(name string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator {
		index, _ := parquetquery.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}
		return parquetquery.NewColumnIterator(ctx, pf.RowGroups(), index, name, 1000, predicate, selectAs)
	}

	var spanConditions []traceql.Condition
	var resourceConditions []traceql.Condition

	// Break conditions up into span-level or resource-level
	for _, cond := range conditions {
		switch cond.Selector {
		case LabelName:
			spanConditions = append(spanConditions, cond)
		case LabelServiceName:
			resourceConditions = append(resourceConditions, cond)
		case LabelHTTPStatusCode:
			spanConditions = append(spanConditions, cond)
		default:
			if strings.HasPrefix(cond.Selector, ".") {
				isSpan := true
				isRes := true
				if strings.HasPrefix(cond.Selector, ".span.") {
					isRes = false
				} else if strings.HasPrefix(cond.Selector, ".resource") {
					isSpan = false
				}

				if isSpan {
					spanConditions = append(spanConditions, cond)
				}
				if isRes {
					resourceConditions = append(resourceConditions, cond)
				}
			}
		}
	}

	var spanItrs []parquetquery.Iterator
	spanAttrKeys := []string{}
	spanAttrStringPreds := []parquetquery.Predicate{}
	for _, cond := range spanConditions {

		switch cond.Selector {
		case LabelName:
			var names []string
			for _, operand := range cond.Operands {
				names = append(names, operand.(string))
			}
			spanItrs = append(spanItrs, makeIter("rs.ils.Spans.Name", parquetquery.NewStringInPredicate(names), cond.Selector))
		case LabelHTTPStatusCode:
			pred, err := createIntPredicate(cond.Operation, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating http status code predicate")
			}
			spanItrs = append(spanItrs, makeIter("rs.ils.Spans.HttpStatusCode", pred, cond.Selector))
		default:
			s := cond.Selector
			s = strings.TrimPrefix(s, ".span")
			s = strings.TrimPrefix(s, ".")
			spanAttrKeys = append(spanAttrKeys, s)

			pred, err := createStringPredicate(cond.Operation, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute span predicate")
			}
			if pred != nil {
				spanAttrStringPreds = append(spanAttrStringPreds, pred)
			}
		}
	}
	var spanAttrIters []parquetquery.Iterator
	if len(spanAttrKeys) > 0 {
		spanAttrIters = append(spanAttrIters, makeIter("rs.ils.Spans.Attrs.Key", parquetquery.NewStringInPredicate(spanAttrKeys), "key"))
	}
	if len(spanAttrStringPreds) > 0 {
		spanAttrIters = append(spanAttrIters, makeIter("rs.ils.Spans.Attrs.Value", parquetquery.NewOrPredicate(spanAttrStringPreds...), "strings"))
	}
	if len(spanAttrIters) > 0 {
		spanItrs = append(spanItrs, parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, spanAttrIters, nil))
	}
	spanItrs = append(spanItrs, makeIter("rs.ils.Spans.ID", nil, "ID"))
	spanItrs = append(spanItrs, makeIter("rs.ils.Spans.StartUnixNanos", nil, "StartTimeUnixNanos"))
	spanItrs = append(spanItrs, makeIter("rs.ils.Spans.EndUnixNanos", nil, "EndTimeUnixNanos"))

	spanIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpansILSSpan, spanItrs, &spanCollector{})

	resourceIters := []parquetquery.Iterator{
		spanIter,
	}
	resourceAttrKeys := []string{}
	resourceAttrStringPreds := []parquetquery.Predicate{}
	for _, cond := range resourceConditions {
		switch cond.Selector {
		case LabelServiceName:
			var names []string
			for _, operand := range cond.Operands {
				names = append(names, operand.(string))
			}
			resourceIters = append(resourceIters, makeIter("rs.Resource.ServiceName", parquetquery.NewStringInPredicate(names), cond.Selector))
		default:
			s := cond.Selector
			s = strings.TrimPrefix(s, ".resource")
			s = strings.TrimPrefix(s, ".")
			resourceAttrKeys = append(resourceAttrKeys, s)

			pred, err := createStringPredicate(cond.Operation, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute resource predicate")
			}
			if pred != nil {
				resourceAttrStringPreds = append(resourceAttrStringPreds, pred)
			}
		}
	}
	var resourceAttrIters []parquetquery.Iterator
	if len(resourceAttrKeys) > 0 {
		resourceAttrIters = append(resourceAttrIters, makeIter("rs.Resource.Attrs.Key", parquetquery.NewStringInPredicate(resourceAttrKeys), "key"))
	}
	if len(resourceAttrStringPreds) > 0 {
		resourceAttrIters = append(resourceAttrIters, makeIter("rs.Resource.Attrs.Value", parquetquery.NewOrPredicate(resourceAttrStringPreds...), "strings"))
	}
	if len(resourceAttrIters) > 0 {
		resourceIters = append(resourceIters, parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, resourceAttrIters, nil))
	}
	resourceIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpans, resourceIters, &batchCollector{})

	traceIters := []parquetquery.Iterator{
		resourceIter,
		makeIter("TraceID", nil, "TraceID"),
	}

	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &traceCollector{}), nil
}

func createStringPredicate(op traceql.Operation, operands []interface{}) (parquetquery.Predicate, error) {
	if op == traceql.OperationNone {
		return nil, nil
	}
	if op != traceql.OperationEq && op != traceql.OperationIn {
		return nil, fmt.Errorf("operand not supported for strings: %+v", op)
	}

	vals := []string{}

	for _, op := range operands {
		s, ok := op.(string)
		if !ok {
			return nil, fmt.Errorf("operand is not string: %+v", op)
		}
		vals = append(vals, s)
	}

	return parquetquery.NewStringInPredicate(vals), nil
}

func createIntPredicate(op traceql.Operation, operands []interface{}) (parquetquery.Predicate, error) {
	if op == traceql.OperationNone {
		return nil, nil
	}

	// Ensure operand is int
	var i int64
	switch v := operands[0].(type) {
	case int:
		i = int64(v)
	case int64:
		i = v
	default:
		return nil, fmt.Errorf("operand is not int: %+v", operands[0])
	}

	// Defaults
	min := int64(math.MinInt64)
	max := int64(math.MaxInt64)

	switch op {
	case traceql.OperationEq:
		min = i
		max = i
	case traceql.OperationGT:
		min = i + 1
	case traceql.OperationLT:
		max = i - 1
	}

	return parquetquery.NewIntBetweenPredicate(min, max), nil
}

// This turns groups of span values into Span objects
type spanCollector struct {
}

var _ parquetquery.GroupPredicate = (*spanCollector)(nil)

func (c *spanCollector) KeepGroup(res *parquetquery.IteratorResult) bool {

	span := &traceql.Span{
		Attributes: make(map[string]interface{}),
	}

	for _, kv := range res.Entries {
		switch kv.Key {
		case "ID":
			span.ID = kv.Value.ByteArray()
		case "StartTimeUnixNanos":
			span.StartTimeUnixNanos = kv.Value.Uint64()
		case "EndTimeUnixNanos":
			span.EndtimeUnixNanos = kv.Value.Uint64()
		default:
			switch kv.Value.Kind() {
			case parquet.Int32:
				span.Attributes[kv.Key] = kv.Value.Int32()
			case parquet.Int64:
				span.Attributes[kv.Key] = kv.Value.Int64()
			case parquet.ByteArray:
				span.Attributes[kv.Key] = kv.Value.String()
			}
		}
	}

	res.AppendOtherValue("span", span)
	res.Entries = nil

	return true
}

// This turns groups of batch values and Spans into SpanSets
type batchCollector struct {
}

var _ parquetquery.GroupPredicate = (*batchCollector)(nil)

func (c *batchCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	sp := &traceql.Spanset{}

	for _, kv := range res.OtherEntries {
		switch kv.Key {
		case "span":
			if span, ok := kv.Value.(*traceql.Span); ok {
				sp.Spans = append(sp.Spans, span)
			}
		}
	}

	// Copy resource-level values to the individual spans now
	resAttrs := res.ToMap()
	for k, v := range resAttrs {
		for _, span := range sp.Spans {
			if _, alreadyExists := span.Attributes[k]; !alreadyExists {
				switch v[0].Kind() {
				case parquet.Int64:
					span.Attributes[k] = v[0].Int64()
				case parquet.ByteArray:
					span.Attributes[k] = v[0].String()
				}
			}
		}
	}

	// Throw out batches that don't have any attributes
	hasAttrs := false
	for _, span := range sp.Spans {
		if len(span.Attributes) > 0 {
			hasAttrs = true
			break
		}
	}
	if !hasAttrs {
		return false
	}

	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("spanset", sp)
	res.Entries = nil

	return true
}

type traceCollector struct {
}

var _ parquetquery.GroupPredicate = (*traceCollector)(nil)

func (c *traceCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	spanset := res.OtherEntries[0].Value.(*traceql.Spanset)

	for k, v := range res.ToMap() {
		switch k {
		case "TraceID":
			spanset.TraceID = v[0].ByteArray()
		}
	}

	return true
}

type spansetIterator struct {
	iter parquetquery.Iterator
}

var _ traceql.SpansetIterator = (*spansetIterator)(nil)

func (i *spansetIterator) Next(ctx context.Context) (*traceql.Spanset, error) {

	res := i.iter.Next()
	if res == nil {
		return nil, nil
	}

	// The spanset is in the OtherEntries
	spanset := res.OtherEntries[0].Value.(*traceql.Spanset)

	return spanset, nil
}
