package vparquet5

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
)

func (b *backendBlock) FetchSpansOnly(ctx context.Context, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansOnlyResponse, error) {
	pf, rr, err := b.openForSearch(ctx, opts)
	if err != nil {
		return traceql.FetchSpansOnlyResponse{}, err
	}

	rgs := rowGroupsFromFile(pf, opts)

	makeIter := makeIterFunc(ctx, rgs, pf)

	iter, span, err := create(makeIter, nil, req.Conditions, req.StartTimeUnixNanos, req.EndTimeUnixNanos, req.AllConditions, false, b.meta.DedicatedColumns)
	if err != nil {
		return traceql.FetchSpansOnlyResponse{}, err
	}

	if len(req.SecondPassConditions) > 0 || req.SecondPassSelectAll {
		iter, span, err = create(makeIter, iter, req.SecondPassConditions, req.StartTimeUnixNanos, req.EndTimeUnixNanos, false, req.SecondPassSelectAll, b.meta.DedicatedColumns)
		if err != nil {
			return traceql.FetchSpansOnlyResponse{}, err
		}
	}

	return traceql.FetchSpansOnlyResponse{
		Results: &spanOnlyIterator{iter: iter, span: span},
		Bytes:   func() uint64 { return rr.BytesRead() },
	}, nil
}

type spanOnlyIterator struct {
	iter parquetquery.Iterator
	span *span
}

var _ traceql.SpanIterator = (*spanOnlyIterator)(nil)

func (i *spanOnlyIterator) Next(ctx context.Context) (traceql.Span, error) {
	res, err := i.iter.Next()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	// There is always exactly one buffer span and we reuse it.
	// return res.OtherEntries[0].Value.(traceql.Span), nil
	return i.span, nil
}

func (i *spanOnlyIterator) Close() {
	i.iter.Close()
}

func create(makeIter makeIterFn,
	driver parquetquery.Iterator,
	conditions []traceql.Condition,
	start, end uint64,
	allConditions bool,
	selectAll bool,
	dedicatedColumns backend.DedicatedColumns,
) (parquetquery.Iterator, *span, error) {
	catConditions, mingledConditions, err := categorizeConditions(conditions)
	if err != nil {
		return nil, nil, err
	}

	// Optimization for queries like {resource.x... && span.y ...}
	// Requires no mingled scopes like .foo=x, which could be satisfied
	// by either resource or span.
	allConditions = allConditions && !mingledConditions

	// Don't return the final spanset upstream unless it matched at least 1 condition
	// anywhere, except in the case of the empty query: {}
	batchRequireAtLeastOneMatchOverall := len(conditions) > 0 && len(catConditions.trace) == 0

	traceIters, traceOptional, err := createTraceIterators(makeIter, catConditions.trace, start, end, allConditions, dedicatedColumns)
	if err != nil {
		return nil, nil, err
	}

	resIters, resOptional, err := createResourceIterators(makeIter, catConditions.resource /*batchRequireAtLeastOneMatchOverall,*/, allConditions, dedicatedColumns, selectAll)
	if err != nil {
		return nil, nil, err
	}

	spanIters, spanOptional, err := createSpanIterators(makeIter, driver == nil, catConditions.span, allConditions, selectAll, dedicatedColumns)
	if err != nil {
		return nil, nil, err
	}

	eventIters, eventOptional, err := createEventIterators(makeIter, catConditions.event, allConditions, selectAll)
	if err != nil {
		return nil, nil, err
	}

	linkIters, linkOptional, err := createLinkIterators(makeIter, catConditions.link, allConditions, selectAll)
	if err != nil {
		return nil, nil, err
	}

	instIters, instOptional, err := createInstrumentationIterators(makeIter, catConditions.instrumentation, allConditions, selectAll)
	if err != nil {
		return nil, nil, err
	}

	/*var required []parquetquery.Iterator
	var optional []parquetquery.Iterator

	if driver != nil {
		required = append(required, driver)
	}

	required = append(required, traceIters...)
	required = append(required, resIters...)
	required = append(required, spanIters...)
	optional = append(optional, traceOptional...)
	optional = append(optional, resOptional...)
	optional = append(optional, spanOptional...)*/

	spanCol := NewSpanCollector2()
	switch {
	case selectAll:
		// We are selecting everything so don't assert any restrictions on the number of attributes.
		spanCol.minAttributes = 0
	case allConditions:
		// Asserting that every condition is met.
		// So the number of matched attributes should match every distinct condition.
		distinct := map[string]struct{}{}
		for _, cond := range conditions {
			distinct[cond.Attribute.Name] = struct{}{}
		}
		spanCol.minAttributes = len(distinct)
	case batchRequireAtLeastOneMatchOverall:
		// TODO - Do we still need this?
		spanCol.minAttributes = 1
	}

	options := []parquetquery.LeftJoinIteratorOption{
		parquetquery.WithPool(pqSpanPool),
		parquetquery.WithCollector(spanCol),
	}

	if driver != nil {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, driver, false))
	}

	for _, iter := range traceIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelTrace, iter, false))
	}
	for _, iter := range resIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpans, iter, false))
	}
	for _, iter := range instIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelInstrumentationScope, iter, false))
	}
	for _, iter := range spanIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, false))
	}
	for _, iter := range eventIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, true))
	}
	for _, iter := range linkIters {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, true))
	}

	for _, iter := range traceOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelTrace, iter, true))
	}
	for _, iter := range resOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpans, iter, true))
	}
	for _, iter := range instOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelInstrumentationScope, iter, true))
	}
	for _, iter := range spanOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, true))
	}
	for _, iter := range eventOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, true))
	}
	for _, iter := range linkOptional {
		options = append(options, parquetquery.WithIterator(DefinitionLevelResourceSpansILSSpan, iter, true))
	}

	iter, err := parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, nil, nil, nil, options...)
	if err != nil {
		return nil, nil, err
	}

	return iter, &spanCol.at, nil
}

