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

// Helper function to create an iterator, that abstracts away
// context like file and rowgroups.
type makeIterFunc func(columnName string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator

// Helper function to create a column predicate for the given conditions
type makePredicateForCondition func(traceql.Operation, []interface{}) (parquetquery.Predicate, error)

const (
	columnPathTraceID             = "TraceID"
	columnPathResourceAttrKey     = "rs.Resource.Attrs.Key"
	columnPathResourceAttrString  = "rs.Resource.Attrs.Value"
	columnPathResourceServiceName = "rs.Resource.ServiceName"
	columnPathSpanID              = "rs.ils.Spans.ID"
	columnPathSpanName            = "rs.ils.Spans.Name"
	columnPathSpanStartTime       = "rs.ils.Spans.StartUnixNanos"
	columnPathSpanEndTime         = "rs.ils.Spans.EndUnixNanos"
	columnPathSpanAttrKey         = "rs.ils.Spans.Attrs.Key"
	columnPathSpanAttrString      = "rs.ils.Spans.Attrs.Value"
	columnPathSpanHttpStatusCode  = "rs.ils.Spans.HttpStatusCode"
)

type columnLevel int

const (
	columnLevelResource = iota
	columnLevelSpan
)

// Lookup table of all well-known intrinsics and dedicated columns
// TODO - Add parameter type checking here somehow, i.e. assert
//		  that if searching "name" then the operands are strings
var wellKnownColumnLookups = map[string]struct {
	columnPath  string      // path.to.column
	level       columnLevel // span or resource level
	predicateFn makePredicateForCondition
}{
	// Resource-level intrinsics and columns
	LabelServiceName: {columnPathResourceServiceName, columnLevelResource, createStringPredicate},

	// Span-level intrinsics and columns
	LabelName:           {columnPathSpanName, columnLevelSpan, createStringPredicate},
	LabelHTTPStatusCode: {columnPathSpanHttpStatusCode, columnLevelSpan, createIntPredicate},
}

// Fetch spansets from the block for the given traceql request.
func (b *backendBlock) Fetch(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {

	rr := NewBackendReaderAt(ctx, b.r, DataFileName, b.meta.BlockID, b.meta.TenantID)

	// 32 MB memory buffering
	br := tempo_io.NewBufferedReaderAt(rr, int64(b.meta.Size), 512*1024, 64)

	pf, err := parquet.OpenFile(br, int64(b.meta.Size))
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "opening parquet file")
	}

	iter, err := fetch(ctx, req.Conditions, pf)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "creating fetch iter")
	}

	return traceql.FetchSpansResponse{
		Results: iter,
	}, nil
}

// spansetIterator turns the parquet iterator into the final
// traceql iterator.  Every row it receives is one spanset.
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

func fetch(ctx context.Context, conditions []traceql.Condition, pf *parquet.File) (*spansetIterator, error) {

	// Categorize conditions into span-level or resource-level
	var (
		spanConditions     []traceql.Condition
		resourceConditions []traceql.Condition
	)
	for _, cond := range conditions {

		// Well-known column or intrinsic?
		if entry, ok := wellKnownColumnLookups[cond.Selector]; ok {
			switch entry.level {
			case columnLevelSpan:
				spanConditions = append(spanConditions, cond)
			case columnLevelResource:
				resourceConditions = append(resourceConditions, cond)
			}
			continue
		}

		// Attribute selector?
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

			continue
		}

		return nil, fmt.Errorf("unknown traceql selector: %s", cond.Selector)
	}

	// For now we iterate all row groups in the file
	// TODO: Add sharding params to the traceql request?
	makeIter := func(name string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator {
		index, _ := parquetquery.GetColumnIndexByPath(pf, name)
		if index == -1 {
			// TODO - don't panic, error instead
			panic("column not found in parquet file:" + name)
		}
		return parquetquery.NewColumnIterator(ctx, pf.RowGroups(), index, name, 1000, predicate, selectAs)
	}

	spanIter, err := createSpanIterator(makeIter, spanConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span iterator")
	}

	resourceIter, err := createResourceIterator(makeIter, spanIter, resourceConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating resource iterator")
	}

	traceIter, err := createTraceIterator(makeIter, resourceIter)
	if err != nil {
		return nil, errors.Wrap(err, "creating trace iterator")
	}

	return &spansetIterator{traceIter}, nil
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createSpanIterator(makeIter makeIterFunc, conditions []traceql.Condition) (parquetquery.Iterator, error) {

	var iters []parquetquery.Iterator
	attrKeys := []string{}
	attrStringPreds := []parquetquery.Predicate{}

	for _, cond := range conditions {

		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[cond.Selector]; ok {
			pred, err := entry.predicateFn(cond.Operation, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating predicate")
			}
			iters = append(iters, makeIter(entry.columnPath, pred, cond.Selector))
			continue
		}

		// Arbitrary attribute lookup
		s := cond.Selector
		s = strings.TrimPrefix(s, ".span")
		s = strings.TrimPrefix(s, ".")
		attrKeys = append(attrKeys, s)

		pred, err := createStringPredicate(cond.Operation, cond.Operands)
		if err != nil {
			return nil, errors.Wrap(err, "creating attribute span predicate")
		}
		if pred != nil {
			attrStringPreds = append(attrStringPreds, pred)
		}
	}
	var attrIters []parquetquery.Iterator
	if len(attrKeys) > 0 {
		attrIters = append(attrIters, makeIter(columnPathSpanAttrKey, parquetquery.NewStringInPredicate(attrKeys), "key"))
	}
	if len(attrStringPreds) > 0 {
		attrIters = append(attrIters, makeIter(columnPathSpanAttrString, parquetquery.NewOrPredicate(attrStringPreds...), "strings"))
	}
	if len(attrIters) > 0 {
		// Join iterator here keeps key/value pairs together
		iters = append(iters, parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, attrIters, nil))
	}

	// Static columns that are always loaded
	iters = append(iters, makeIter(columnPathSpanID, nil, columnPathSpanID))
	iters = append(iters, makeIter(columnPathSpanStartTime, nil, columnPathSpanStartTime))
	iters = append(iters, makeIter(columnPathSpanEndTime, nil, columnPathSpanEndTime))

	spanIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpansILSSpan, iters, &spanCollector{})

	return spanIter, nil
}

