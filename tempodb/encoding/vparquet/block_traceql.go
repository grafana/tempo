package vparquet

import (
	"context"
	"fmt"
	"math"
	"reflect"

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
type makePredicateForCondition func(traceql.Operator, traceql.Operands) (parquetquery.Predicate, error)

const (
	columnPathTraceID                  = "TraceID"
	columnPathResourceAttrKey          = "rs.Resource.Attrs.Key"
	columnPathResourceAttrString       = "rs.Resource.Attrs.Value"
	columnPathResourceAttrInt          = "rs.Resource.Attrs.ValueInt"
	columnPathResourceAttrDouble       = "rs.Resource.Attrs.ValueDouble"
	columnPathResourceAttrBool         = "rs.Resource.Attrs.ValueBool"
	columnPathResourceServiceName      = "rs.Resource.ServiceName"
	columnPathResourceCluster          = "rs.Resource.Cluster"
	columnPathResourceNamespace        = "rs.Resource.Namespace"
	columnPathResourcePod              = "rs.Resource.Pod"
	columnPathResourceContainer        = "rs.Resource.Container"
	columnPathResourceK8sClusterName   = "rs.Resource.K8sClusterName"
	columnPathResourceK8sNamespaceName = "rs.Resource.K8sNamespaceName"
	columnPathResourceK8sPodName       = "rs.Resource.K8sPodName"
	columnPathResourceK8sContainerName = "rs.Resource.K8sContainerName"

	columnPathSpanID             = "rs.ils.Spans.ID"
	columnPathSpanName           = "rs.ils.Spans.Name"
	columnPathSpanStartTime      = "rs.ils.Spans.StartUnixNanos"
	columnPathSpanEndTime        = "rs.ils.Spans.EndUnixNanos"
	columnPathSpanAttrKey        = "rs.ils.Spans.Attrs.Key"
	columnPathSpanAttrString     = "rs.ils.Spans.Attrs.Value"
	columnPathSpanAttrInt        = "rs.ils.Spans.Attrs.ValueInt"
	columnPathSpanAttrDouble     = "rs.ils.Spans.Attrs.ValueDouble"
	columnPathSpanAttrBool       = "rs.ils.Spans.Attrs.ValueBool"
	columnPathSpanHttpStatusCode = "rs.ils.Spans.HttpStatusCode"
	columnPathSpanHttpMethod     = "rs.ils.Spans.HttpMethod"
	columnPathSpanHttpUrl        = "rs.ils.Spans.HttpUrl"
)

// Lookup table of all well-known attributes with dedicated columns
var wellKnownColumnLookups = map[string]struct {
	columnPath string                 // path.to.column
	level      traceql.AttributeScope // span or resource level
	//predicateFn makePredicateForCondition // Predicate fn
	typ traceql.StaticType // Data type
}{
	// Resource-level columns
	LabelServiceName:      {columnPathResourceServiceName, traceql.AttributeScopeResource, traceql.TypeString},
	LabelCluster:          {columnPathResourceCluster, traceql.AttributeScopeResource, traceql.TypeString},
	LabelNamespace:        {columnPathResourceNamespace, traceql.AttributeScopeResource, traceql.TypeString},
	LabelPod:              {columnPathResourcePod, traceql.AttributeScopeResource, traceql.TypeString},
	LabelContainer:        {columnPathResourceContainer, traceql.AttributeScopeResource, traceql.TypeString},
	LabelK8sClusterName:   {columnPathResourceK8sClusterName, traceql.AttributeScopeResource, traceql.TypeString},
	LabelK8sNamespaceName: {columnPathResourceK8sNamespaceName, traceql.AttributeScopeResource, traceql.TypeString},
	LabelK8sPodName:       {columnPathResourceK8sPodName, traceql.AttributeScopeResource, traceql.TypeString},
	LabelK8sContainerName: {columnPathResourceK8sContainerName, traceql.AttributeScopeResource, traceql.TypeString},

	// Span-level columns
	LabelHTTPStatusCode: {columnPathSpanHttpStatusCode, traceql.AttributeScopeSpan, traceql.TypeInt},
	LabelHTTPMethod:     {columnPathSpanHttpMethod, traceql.AttributeScopeSpan, traceql.TypeString},
	LabelHTTPUrl:        {columnPathSpanHttpUrl, traceql.AttributeScopeSpan, traceql.TypeString},
}

// Fetch spansets from the block for the given TraceQL FetchSpansRequest. The request is checked for
// internal consistencies:  operand count matches the operation, all operands in each condition are identical
// types, and the operand type is compatible with the operation.
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

	iter, err := fetch(ctx, req, pf)
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

		switch cond.Op {

		case traceql.OpNone:
			if opCount != 0 {
				return fmt.Errorf("operanion none must have 0 arguments. condition: %+v", cond)
			}

		case traceql.OpEqual, traceql.OpGreater, traceql.OpLess, traceql.OpRegex:
			if opCount != 1 {
				return fmt.Errorf("operation %v must have exactly 1 argument. condition: %+v", cond.Op, cond)
			}

		default:
			return fmt.Errorf("unknown operation. condition: %+v", cond)
		}

		// Verify all operands are of the same type
		if opCount == 0 {
			continue
		}

		for i := 1; i < opCount; i++ {
			if reflect.TypeOf(cond.Operands[0]) != reflect.TypeOf(cond.Operands[i]) {
				return fmt.Errorf("operands must be of the same type. condition: %+v", cond)
			}
		}
	}

	return nil
}