func createTraceIterators(
	makeIter makeIterFn,
	conditions []traceql.Condition,
	start, end uint64,
	allConditions bool,
	_ backend.DedicatedColumns,
) (required, optional []parquetquery.Iterator, err error) {
	for _, cond := range conditions {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicTraceID:
			optional = append(optional, makeIter(columnPathTraceID, nil, columnPathTraceID))
		case traceql.IntrinsicTraceDuration:
			optional = append(optional, makeIter(columnPathDurationNanos, nil, columnPathDurationNanos))
		case traceql.IntrinsicTraceStartTime:
			optional = append(optional, makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano))
		}
	}

	// evaluate time range
	// Time range filtering?
	if start > 0 && end > 0 {
		// Here's how we detect the span overlaps the time window:
		// Span start <= req.End
		// Span end >= req.Start
		var startFilter, endFilter parquetquery.Predicate
		startFilter = parquetquery.NewIntBetweenPredicate(0, int64(end))
		endFilter = parquetquery.NewIntBetweenPredicate(int64(start), math.MaxInt64)

		required = append(required, makeIter(columnPathStartTimeUnixNano, startFilter, columnPathStartTimeUnixNano))
		required = append(required, makeIter(columnPathEndTimeUnixNano, endFilter, columnPathEndTimeUnixNano))
	}

	// If all conditions move them to required
	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	return required, optional, nil
}

func createResourceIterators(
	makeIter makeIterFn,
	conditions []traceql.Condition,
	// requireAtLeastOneMatchOverall,
	allConditions bool,
	dedicatedColumns backend.DedicatedColumns,
	selectAll bool,
) (required, optional []parquetquery.Iterator, err error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		genericConditions []traceql.Condition
		columnMapping     = dedicatedColumnsToColumnMapping(dedicatedColumns, backend.DedicatedColumnScopeResource)
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
					return nil, nil, fmt.Errorf("creating predicate: %w", err)
				}
				optional = append(optional, makeIter(entry.columnPath, pred, cond.Attribute.Name))
				continue
			}
		}

		// Attributes stored in dedicated columns
		if c, ok := columnMapping.get(cond.Attribute.Name); ok {
			if cond.Op == traceql.OpNone {
				addPredicate(c.ColumnPath, nil) // No filtering
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}

			// Compatible type?
			typ, _ := c.Type.ToStaticType()
			if typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, nil, fmt.Errorf("creating predicate: %w", err)
				}
				addPredicate(c.ColumnPath, pred)
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	// SecondPass SelectAll
	if selectAll {
		for wellKnownAttr, entry := range wellKnownColumnLookups {
			if entry.level != traceql.AttributeScopeResource {
				continue
			}

			addPredicate(entry.columnPath, nil)
			columnSelectAs[entry.columnPath] = wellKnownAttr
		}

		for k, v := range columnMapping.mapping {
			addPredicate(v.ColumnPath, nil)
			columnSelectAs[v.ColumnPath] = k
		}
	}

	for columnPath, predicates := range columnPredicates {
		optional = append(optional, makeIter(columnPath, orIfNeeded(predicates), columnSelectAs[columnPath]))
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool, allConditions, selectAll)
	if err != nil {
		return nil, nil, fmt.Errorf("creating span attribute iterator: %w", err)
	}
	if attrIter != nil {
		optional = append(optional, attrIter)
	}

	/*minCount := 0
	if allConditions {
		// The final number of expected attributes
		distinct := map[string]struct{}{}
		for _, cond := range conditions {
			distinct[cond.Attribute.Name] = struct{}{}
		}
		minCount = len(distinct)
	}*/

	// This is an optimization for when all of the resource conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	return required, optional, nil
}

