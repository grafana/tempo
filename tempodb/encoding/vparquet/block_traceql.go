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
	columnPathResourceAttrInt     = "rs.Resource.Attrs.ValueInt"
	columnPathResourceAttrDouble  = "rs.Resource.Attrs.ValueDouble"
	columnPathResourceAttrBool    = "rs.Resource.Attrs.ValueBool"
	columnPathResourceServiceName = "rs.Resource.ServiceName"
	columnPathSpanID              = "rs.ils.Spans.ID"
	columnPathSpanName            = "rs.ils.Spans.Name"
	columnPathSpanStartTime       = "rs.ils.Spans.StartUnixNanos"
	columnPathSpanEndTime         = "rs.ils.Spans.EndUnixNanos"
	columnPathSpanAttrKey         = "rs.ils.Spans.Attrs.Key"
	columnPathSpanAttrString      = "rs.ils.Spans.Attrs.Value"
	columnPathSpanAttrInt         = "rs.ils.Spans.Attrs.ValueInt"
	columnPathSpanAttrDouble      = "rs.ils.Spans.Attrs.ValueDouble"
	columnPathSpanAttrBool        = "rs.ils.Spans.Attrs.ValueBool"
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

	err := checkConditions(req.Conditions)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "conditions invalid")
	}

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

func checkConditions(conditions []traceql.Condition) error {
	for _, cond := range conditions {
		opCount := len(cond.Operands)

		switch cond.Operation {

		case traceql.OperationNone:
			if opCount != 0 {
				return fmt.Errorf("operand eq must have exactly 1 argument. condition: %+v", cond)
			}

		case traceql.OperationEq, traceql.OperationGT, traceql.OperationLT:
			if opCount != 1 {
				return fmt.Errorf("operand %v must have exactly 1 argument. condition: %+v", cond.Operation, cond)
			}

		case traceql.OperationIn, traceql.OperationRegexIn:
			if opCount == 0 {
				return fmt.Errorf("operand IN requires at least 1 argument. condition: %+v", cond)
			}

		default:
			return fmt.Errorf("unknown operation. condition: %+v", cond)
		}
	}
	return nil
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

	spanRequireAtLeastOneMatch := len(spanConditions) > 0 && len(resourceConditions) == 0
	spanIter, err := createSpanIterator(makeIter, spanConditions, spanRequireAtLeastOneMatch)
	if err != nil {
		return nil, errors.Wrap(err, "creating span iterator")
	}

	batchRequireAtLeastOneMatch := len(spanConditions) == 0 && len(resourceConditions) > 0
	resourceIter, err := createResourceIterator(makeIter, spanIter, resourceConditions, batchRequireAtLeastOneMatch, len(conditions) > 0)
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
func createSpanIterator(makeIter makeIterFunc, conditions []traceql.Condition, requireAtLeastOneMatch bool) (parquetquery.Iterator, error) {

	var (
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
	)

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

		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, ".span.", DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	// Static columns that are always loaded
	iters = append(iters, makeIter(columnPathSpanID, nil, columnPathSpanID))
	iters = append(iters, makeIter(columnPathSpanStartTime, nil, columnPathSpanStartTime))
	iters = append(iters, makeIter(columnPathSpanEndTime, nil, columnPathSpanEndTime))

	spanIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpansILSSpan, iters, &spanCollector{requireAtLeastOneMatch})

	return spanIter, nil
}

// createResourceIter iterates through all resourcespans-level (batch-level) columns, groups them into rows representing
// one batch each. It builds on top of the span iterator, and turns the groups of spans and resource-level values into
// spansets.  Spansets are returned that match any of the given conditions.