func operandType(operands traceql.Operands) traceql.StaticType {
	if len(operands) > 0 {
		return operands[0].Type
	}
	return traceql.TypeNil
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

// fetch is the core logic for executing the given conditions against the parquet columns. The algorithm
// can be summarized as a hiearchy of iterators where we iterate related columns together and collect the results
// at each level into attributes, spans, and spansets.  Each condition (.foo=bar) is pushed down to the one or more
// matching columns using parquetquery.Predicates.  Results are collected The final return is an iterator where each result is 1 Spanset for each trace.
//
// Diagram:
//
//  Span attribute iterator: key    -----------------------------
//                           ...    --------------------------  |
//  Span attribute iterator: valueN ----------------------|  |  |
//                                                        |  |  |
//                                                        V  V  V
//                                                     -------------
//                                                     | attribute |
//                                                     | collector |
//                                                     -------------
//                                                            |
//                                                            | List of attributes
//                                                            |
//                                                            |
//  Span column iterator 1    ---------------------------     |
//                      ...   ------------------------  |     |
//  Span column iterator N    ---------------------  |  |     |
//    (ex: name, status)                          |  |  |     |
//                                                V  V  V     V
//                                            ------------------
//                                            | span collector |
//                                            ------------------
//                                                            |
//                                                            | List of Spans
//  Resource attribute                                        |
//   iterators:                                               |
//     key     -----------------------------------------      |
//     ...     --------------------------------------  |      |
//     valueN  -----------------------------------  |  |      |
//                                               |  |  |      |
//                                               V  V  V      |
//                                            -------------   |
//                                            | attribute |   |
//                                            | collector |   |
//                                            -------------   |
//                                                      |     |
//                                                      |     |
//                                                      |     |
//                                                      |     |
// Resource column iterator 1  --------------------     |     |
//                      ...    -----------------  |     |     |
// Resource column iterator N  --------------  |  |     |     |
//    (ex: service.name)                    |  |  |     |     |
//                                          V  V  V     V     V
//                                         ----------------------
//                                         |   batch collector  |
//                                         ----------------------
//                                                            |
//                                                            | List of Spansets
// Trace column iterator 1  --------------------------        |
//                      ... -----------------------  |        |
// Trace column iterator N  --------------------  |  |        |
//    (ex: trace ID)                           |  |  |        |
//                                             V  V  V        V
//                                           -------------------
//                                           | trace collector |
//                                           -------------------
//                                                            |
//                                                            | Final Spanset
//                                                            |
//                                                            V

func fetch(ctx context.Context, req traceql.FetchSpansRequest, pf *parquet.File) (*spansetIterator, error) {

	// Categorize conditions into span-level or resource-level
	var (
		spanConditions     []traceql.Condition
		resourceConditions []traceql.Condition
	)
	for _, cond := range req.Conditions {

		switch cond.Attribute.Scope {

		case traceql.AttributeScopeNone:
			spanConditions = append(spanConditions, cond)
			resourceConditions = append(resourceConditions, cond)
			continue

		case traceql.AttributeScopeSpan:
			spanConditions = append(spanConditions, cond)
			continue

		case traceql.AttributeScopeResource:
			resourceConditions = append(resourceConditions, cond)
			continue

		default:
			return nil, fmt.Errorf("unsupported traceql scope: %s", cond.Attribute)
		}
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

	// Global state
	// Span-filtering behavior changes depending on the resource-filtering in effect,
	// and vice-versa.  For example consider the query { span.a=1 }.  If no spans have a=1
	// then it generate the empty spanset.
	// However once we add a resource condition: { span.a=1 || resource.b=2 }, now the span
	// filtering must return all spans, even if no spans have a=1, because they might be
	// matched upstream to a resource.
	var (
		// If there are only span conditions, then don't return a span upstream
		// unless it matches at least 1 span-level condition.
		spanRequireAtLeastOneMatch = len(spanConditions) > 0 && len(resourceConditions) == 0

		// If there are only resource conditions, then don't return a resource upstream
		// unless it matches at least 1 resource-level condition.
		batchRequireAtLeastOneMatch = len(spanConditions) == 0 && len(resourceConditions) > 0

		// Don't return the final spanset upstream unless it matched at least 1 condition
		// anywhere, except in the case of the empty query: {}
		batchRequireAtLeastOneMatchOverall = len(req.Conditions) > 0
	)

	spanIter, err := createSpanIterator(makeIter, spanConditions, req.StartTimeUnixNanos, req.EndTimeUnixNanos, spanRequireAtLeastOneMatch)
	if err != nil {
		return nil, errors.Wrap(err, "creating span iterator")
	}

	resourceIter, err := createResourceIterator(makeIter, spanIter, resourceConditions, batchRequireAtLeastOneMatch, batchRequireAtLeastOneMatchOverall)
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
func createSpanIterator(makeIter makeIterFunc, conditions []traceql.Condition, start, end uint64, requireAtLeastOneMatch bool) (parquetquery.Iterator, error) {

	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
		durationFilter    = false
		durationMin       = uint64(math.MaxUint64) // Initially reversed to exclude all
		durationMax       = uint64(0)              // Initially reversed to exclude all
	)

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	for _, cond := range conditions {

		// Intrinsic?
		switch cond.Attribute.Intrinsic {

		case traceql.IntrinsicName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanName, pred)
			columnSelectAs[columnPathSpanName] = cond.Attribute.String()
			continue

		case traceql.IntrinsicDuration:
			if cond.Operands[0].Type != traceql.TypeDuration {
				return nil, fmt.Errorf("operand %v is not duration", cond.Operands[0])
			}
			durationFilter = true
			v := uint64(cond.Operands[0].D.Nanoseconds())
			// This is kind of hacky. Merge all duration filters onto the min/max range
			switch cond.Op {
			case traceql.OpEqual:
				if v < durationMin {
					durationMin = v
				}
				if v > durationMax {
					durationMax = v
				}
			case traceql.OpGreater:
				durationMax = uint64(math.MaxUint64)
				if v < durationMin {
					durationMin = v
				}
			case traceql.OpLess:
				durationMin = 0
				if v > durationMax {
					durationMax = v
				}
			default:
				return nil, fmt.Errorf("operator %v not supported for duration", cond.Op)
			}
			continue
		}

		// Well-known attribute?
		if entry, ok := wellKnownColumnLookups[cond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeResource {
			if cond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				columnSelectAs[entry.columnPath] = cond.Attribute.Name
				continue
			}

			// Compatible type?
			if entry.typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				addPredicate(entry.columnPath, pred)
				columnSelectAs[entry.columnPath] = cond.Attribute.Name
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
	}

	// Time range filtering?
	var startFilter, endFilter parquetquery.Predicate
	if start > 0 && end > 0 {
		// Here's how we detect the span overlaps the time window:
		// Span start <= req.End
		// Span end >= req.Start
		startFilter = parquetquery.NewIntBetweenPredicate(0, int64(end))
		endFilter = parquetquery.NewIntBetweenPredicate(int64(start), math.MaxInt64)
	}

	// Static columns that are always loaded
	var required []parquetquery.Iterator
	required = append(required, makeIter(columnPathSpanID, nil, columnPathSpanID))
	required = append(required, makeIter(columnPathSpanStartTime, startFilter, columnPathSpanStartTime))
	required = append(required, makeIter(columnPathSpanEndTime, endFilter, columnPathSpanEndTime))

	spanCol := &spanCollector{
		requireAtLeastOneMatch,
		durationFilter,
		durationMin,
		durationMax,
	}

	// This is an optimization for cases when only span conditions are
	// present and we require at least one of them to match.  Wrap
	// up the individual conditions with a union and move it into the
	// required list.  This skips over static columns like ID that are
	// omnipresent.
	if requireAtLeastOneMatch && len(iters) > 0 {
		required = append(required, parquetquery.NewUnionIterator(DefinitionLevelResourceSpansILSSpan, iters, nil))
		iters = nil
	}

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, spanCol), nil
}

// createResourceIterator iterates through all resourcespans-level (batch-level) columns, groups them into rows representing
// one batch each. It builds on top of the span iterator, and turns the groups of spans and resource-level values into
// spansets.  Spansets are returned that match any of the given conditions.
func createResourceIterator(makeIter makeIterFunc, spanIterator parquetquery.Iterator, conditions []traceql.Condition, requireAtLeastOneMatch, requireAtLeastOneMatchOverall bool) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		iters             = []parquetquery.Iterator{}
		genericConditions []traceql.Condition
	)

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	for _, cond := range conditions {

		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[cond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeSpan {
			if cond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				columnSelectAs[entry.columnPath] = cond.Attribute.Name
				continue
			}

			// Compatible type?
			if entry.typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				iters = append(iters, makeIter(entry.columnPath, pred, cond.Attribute.Name))
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	batchCol := &batchCollector{
		requireAtLeastOneMatch,
		requireAtLeastOneMatchOverall,
	}

	required := []parquetquery.Iterator{
		spanIterator,
	}

	// This is an optimization for cases when only resource conditions are
	// present and we require at least one of them to match.  Wrap
	// up the individual conditions with a union and move it into the
	// required list.
	if requireAtLeastOneMatch && len(iters) > 0 {
		required = append(required, parquetquery.NewUnionIterator(DefinitionLevelResourceSpans, iters, nil))
		iters = nil
	}

	// Left join here means the span iterator + 1 are required,
	// and all other resource conditions are optional. Whatever matches
	// is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpans,
		required, iters, batchCol), nil
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

func createPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	switch operands[0].Type {
	case traceql.TypeString:
		return createStringPredicate(op, operands)
	case traceql.TypeInt:
		return createIntPredicate(op, operands)
	case traceql.TypeFloat:
		return createFloatPredicate(op, operands)
	case traceql.TypeBoolean:
		return createBoolPredicate(op, operands)
	default:
		return nil, fmt.Errorf("cannot create predicate for operand: %v", operands[0])
	}
}

func createStringPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	vals := make([]string, 0, len(operands))

	for _, op := range operands {
		if op.Type != traceql.TypeString {
			return nil, fmt.Errorf("operand is not string: %+v", op)
		}
		vals = append(vals, op.S)
	}

	switch op {
	case traceql.OpEqual:
		return parquetquery.NewStringInPredicate(vals), nil

	case traceql.OpRegex:
		return parquetquery.NewRegexInPredicate(vals)

	default:
		return nil, fmt.Errorf("operand not supported for strings: %+v", op)
	}

}

func createIntPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	// Ensure operand is int
	if operands[0].Type != traceql.TypeInt {
		return nil, fmt.Errorf("operand is not int: %+v", operands[0])
	}

	// Defaults
	i := int64(operands[0].N)
	min := int64(math.MinInt64)
	max := int64(math.MaxInt64)

	switch op {
	case traceql.OpEqual:
		min = i
		max = i
	case traceql.OpGreater:
		min = i + 1
	case traceql.OpLess:
		max = i - 1
	default:
		return nil, fmt.Errorf("operand not supported for integers: %+v", op)
	}

	return parquetquery.NewIntBetweenPredicate(min, max), nil
}

func createFloatPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	// Ensure operand is float
	if operands[0].Type != traceql.TypeFloat {
		return nil, fmt.Errorf("operand is not float: %+v", operands[0])
	}

	// Defaults
	i := operands[0].F
	min := math.Inf(-1)
	max := math.Inf(1)

	switch op {
	case traceql.OpEqual:
		min = i
		max = i
	case traceql.OpGreater:
		min = math.Nextafter(i, max)
	case traceql.OpLess:
		max = math.Nextafter(i, min)
	default:
		return nil, fmt.Errorf("operand not supported for floats: %+v", op)
	}

	return parquetquery.NewFloatBetweenPredicate(min, max), nil
}

func createBoolPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	// Ensure operand is bool
	if operands[0].Type != traceql.TypeBoolean {
		return nil, fmt.Errorf("operand is not bool: %+v", operands[0])
	}

	switch op {
	case traceql.OpEqual:
		return parquetquery.NewBoolPredicate(operands[0].B), nil

	default:
		return nil, fmt.Errorf("operand not supported for booleans: %+v", op)
	}
}

func createAttributeIterator(makeIter makeIterFunc, conditions []traceql.Condition,
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

		attrKeys = append(attrKeys, cond.Attribute.Name)

		if cond.Op == traceql.OpNone {
			// This means we have to scan all values, we don't know what type
			// to expect
			attrStringPreds = append(attrStringPreds, nil)
			attrIntPreds = append(attrIntPreds, nil)
			attrFltPreds = append(attrFltPreds, nil)
			boolPreds = append(boolPreds, nil)
			continue
		}

		switch cond.Operands[0].Type {

		case traceql.TypeString:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrStringPreds = append(attrStringPreds, pred)

		case traceql.TypeInt:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrIntPreds = append(attrIntPreds, pred)

		case traceql.TypeFloat:
			pred, err := createFloatPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			attrFltPreds = append(attrFltPreds, pred)

		case traceql.TypeBoolean:
			pred, err := createBoolPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, errors.Wrap(err, "creating attribute predicate")
			}
			boolPreds = append(boolPreds, pred)
		}
	}

	var valueIters []parquetquery.Iterator
	if len(attrStringPreds) > 0 {
		valueIters = append(valueIters, makeIter(strPath, parquetquery.NewOrPredicate(attrStringPreds...), "string"))
	}
	if len(attrIntPreds) > 0 {
		valueIters = append(valueIters, makeIter(intPath, parquetquery.NewOrPredicate(attrIntPreds...), "int"))
	}
	if len(attrFltPreds) > 0 {
		valueIters = append(valueIters, makeIter(floatPath, parquetquery.NewOrPredicate(attrFltPreds...), "float"))
	}
	if len(boolPreds) > 0 {
		valueIters = append(valueIters, makeIter(boolPath, parquetquery.NewOrPredicate(boolPreds...), "bool"))
	}

	if len(valueIters) > 0 {
		// LeftJoin means only look at rows where the key is what we want.
		// Bring in any of the typed values as needed.
		return parquetquery.NewLeftJoinIterator(definitionLevel,
			[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
			valueIters,
			&attributeCollector{}), nil
	}

	return nil, nil
}