func createSpanIterators(
	makeIter makeIterFn,
	needDriver bool,
	conditions []traceql.Condition,
	allConditions bool,
	selectAll bool,
	dedicatedColumns backend.DedicatedColumns,
) (required, optional []parquetquery.Iterator, err error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		genericConditions []traceql.Condition
		columnMapping     = dedicatedColumnsToColumnMapping(dedicatedColumns, backend.DedicatedColumnScopeSpan)
	)

	// todo: improve these methods. if addPredicate gets a nil predicate shouldn't it just wipe out the existing predicates instead of appending?
	// nil predicate matches everything. what's the point of also evaluating a "real" predicate?
	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	addNilPredicateIfNotAlready := func(path string) {
		preds := columnPredicates[path]
		foundOpNone := false

		// check to see if there is a nil predicate and only add if it doesn't exist
		for _, pred := range preds {
			if pred == nil {
				foundOpNone = true
				break
			}
		}

		if !foundOpNone {
			addPredicate(path, nil)
			columnSelectAs[path] = path
		}
	}

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicNone:
		case traceql.IntrinsicSpanID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, true)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanID, pred)
			columnSelectAs[columnPathSpanID] = columnPathSpanID
			continue

		case traceql.IntrinsicParentID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, true)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanParentSpanID, pred)
			columnSelectAs[columnPathSpanParentSpanID] = columnPathSpanParentSpanID
			continue
		case traceql.IntrinsicSpanStartTime:
			// TODO - We also need to scale the operands if using lower precision.
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}

			/*if sampler != nil {
				pred = newSamplingPredicate(sampler, pred)
				// Removed so that it's not used down below.
				sampler = nil
			}*/

			// Choose the least precise column possible.
			// The step interval must be an even multiple of the pre-rounded precision.
			switch {
			case cond.Precision >= 3600*time.Second && cond.Precision%(3600*time.Second) == 0:
				addPredicate(columnPathSpanStartRounded3600, pred)
				columnSelectAs[columnPathSpanStartRounded3600] = columnPathSpanStartRounded3600
			case cond.Precision >= 300*time.Second && cond.Precision%(300*time.Second) == 0:
				addPredicate(columnPathSpanStartRounded300, pred)
				columnSelectAs[columnPathSpanStartRounded300] = columnPathSpanStartRounded300
			case cond.Precision >= 60*time.Second && cond.Precision%(60*time.Second) == 0:
				addPredicate(columnPathSpanStartRounded60, pred)
				columnSelectAs[columnPathSpanStartRounded60] = columnPathSpanStartRounded60
			case cond.Precision >= 15*time.Second && cond.Precision%(15*time.Second) == 0:
				addPredicate(columnPathSpanStartRounded15, pred)
				columnSelectAs[columnPathSpanStartRounded15] = columnPathSpanStartRounded15
			default:
				addPredicate(columnPathSpanStartTime, pred)
				columnSelectAs[columnPathSpanStartTime] = columnPathSpanStartTime
			}
			continue
		case traceql.IntrinsicName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanName, pred)
			columnSelectAs[columnPathSpanName] = columnPathSpanName
			continue

		case traceql.IntrinsicKind:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanKind, pred)
			columnSelectAs[columnPathSpanKind] = columnPathSpanKind
			continue

		case traceql.IntrinsicDuration:
			pred, err := createDurationPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanDuration, pred)
			columnSelectAs[columnPathSpanDuration] = columnPathSpanDuration
			continue

		case traceql.IntrinsicStatus:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanStatusCode, pred)
			columnSelectAs[columnPathSpanStatusCode] = columnPathSpanStatusCode
			continue
		case traceql.IntrinsicStatusMessage:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanStatusMessage, pred)
			columnSelectAs[columnPathSpanStatusMessage] = columnPathSpanStatusMessage
			continue

		case traceql.IntrinsicStructuralDescendant:
			addNilPredicateIfNotAlready(columnPathSpanNestedSetLeft)
			addNilPredicateIfNotAlready(columnPathSpanNestedSetRight)
			continue

		case traceql.IntrinsicStructuralChild:
			addNilPredicateIfNotAlready(columnPathSpanNestedSetLeft)
			addNilPredicateIfNotAlready(columnPathSpanParentID)
			continue

		case traceql.IntrinsicStructuralSibling:
			addNilPredicateIfNotAlready(columnPathSpanParentID)
			continue

		case traceql.IntrinsicNestedSetLeft:
			// nestedSetLeftExplicit = true
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanNestedSetLeft, pred)
			columnSelectAs[columnPathSpanNestedSetLeft] = columnPathSpanNestedSetLeft
			continue
		case traceql.IntrinsicNestedSetRight:
			// nestedSetRightExplicit = true
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanNestedSetRight, pred)
			columnSelectAs[columnPathSpanNestedSetRight] = columnPathSpanNestedSetRight
			continue
		case traceql.IntrinsicNestedSetParent:
			// nestedSetParentExplicit = true
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			addPredicate(columnPathSpanParentID, pred)
			columnSelectAs[columnPathSpanParentID] = columnPathSpanParentID
			continue
		default:
			panic("unhandled intrinsic: " + cond.Attribute.String())
		}

		// Attributes stored in dedicated columns
		if c, ok := columnMapping.get(cond.Attribute.Name); ok {
			if cond.Op == traceql.OpNone {
				addPredicate(c.ColumnPath, nil) // No filtering
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}

			// Compatible type?
			typ, _ := c.Type.ToStaticType()
			if typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, nil, fmt.Errorf("creating predicate: %w", err)
				}
				addPredicate(c.ColumnPath, pred)
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	// SecondPass SelectAll
	if selectAll {
		for wellKnownAttr, entry := range wellKnownColumnLookups {
			if entry.level != traceql.AttributeScopeSpan {
				continue
			}

			addPredicate(entry.columnPath, nil)
			columnSelectAs[entry.columnPath] = wellKnownAttr
		}

		for intrins, entry := range intrinsicColumnLookups {
			if entry.scope != intrinsicScopeSpan {
				continue
			}
			// These intrinsics aren't included in select all because they
			// aren't useful for compare().
			switch intrins {
			case traceql.IntrinsicSpanID,
				traceql.IntrinsicParentID,
				traceql.IntrinsicSpanStartTime,
				traceql.IntrinsicStructuralDescendant,
				traceql.IntrinsicStructuralChild,
				traceql.IntrinsicStructuralSibling,
				traceql.IntrinsicNestedSetLeft,
				traceql.IntrinsicNestedSetRight,
				traceql.IntrinsicNestedSetParent:
				continue
			}
			addPredicate(entry.columnPath, nil)
			columnSelectAs[entry.columnPath] = entry.columnPath
		}

		for k, v := range columnMapping.mapping {
			addPredicate(v.ColumnPath, nil)
			columnSelectAs[v.ColumnPath] = k
		}
	}

	for columnPath, predicates := range columnPredicates {
		optional = append(optional, makeIter(columnPath, orIfNeeded(predicates), columnSelectAs[columnPath]))
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool, allConditions, selectAll)
	if err != nil {
		return nil, nil, fmt.Errorf("creating span attribute iterator: %w", err)
	}
	if attrIter != nil {
		optional = append(optional, attrIter)
	}

	/*if len(innerIterators) != 0 {
		required = innerIterators
	}*/

	/*minCount := 0
	if allConditions {
		// The final number of expected attributes.
		distinct := map[string]struct{}{}
		for _, cond := range conditions {
			distinct[cond.Attribute.Name] = struct{}{}
		}
		minCount = len(distinct)
	}*/

	// This is an optimization for when all of the span conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	// if there are no direct conditions imposed on the span/span attributes level we are purposefully going to request the "Kind" column
	//  b/c it is extremely cheap to retrieve. retrieving matching spans in this case will allow aggregates such as "count" to be computed
	//  how do we know to pull duration for things like | avg(duration) > 1s? look at avg(span.http.status_code) it pushes a column request down here
	//  the entire engine is built around spans. we have to return at least one entry for every span to the layers above for things to work
	// TODO: note that if the query is { kind = client } the fetch layer will actually create two iterators over the kind column. this is evidence
	//  this spaniterator code could be tightened up
	// Also note that this breaks optimizations related to requireAtLeastOneMatch and requireAtLeastOneMatchOverall b/c it will add a kind attribute
	//  to the span attributes map in spanCollector
	if len(required) == 0 && needDriver {
		var pred parquetquery.Predicate
		/*if sampler != nil {
			pred = newSamplingPredicate(sampler, nil)
		}*/
		required = []parquetquery.Iterator{makeIter(columnPathSpanStatusCode, pred, "")}
	}

	return required, optional, nil
}