// createResourceIter iterates through all resourcespans-level (batch-level) columns, groups them into rows representing
// one batch each. It builds on top of the span iterator, and turns the groups of spans and resource-level values into
// spansets.  Spansets are returned that match any of the given conditions.

func createResourceIterator(makeIter makeIterFunc, spanIterator parquetquery.Iterator, conditions []traceql.Condition) (parquetquery.Iterator, error) {
	iters := []parquetquery.Iterator{
		spanIterator,
	}
	attrKeys := []string{}
	attrStringPreds := []parquetquery.Predicate{}
	for _, cond := range conditions {

		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[cond.Selector]; ok {
			pred, err := entry.predicateFn(cond.Operation, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating predicate")
			}
			iters = append(iters, makeIter(entry.columnPath, pred, cond.Selector))
			continue
		}

		s := cond.Selector
		s = strings.TrimPrefix(s, ".resource")
		s = strings.TrimPrefix(s, ".")
		attrKeys = append(attrKeys, s)

		pred, err := createStringPredicate(cond.Operation, cond.Operands)
		if err != nil {
			return nil, errors.Wrap(err, "creating attribute resource predicate")
		}
		if pred != nil {
			attrStringPreds = append(attrStringPreds, pred)
		}
	}
	var attrIters []parquetquery.Iterator
	if len(attrKeys) > 0 {
		attrIters = append(attrIters, makeIter(columnPathResourceAttrKey, parquetquery.NewStringInPredicate(attrKeys), "key"))
	}
	if len(attrStringPreds) > 0 {
		attrIters = append(attrIters, makeIter(columnPathResourceAttrString, parquetquery.NewOrPredicate(attrStringPreds...), "strings"))
	}
	if len(attrIters) > 0 {
		iters = append(iters, parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, attrIters, nil))
	}

	// Final resource iterator.
	// Union to return resources that match any of the conditions.
	// BatchCollector to group the spans into spanset
	resourceIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpans, iters, &batchCollector{})

	return resourceIter, nil
}

func createTraceIterator(makeIter makeIterFunc, resourceIter parquetquery.Iterator) (parquetquery.Iterator, error) {
	traceIters := []parquetquery.Iterator{
		resourceIter,
		// Add static columns that are always return
		makeIter(columnPathTraceID, nil, columnPathTraceID),
	}

	// Final trace iterator
	// Join iterator means it requires matching resources to have been found
	// TraceCollor adds trace-level data to the spansets
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

	// Merge all individual columns into the span
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathSpanID:
			span.ID = kv.Value.ByteArray()
		case columnPathSpanStartTime:
			span.StartTimeUnixNanos = kv.Value.Uint64()
		case columnPathSpanEndTime:
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

// batchCollector receives rows of matching resource-level
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

// traceCollector receives rows from the resource-level matches.
// It adds trace-level attributes into the spansets before
// they are returned
type traceCollector struct {
}

var _ parquetquery.GroupPredicate = (*traceCollector)(nil)

func (c *traceCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	spanset := res.OtherEntries[0].Value.(*traceql.Spanset)

	for k, v := range res.ToMap() {
		switch k {
		case columnPathTraceID:
			spanset.TraceID = v[0].ByteArray()
		}
	}

	return true
}