func createResourceIterator(makeIter makeIterFunc, spanIterator parquetquery.Iterator, conditions []traceql.Condition, requireAtLeastOneMatch, requireAtLeastOneMatchOverall bool) (parquetquery.Iterator, error) {
	var (
		iters             = []parquetquery.Iterator{spanIterator}
		genericConditions []traceql.Condition
	)

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

		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, ".resource.", DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	// Final resource iterator.
	// Union to return resources that match any of the conditions.
	// BatchCollector to group the spans into spanset
	resourceIter := parquetquery.NewUnionIterator(DefinitionLevelResourceSpans, iters, &batchCollector{requireAtLeastOneMatch, requireAtLeastOneMatchOverall})

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

	vals := make([]string, 0, len(operands))

	for _, op := range operands {
		s, ok := op.(string)
		if !ok {
			return nil, fmt.Errorf("operand is not string: %+v", op)
		}
		vals = append(vals, s)
	}

	switch op {
	case traceql.OperationEq, traceql.OperationIn:
		return parquetquery.NewStringInPredicate(vals), nil

	case traceql.OperationRegexIn:
		return parquetquery.NewRegexInPredicate(vals)

	default:
		return nil, fmt.Errorf("operand not supported for strings: %+v", op)
	}

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

func createFloatPredicate(op traceql.Operation, operands []interface{}) (parquetquery.Predicate, error) {
	if op == traceql.OperationNone {
		return nil, nil
	}

	// Ensure operand is int
	var i float64
	switch v := operands[0].(type) {
	case float32:
		i = float64(v)
	case float64:
		i = v
	default:
		return nil, fmt.Errorf("operand is not float: %+v", operands[0])
	}

	// Defaults
	min := math.Inf(-1)
	max := math.Inf(1)

	switch op {
	case traceql.OperationEq:
		min = i
		max = i
	case traceql.OperationGT:
		min = math.Nextafter(i, max)
	case traceql.OperationLT:
		max = math.Nextafter(i, min)
	}

	return parquetquery.NewFloatBetweenPredicate(min, max), nil
}

func createBoolPredicate(op traceql.Operation, operands []interface{}) (parquetquery.Predicate, error) {
	if op == traceql.OperationNone {
		return nil, nil
	}

	// Ensure operand is bool
	var b bool
	switch v := operands[0].(type) {
	case bool:
		b = v
	default:
		return nil, fmt.Errorf("operand is not bool: %+v", operands[0])
	}

	return parquetquery.NewBoolPredicate(b), nil
}

func createAttributeIterator(makeIter makeIterFunc, conditions []traceql.Condition,
	prefix string,
	definitionLevel int,
	keyPath, strPath, intPath, floatPath, boolPath string,
) (parquetquery.Iterator, error) {
	var (
		attrKeys        = []string{}
		attrStringPreds = []parquetquery.Predicate{}
		attrIntPreds    = []parquetquery.Predicate{}
		attrFltPreds    = []parquetquery.Predicate{}
		boolPreds       = []parquetquery.Predicate{}
	)
	for _, cond := range conditions {

		// Arbitrary attribute lookup
		s := cond.Selector
		s = strings.TrimPrefix(s, prefix)
		s = strings.TrimPrefix(s, ".")
		attrKeys = append(attrKeys, s)

		if cond.Operation == traceql.OperationNone {
			// This means we have to scan all values, we don't know what type
			// to expect
			attrStringPreds = append(attrStringPreds, nil)
			attrIntPreds = append(attrIntPreds, nil)
			attrFltPreds = append(attrFltPreds, nil)
			boolPreds = append(boolPreds, nil)
			continue
		}

		var (
			stringOperands = []interface{}{}
			intOperands    = []interface{}{}
			fltOperands    = []interface{}{}
			boolOperands   = []interface{}{}
		)
		for _, op := range cond.Operands {
			switch opv := op.(type) {
			case string:
				stringOperands = append(stringOperands, opv)
			case int, int64:
				intOperands = append(intOperands, opv)
			case float32, float64:
				fltOperands = append(fltOperands, opv)
			case bool:
				boolOperands = append(boolOperands, opv)
			}
		}

		if len(stringOperands) > 0 {
			pred, err := createStringPredicate(cond.Operation, stringOperands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrStringPreds = append(attrStringPreds, pred)
		}

		if len(intOperands) > 0 {
			pred, err := createIntPredicate(cond.Operation, intOperands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrIntPreds = append(attrIntPreds, pred)
		}

		if len(fltOperands) > 0 {
			pred, err := createFloatPredicate(cond.Operation, fltOperands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrFltPreds = append(attrFltPreds, pred)
		}
		if len(boolOperands) > 0 {
			pred, err := createBoolPredicate(cond.Operation, boolOperands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			boolPreds = append(boolPreds, pred)
		}
	}
	var attrIters []parquetquery.Iterator
	if len(attrKeys) > 0 {
		attrIters = append(attrIters, makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key"))
	}
	if len(attrStringPreds) > 0 {
		attrIters = append(attrIters, makeIter(strPath, parquetquery.NewOrPredicate(attrStringPreds...), "string"))
	}
	if len(attrIntPreds) > 0 {
		attrIters = append(attrIters, makeIter(intPath, parquetquery.NewOrPredicate(attrIntPreds...), "int"))
	}
	if len(attrFltPreds) > 0 {
		attrIters = append(attrIters, makeIter(floatPath, parquetquery.NewOrPredicate(attrFltPreds...), "float"))
	}
	if len(boolPreds) > 0 {
		attrIters = append(attrIters, makeIter(boolPath, parquetquery.NewOrPredicate(boolPreds...), "bool"))
	}

	var iter parquetquery.Iterator
	if len(attrIters) > 0 {
		iter = parquetquery.NewUnionIterator(definitionLevel, attrIters, &attributeCollector{})
	}
	return iter, nil
}

// This turns groups of span values into Span objects
type spanCollector struct {
	requireAtLeastOneMatch bool
}

var _ parquetquery.GroupPredicate = (*spanCollector)(nil)

func (c *spanCollector) KeepGroup(res *parquetquery.IteratorResult) bool {

	span := &traceql.Span{
		Attributes: make(map[string]interface{}),
	}

	for _, e := range res.OtherEntries {
		span.Attributes[e.Key] = e.Value
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
			// TODO - This exists for span-level dedicated columns like http.status_code
			// Are nils possible here?
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

	if c.requireAtLeastOneMatch {
		matchFound := false
		for _, v := range span.Attributes {
			if v != nil {
				matchFound = true
				break
			}
		}

		if !matchFound {
			return false
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("span", span)

	return true
}

// batchCollector receives rows of matching resource-level
// This turns groups of batch values and Spans into SpanSets
type batchCollector struct {
	requireAtLeastOneMatch        bool
	requireAtLeastOneMatchOverall bool
}

var _ parquetquery.GroupPredicate = (*batchCollector)(nil)

func (c *batchCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	sp := &traceql.Spanset{}

	resAttrs := make(map[string]interface{})

	for _, kv := range res.OtherEntries {
		if span, ok := kv.Value.(*traceql.Span); ok {
			sp.Spans = append(sp.Spans, *span)
			continue
		}
		resAttrs[kv.Key] = kv.Value
	}

	// Throw out batches without any spans
	if len(sp.Spans) == 0 {
		return false
	}

	// Gather Attributes from dedicated resource-level columns
	for _, e := range res.Entries {
		switch e.Value.Kind() {
		case parquet.Int64:
			resAttrs[e.Key] = e.Value.Int64()
		case parquet.ByteArray:
			resAttrs[e.Key] = e.Value.String()
		}
	}

	if c.requireAtLeastOneMatch && len(resAttrs) == 0 {
		return false
	}

	// Copy resource-level attributes to the individual spans now
	for k, v := range resAttrs {
		for _, span := range sp.Spans {
			if _, alreadyExists := span.Attributes[k]; !alreadyExists {
				span.Attributes[k] = v
			}
		}
	}

	// Remove unmatched attributes
	for _, span := range sp.Spans {
		for k, v := range span.Attributes {
			if v == nil {
				delete(span.Attributes, k)
			}
		}
	}

	// Throw out batches that don't have any attributes
	if c.requireAtLeastOneMatchOverall {
		matchFound := false
		for _, span := range sp.Spans {
			if len(span.Attributes) > 0 {
				matchFound = true
				break
			}
		}
		if !matchFound {
			return false
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("spanset", sp)

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

	for _, e := range res.Entries {
		switch e.Key {
		case columnPathTraceID:
			spanset.TraceID = e.Value.ByteArray()
		}
	}

	return true
}

// attributeCollector receives rows from the individual key/string/int/etc
// columns and joins them together into map[key]value entries with the
// right type.
type attributeCollector struct {
}

var _ parquetquery.GroupPredicate = (*attributeCollector)(nil)

func (c *attributeCollector) KeepGroup(res *parquetquery.IteratorResult) bool {

	var key string
	var val interface{}

	for _, e := range res.Entries {
		switch e.Key {
		case "key":
			key = e.Value.String()
		case "string":
			if e.Value.Kind() >= 0 {
				val = e.Value.String()
			}
		case "int":
			if e.Value.Kind() >= 0 {
				val = e.Value.Int64()
			}
		case "float":
			if e.Value.Kind() >= 0 {
				val = e.Value.Double()
			}
		case "bool":
			if e.Value.Kind() >= 0 {
				val = e.Value.Boolean()
			}
		}
	}

	if key == "" {
		// This means we got a value within range but some other key.
		// Just ignore
		return false
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(key, val)

	return true
}