func createEventIterators(makeIter makeIterFn, conditions []traceql.Condition, allConditions bool, selectAll bool) (required, optional []parquetquery.Iterator, err error) {
	var genericConditions []traceql.Condition

	for _, cond := range conditions {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicEventName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathEventName, pred, columnPathEventName))
			continue
		case traceql.IntrinsicEventTimeSinceStart:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathEventTimeSinceStart, pred, columnPathEventTimeSinceStart))
			continue
		}
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceSpansILSSpanEventAttrs,
		columnPathEventAttrKey, columnPathEventAttrString, columnPathEventAttrInt, columnPathEventAttrDouble, columnPathEventAttrBool, allConditions, selectAll)
	if err != nil {
		return nil, nil, fmt.Errorf("creating event attribute iterator: %w", err)
	}

	if attrIter != nil {
		optional = append(optional, attrIter)
	}

	// This is an optimization for when all of the span conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	return required, optional, nil
}

func createLinkIterators(makeIter makeIterFn, conditions []traceql.Condition, allConditions, selectAll bool) (required, optional []parquetquery.Iterator, err error) {
	var genericConditions []traceql.Condition

	for _, cond := range conditions {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicLinkTraceID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, false)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathLinkTraceID, pred, columnPathLinkTraceID))
			continue

		case traceql.IntrinsicLinkSpanID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, true)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathLinkSpanID, pred, columnPathLinkSpanID))
			continue
		}
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceSpansILSSpanLinkAttrs,
		columnPathLinkAttrKey, columnPathLinkAttrString, columnPathLinkAttrInt, columnPathLinkAttrDouble, columnPathLinkAttrBool, allConditions, selectAll)
	if err != nil {
		return nil, nil, fmt.Errorf("creating link attribute iterator: %w", err)
	}

	if attrIter != nil {
		optional = append(optional, attrIter)
	}

	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	return required, optional, nil
}