// This turns groups of span values into Span objects
type spanCollector struct {
	requireAtLeastOneMatch bool

	duration                 bool
	durationMin, durationMax uint64
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
			case parquet.Boolean:
				span.Attributes[kv.Key] = kv.Value.Boolean()
			case parquet.Int32, parquet.Int64:
				span.Attributes[kv.Key] = kv.Value.Int64()
			case parquet.Float:
				span.Attributes[kv.Key] = kv.Value.Float()
			case parquet.ByteArray:
				span.Attributes[kv.Key] = kv.Value.String()
			}
		}
	}

	// TODO - We don't have a dedicated span duration column (oops)
	// so for now we calculate it from the much larger start and end columns
	// Introduce a dedicated column for efficiency
	if c.duration {
		dur := span.EndtimeUnixNanos - span.StartTimeUnixNanos
		if dur < c.durationMin || dur > c.durationMax {
			return false
		}
		// This satisfies subsequent logic that checks to see if the span
		// ever matched anything.  TODO: Find a more efficient way to do
		// this since duration is already present in the span data (start/end times)
		// We need to flag that this span matched "something"
		span.Attributes[traceql.IntrinsicDuration.String()] = dur
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

	// TODO - This wraps everything up in a spanset per batch.
	// We probably don't need to do this, since the traceCollector
	// flattens it into 1 spanset per trace.  All we really need
	// todo is merge the resource-level attributes onto the spans
	// and filter out spans that didn't match anything.

	resAttrs := make(map[string]interface{})
	spans := make([]traceql.Span, 0, len(res.OtherEntries))

	for _, kv := range res.OtherEntries {
		if span, ok := kv.Value.(*traceql.Span); ok {
			spans = append(spans, *span)
			continue
		}

		// Attributes show up here
		resAttrs[kv.Key] = kv.Value
	}

	// Throw out batches without any spans
	if len(spans) == 0 {
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
		for _, span := range spans {
			if _, alreadyExists := span.Attributes[k]; !alreadyExists {
				span.Attributes[k] = v
			}
		}
	}

	// Remove unmatched attributes
	for _, span := range spans {
		for k, v := range span.Attributes {
			if v == nil {
				delete(span.Attributes, k)
			}
		}
	}

	sp := &traceql.Spanset{
		Spans: make([]traceql.Span, 0, len(spans)),
	}

	// Copy over only spans that matched something
	if c.requireAtLeastOneMatchOverall {
		for _, span := range spans {
			if len(span.Attributes) > 0 {
				sp.Spans = append(sp.Spans, span)
			}
		}
	} else {
		sp.Spans = spans
	}

	// Throw out batches without any spans
	if len(sp.Spans) == 0 {
		return false
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
	finalSpanset := &traceql.Spanset{}

	for _, e := range res.Entries {
		switch e.Key {
		case columnPathTraceID:
			finalSpanset.TraceID = e.Value.ByteArray()
		}
	}

	for _, e := range res.OtherEntries {
		if spanset, ok := e.Value.(*traceql.Spanset); ok {
			finalSpanset.Spans = append(finalSpanset.Spans, spanset.Spans...)
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue("spanset", finalSpanset)

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
		// Ignore nulls, this leaves val as the remaining found value,
		// or nil if the key was found but no matching values
		if e.Value.Kind() < 0 {
			continue
		}

		switch e.Key {
		case "key":
			key = e.Value.String()
		case "string":
			val = e.Value.String()
		case "int":
			val = e.Value.Int64()
		case "float":
			val = e.Value.Double()
		case "bool":
			val = e.Value.Boolean()
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(key, val)

	return true
}