func createInstrumentationIterators(makeIter makeIterFn, conditions []traceql.Condition, allConditions, selectAll bool) (required, optional []parquetquery.Iterator, err error) {
	var genericConditions []traceql.Condition
	for _, cond := range conditions {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicInstrumentationName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathInstrumentationName, pred, columnPathInstrumentationName))
			continue
		case traceql.IntrinsicInstrumentationVersion:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, nil, err
			}
			optional = append(optional, makeIter(columnPathInstrumentationVersion, pred, columnPathInstrumentationVersion))
			continue
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	if selectAll {
		for _, entry := range intrinsicColumnLookups {
			if entry.scope != intrinsicScopeInstrumentation {
				continue
			}
			optional = append(optional, makeIter(entry.columnPath, nil, entry.columnPath))
		}
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelInstrumentationScopeAttrs,
		columnPathInstrumentationAttrKey, columnPathInstrumentationAttrString, columnPathInstrumentationAttrInt, columnPathInstrumentationAttrDouble, columnPathInstrumentationAttrBool, allConditions, selectAll)
	if err != nil {
		return nil, nil, fmt.Errorf("creating instrumentation attribute iterator: %w", err)
	}
	if attrIter != nil {
		optional = append(optional, attrIter)
	}

	// This is an optimization for when all of the resource conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, optional...)
		optional = nil
	}

	return required, optional, nil
}

type spanCollector2 struct {
	minAttributes           int
	nestedSetLeftExplicit   bool
	nestedSetRightExplicit  bool
	nestedSetParentExplicit bool

	at    span
	atRes parquetquery.IteratorResult
}

var _ parquetquery.Collector = (*spanCollector2)(nil)

func (c *spanCollector2) String() string {
	return fmt.Sprintf("spanCollector(%d)", c.minAttributes)
}

func NewSpanCollector2() *spanCollector2 {
	// We always return this span.
	c := &spanCollector2{}
	c.atRes.AppendOtherValue(otherEntrySpanKey, &c.at)
	return c
}

func (c *spanCollector2) Reset(rowNumber parquetquery.RowNumber) {
	if rowNumber[DefinitionLevelTrace] != c.at.rowNum[DefinitionLevelTrace] {
		c.at.traceAttrs = c.at.traceAttrs[:0]
	}

	if rowNumber[DefinitionLevelResourceSpans] != c.at.rowNum[DefinitionLevelResourceSpans] {
		c.at.resourceAttrs = c.at.resourceAttrs[:0]
	}
	if rowNumber[DefinitionLevelResourceSpansILSSpan] != c.at.rowNum[DefinitionLevelResourceSpansILSSpan] {
		c.at.spanAttrs = c.at.spanAttrs[:0]
		c.at.eventAttrs = c.at.eventAttrs[:0]
		c.at.linkAttrs = c.at.linkAttrs[:0]
	}
	c.at.rowNum = rowNumber
	c.atRes.RowNumber = rowNumber
}

func (c *spanCollector2) Collect(res *parquetquery.IteratorResult) {
	sp := &c.at

	c.at.rowNum = res.RowNumber

	for _, e := range res.OtherEntries {
		switch v := e.Value.(type) {
		case *span:
			// Copy data from first pass span to this one.
			sp.rowNum = v.rowNum
			sp.id = v.id
			sp.startTimeUnixNanos = v.startTimeUnixNanos
			sp.durationNanos = v.durationNanos
			sp.spanAttrs = append(sp.spanAttrs, v.spanAttrs...)
			sp.traceAttrs = append(sp.traceAttrs, v.traceAttrs...)
			sp.resourceAttrs = append(sp.resourceAttrs, v.resourceAttrs...)
			sp.eventAttrs = append(sp.eventAttrs, v.eventAttrs...)
			sp.linkAttrs = append(sp.linkAttrs, v.linkAttrs...)
			sp.nestedSetParent = v.nestedSetParent
			sp.nestedSetLeft = v.nestedSetLeft
			sp.nestedSetRight = v.nestedSetRight
		case traceql.Static:
			switch {
			case res.RowNumber[DefinitionLevelResourceSpansILSSpanAttrs] >= 0:
				sp.addSpanAttr(newSpanAttr(e.Key), v)
			case res.RowNumber[DefinitionLevelResourceSpans] >= 0:
				sp.resourceAttrs = append(sp.resourceAttrs, attrVal{traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, e.Key), v})
			}
		case *event:
			sp.setEventAttrs(v.attrs)
			putEvent(v)
		case *link:
			sp.setLinkAttrs(v.attrs)
			putLink(v)
		default:
			panic("unhandled other entry value type: " + "key:" + e.Key)
		}
	}

	var durationNanos uint64

	// Merge all individual columns into the span
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathTraceID:
			sp.traceAttrs = append(sp.traceAttrs, attrVal{traceql.IntrinsicTraceIDAttribute, traceql.NewStaticString(util.TraceIDToHexString(kv.Value.ByteArray()))})
		case columnPathSpanID:
			sp.id = kv.Value.ByteArray()
			sp.addSpanAttr(traceql.IntrinsicSpanIDAttribute, traceql.NewStaticString(util.SpanIDToHexString(kv.Value.ByteArray())))
		case columnPathSpanParentSpanID:
			sp.addSpanAttr(traceql.IntrinsicParentIDAttribute, traceql.NewStaticString(util.SpanIDToHexString(kv.Value.ByteArray())))
		case columnPathSpanStartTime:
			sp.startTimeUnixNanos = kv.Value.Uint64()
		case columnPathSpanStartRounded15:
			sp.startTimeUnixNanos = intervalMapper15Seconds.TimestampOf(int(kv.Value.Int64()))
		case columnPathSpanStartRounded60:
			sp.startTimeUnixNanos = intervalMapper60Seconds.TimestampOf(int(kv.Value.Int64()))
		case columnPathSpanStartRounded300:
			sp.startTimeUnixNanos = intervalMapper300Seconds.TimestampOf(int(kv.Value.Int64()))
		case columnPathSpanStartRounded3600:
			sp.startTimeUnixNanos = intervalMapper3600Seconds.TimestampOf(int(kv.Value.Int64()))
		case columnPathSpanDuration:
			durationNanos = kv.Value.Uint64()
			sp.durationNanos = durationNanos
			sp.addSpanAttr(traceql.IntrinsicDurationAttribute, traceql.NewStaticDuration(time.Duration(durationNanos)))
		case columnPathSpanName:
			sp.addSpanAttr(traceql.IntrinsicNameAttribute, traceql.NewStaticString(unsafeToString(kv.Value.Bytes())))
		case columnPathSpanStatusCode:
			sp.addSpanAttr(traceql.IntrinsicStatusAttribute, traceql.NewStaticStatus(otlpStatusToTraceqlStatus(kv.Value.Uint64())))
		case columnPathSpanStatusMessage:
			sp.addSpanAttr(traceql.IntrinsicStatusMessageAttribute, traceql.NewStaticString(unsafeToString(kv.Value.Bytes())))
		case columnPathSpanKind:
			sp.addSpanAttr(traceql.IntrinsicKindAttribute, traceql.NewStaticKind(otlpKindToTraceqlKind(kv.Value.Uint64())))
		case columnPathSpanParentID:
			sp.nestedSetParent = kv.Value.Int32()
			if c.nestedSetParentExplicit {
				sp.addSpanAttr(traceql.IntrinsicNestedSetParentAttribute, traceql.NewStaticInt(int(kv.Value.Int32())))
			}
		case columnPathSpanNestedSetLeft:
			sp.nestedSetLeft = kv.Value.Int32()
			if c.nestedSetLeftExplicit {
				sp.addSpanAttr(traceql.IntrinsicNestedSetLeftAttribute, traceql.NewStaticInt(int(kv.Value.Int32())))
			}
		case columnPathSpanNestedSetRight:
			sp.nestedSetRight = kv.Value.Int32()
			if c.nestedSetRightExplicit {
				sp.addSpanAttr(traceql.IntrinsicNestedSetRightAttribute, traceql.NewStaticInt(int(kv.Value.Int32())))
			}
		default:
			if res.RowNumber[DefinitionLevelResourceSpansILSSpan] == -1 {
				// Resource level attribute
				switch kv.Value.Kind() {
				case parquet.Boolean:
					sp.resourceAttrs = append(sp.resourceAttrs, attrVal{newResAttr(kv.Key), traceql.NewStaticBool(kv.Value.Boolean())})
				case parquet.Int32, parquet.Int64:
					sp.resourceAttrs = append(sp.resourceAttrs, attrVal{newResAttr(kv.Key), traceql.NewStaticInt(int(kv.Value.Int64()))})
				case parquet.Float:
					sp.resourceAttrs = append(sp.resourceAttrs, attrVal{newResAttr(kv.Key), traceql.NewStaticFloat(kv.Value.Double())})
				case parquet.ByteArray:
					sp.resourceAttrs = append(sp.resourceAttrs, attrVal{newResAttr(kv.Key), traceql.NewStaticString(unsafeToString(kv.Value.Bytes()))})
				}
			} else {
				// TODO - This exists for span-level dedicated columns like http.status_code
				// Are nils possible here?
				switch kv.Value.Kind() {
				case parquet.Boolean:
					sp.addSpanAttr(newSpanAttr(kv.Key), traceql.NewStaticBool(kv.Value.Boolean()))
				case parquet.Int32, parquet.Int64:
					sp.addSpanAttr(newSpanAttr(kv.Key), traceql.NewStaticInt(int(kv.Value.Int64())))
				case parquet.Float:
					sp.addSpanAttr(newSpanAttr(kv.Key), traceql.NewStaticFloat(kv.Value.Double()))
				case parquet.ByteArray:
					sp.addSpanAttr(newSpanAttr(kv.Key), traceql.NewStaticString(unsafeToString(kv.Value.Bytes())))
				}
			}
		}
	}
}

func (c *spanCollector2) Result() *parquetquery.IteratorResult {
	if c.minAttributes == 0 {
		return &c.atRes
	}

	if c.at.attributesMatched() >= c.minAttributes {
		return &c.atRes
	}

	return nil
}
