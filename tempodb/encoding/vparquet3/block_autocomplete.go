package vparquet3

import (
	"context"
	"fmt"
	"math"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
	"github.com/pkg/errors"
)

func (b *backendBlock) SuperFetch(ctx context.Context, req traceql.AutocompleteRequest, cb traceql.AutocompleteCallback, opts common.SearchOptions) error {
	err := checkConditions(req.Conditions)
	if err != nil {
		return errors.Wrap(err, "conditions invalid")
	}

	pf, _, err := b.openForSearch(ctx, opts)
	if err != nil {
		return err
	}

	iter, err := autocompleteIter(ctx, req, cb, pf, opts, b.meta.DedicatedColumns)
	if err != nil {
		return errors.Wrap(err, "creating fetch iter")
	}

	// TODO: The iter shouldn't be exhausted here, it should be returned to the caller
	for {
		// Exhaust the iterator
		fmt.Println("exhausting iter in backendBlock")
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if res == nil {
			break
		}
	}
	fmt.Println("done exhausting iter in backendBlock")

	return nil
}

// autocompleteIter creates an iterator that will collect values for a given attribute/tag.
func autocompleteIter(ctx context.Context, req traceql.AutocompleteRequest, cb traceql.AutocompleteCallback, pf *parquet.File, opts common.SearchOptions, dc backend.DedicatedColumns) (parquetquery.Iterator, error) {
	iter, err := createDistinctIterator(ctx, nil, req.Conditions, true, 0, math.MaxInt64, cb, req.TagName, pf, opts, dc)
	if err != nil {
		return nil, fmt.Errorf("error creating iterator: %w", err)
	}

	fmt.Println("request", req.Conditions)
	fmt.Println(iter)
	fmt.Println("------------------")

	return iter, nil
}

func createDistinctIterator(
	ctx context.Context,
	primaryIter parquetquery.Iterator,
	conds []traceql.Condition,
	allConditions bool,
	start, end uint64,
	cb traceql.AutocompleteCallback,
	key string,
	pf *parquet.File,
	opts common.SearchOptions,
	dc backend.DedicatedColumns,
) (parquetquery.Iterator, error) {
	// Categorize conditions into span-level or resource-level
	var (
		mingledConditions  bool
		spanConditions     []traceql.Condition
		resourceConditions []traceql.Condition
		traceConditions    []traceql.Condition
	)
	for _, cond := range conds {
		// If no-scoped intrinsic then assign default scope
		scope := cond.Attribute.Scope
		if cond.Attribute.Scope == traceql.AttributeScopeNone {
			if lookup, ok := intrinsicColumnLookups[cond.Attribute.Intrinsic]; ok {
				scope = lookup.scope
			}
		}

		switch scope {

		case traceql.AttributeScopeNone:
			mingledConditions = true
			spanConditions = append(spanConditions, cond)
			resourceConditions = append(resourceConditions, cond)
			continue

		case traceql.AttributeScopeSpan, intrinsicScopeSpan:
			spanConditions = append(spanConditions, cond)
			continue

		case traceql.AttributeScopeResource:
			resourceConditions = append(resourceConditions, cond)
			continue

		case intrinsicScopeTrace:
			traceConditions = append(traceConditions, cond)
			continue

		default:
			return nil, fmt.Errorf("unsupported traceql scope: %s", cond.Attribute)
		}
	}

	rgs := rowGroupsFromFile(pf, opts)
	makeIter := makeIterFunc(ctx, rgs, pf)

	// Global state
	// Span-filtering behavior changes depending on the resource-filtering in effect,
	// and vice-versa.  For example consider the query { span.a=1 }.  If no spans have a=1
	// then it generate the empty spanset.
	// However once we add a resource condition: { span.a=1 || resource.b=2 }, now the span
	// filtering must return all spans, even if no spans have a=1, because they might be
	// matched upstream to a resource.
	// TODO - After introducing AllConditions it seems like some of this logic overlaps.
	//        Determine if it can be generalized or simplified.
	var (
		// If there are only span conditions, then don't return a span upstream
		// unless it matches at least 1 span-level condition.
		spanRequireAtLeastOneMatch = len(spanConditions) > 0 && len(resourceConditions) == 0 && len(traceConditions) == 0

		// If there are only resource conditions, then don't return a resource upstream
		// unless it matches at least 1 resource-level condition.
		batchRequireAtLeastOneMatch = len(spanConditions) == 0 && len(resourceConditions) > 0 && len(traceConditions) == 0

		// Don't return the final spanset upstream unless it matched at least 1 condition
		// anywhere, except in the case of the empty query: {}
		batchRequireAtLeastOneMatchOverall = len(conds) > 0 && len(traceConditions) == 0 && len(traceConditions) == 0
	)

	// Optimization for queries like {resource.x... && span.y ...}
	// Requires no mingled scopes like .foo=x, which could be satisfied
	// one either resource or span.
	allConditions = allConditions && !mingledConditions

	var (
		spanIter, resourceIter, traceIter parquetquery.Iterator
		err                               error
	)

	if len(spanConditions) > 0 {
		spanIter, err = createDistinctSpanIterator(makeIter, primaryIter, spanConditions, cb, key, spanRequireAtLeastOneMatch, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating span iterator")
		}
	}

	if len(resourceConditions) > 0 {
		resourceIter, err = createDistinctResourceIterator(makeIter, spanIter, resourceConditions, cb, key, batchRequireAtLeastOneMatch, batchRequireAtLeastOneMatchOverall, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating resource iterator")
		}
	}

	if len(traceConditions) > 0 {
		traceIter, err = createDistinctTraceIterator(makeIter, resourceIter, traceConditions, cb, key, start, end, allConditions)
		if err != nil {
			return nil, errors.Wrap(err, "creating trace iterator")
		}
		return traceIter, nil
	}

	if traceIter != nil {
		return traceIter, nil
	} else if resourceIter != nil {
		return resourceIter, nil
	} else if spanIter != nil {
		return spanIter, nil
	} else {
		return nil, nil
	}
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createDistinctSpanIterator(makeIter makeIterFn, primaryIter parquetquery.Iterator, conditions []traceql.Condition, cb traceql.AutocompleteCallback, key string, requireAtLeastOneMatch, allConditions bool, dedicatedColumns backend.DedicatedColumns) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
		columnMapping     = dedicatedColumnsToColumnMapping(dedicatedColumns, backend.DedicatedColumnScopeSpan)
	)

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	selectColumnIfNotAlready := func(path string) {
		if columnPredicates[path] == nil {
			addPredicate(path, nil)
			columnSelectAs[path] = path
		}
	}

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicSpanID:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanID, pred)
			columnSelectAs[columnPathSpanID] = columnPathSpanID
			continue

		case traceql.IntrinsicSpanStartTime:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanStartTime, pred)
			columnSelectAs[columnPathSpanStartTime] = columnPathSpanStartTime
			continue

		case traceql.IntrinsicName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanName, pred)
			columnSelectAs[columnPathSpanName] = columnPathSpanName
			continue

		case traceql.IntrinsicKind:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanKind, pred)
			columnSelectAs[columnPathSpanKind] = columnPathSpanKind
			continue

		case traceql.IntrinsicDuration:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanDuration, pred)
			columnSelectAs[columnPathSpanDuration] = columnPathSpanDuration
			continue

		case traceql.IntrinsicStatus:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanStatusCode, pred)
			columnSelectAs[columnPathSpanStatusCode] = columnPathSpanStatusCode
			continue
		case traceql.IntrinsicStatusMessage:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanStatusMessage, pred)
			columnSelectAs[columnPathSpanStatusMessage] = columnPathSpanStatusMessage
			continue

		case traceql.IntrinsicStructuralDescendant:
			selectColumnIfNotAlready(columnPathSpanNestedSetLeft)
			selectColumnIfNotAlready(columnPathSpanNestedSetRight)

		case traceql.IntrinsicStructuralChild:
			selectColumnIfNotAlready(columnPathSpanNestedSetLeft)
			selectColumnIfNotAlready(columnPathSpanParentID)

		case traceql.IntrinsicStructuralSibling:
			selectColumnIfNotAlready(columnPathSpanParentID)
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
					return nil, errors.Wrap(err, "creating predicate")
				}
				addPredicate(c.ColumnPath, pred)
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, genericConditions, cb, key, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
	}

	var required []parquetquery.Iterator
	if primaryIter != nil {
		required = []parquetquery.Iterator{primaryIter}
	}

	// This is an optimization for when all of the span conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, iters...)
		iters = nil
	}

	// This is an optimization for cases when allConditions is false, and
	// only span conditions are present, and we require at least one of them to match.
	// Wrap up the individual conditions with a union and move it into the required list.
	// This skips over static columns like ID that are omnipresent. This is also only
	// possible when there isn't a duration filter because it's computed from start/end.
	if requireAtLeastOneMatch && len(iters) > 0 {
		required = append(required, parquetquery.NewUnionIterator(DefinitionLevelResourceSpansILSSpan, iters, nil))
		iters = nil
	}

	// if there are no direct conditions imposed on the span/span attributes level we are purposefully going to request the "Kind" column
	//  b/c it is extremely cheap to retrieve. retrieving matching spans in this case will allow aggregates such as "count" to be computed
	//  how do we know to pull duration for things like | avg(duration) > 1s? look at avg(span.http.status_code) it pushes a column request down here
	//  the entire engine is built around spans. we have to return at least one entry for every span to the layers above for things to work
	// TODO: note that if the query is { kind = client } the fetch layer will actually create two iterators over the kind column. this is evidence
	//  this spaniterator code could be tightened up
	// Also note that this breaks optimizations related to requireAtLeastOneMatch and requireAtLeastOneMatchOverall b/c it will add a kind attribute
	//  to the span attributes map in spanCollector
	if len(required) == 0 {
		required = []parquetquery.Iterator{makeIter(columnPathSpanKind, nil, "")}
	}

	spanCol := &distinctSpanCollector{
		cb:  cb,
		key: key,
	}

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, spanCol), nil
}

func createDistinctAttributeIterator(makeIter makeIterFn, conditions []traceql.Condition, cb traceql.AutocompleteCallback, key string,
	definitionLevel int,
	keyPath, strPath, intPath, floatPath, boolPath string,
	allConditions bool,
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

		// if all conditions must be true we can use a simple join iterator to test the values one column at a time.
		// len(valueIters) must be 1 to handle queries like `{ span.foo = "x" && span.bar > 1}`
		if allConditions && len(valueIters) == 1 {
			iters := append([]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")}, valueIters...)
			return parquetquery.NewJoinIterator(definitionLevel,
				iters,
				&distinctAttrCollector{
					cb:  cb,
					key: key,
				}), nil
		}

		return parquetquery.NewLeftJoinIterator(definitionLevel,
			[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
			valueIters,
			&distinctAttrCollector{
				cb:  cb,
				key: key,
			}), nil
	}

	return nil, nil
}

func createDistinctResourceIterator(makeIter makeIterFn, spanIterator parquetquery.Iterator, conditions []traceql.Condition, cb traceql.AutocompleteCallback, key string, requireAtLeastOneMatch, _, allConditions bool, dedicatedColumns backend.DedicatedColumns) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		iters             = []parquetquery.Iterator{}
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
					return nil, errors.Wrap(err, "creating predicate")
				}
				iters = append(iters, makeIter(entry.columnPath, pred, cond.Attribute.Name))
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
					return nil, errors.Wrap(err, "creating predicate")
				}
				addPredicate(c.ColumnPath, pred)
				columnSelectAs[c.ColumnPath] = cond.Attribute.Name
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, genericConditions, cb, key, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	batchCol := &distinctBatchCollector{
		cb:  cb,
		key: key,
	}

	var required []parquetquery.Iterator

	// This is an optimization for when all of the resource conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(required, iters...)
		iters = nil
	}

	// This is an optimization for cases when only resource conditions are
	// present and we require at least one of them to match.  Wrap
	// up the individual conditions with a union and move it into the
	// required list.
	if requireAtLeastOneMatch && len(iters) > 0 {
		required = append(required, parquetquery.NewUnionIterator(DefinitionLevelResourceSpans, iters, nil))
		iters = nil
	}

	// Put span iterator last so it is only read when
	// the resource conditions are met.
	required = append(required, spanIterator)

	// Left join here means the span iterator + 1 are required,
	// and all other resource conditions are optional. Whatever matches
	// is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpans,
		required, iters, batchCol), nil
}

func createDistinctTraceIterator(makeIter makeIterFn, resourceIter parquetquery.Iterator, conds []traceql.Condition, cb traceql.AutocompleteCallback, key string, start, end uint64, allConditions bool) (parquetquery.Iterator, error) {
	traceIters := make([]parquetquery.Iterator, 0, 3)

	var err error

	// add conditional iterators first. this way if someone searches for { traceDuration > 1s && span.foo = "bar"} the query will
	// be sped up by searching for traceDuration first. note that we can only set the predicates if all conditions is true.
	// otherwise we just pass the info up to the engine to make a choice
	for _, cond := range conds {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicTraceID:
			traceIters = append(traceIters, makeIter(columnPathTraceID, nil, columnPathTraceID))
		case traceql.IntrinsicTraceDuration:
			var pred parquetquery.Predicate
			if allConditions {
				pred, err = createIntPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, err
				}
			}
			traceIters = append(traceIters, makeIter(columnPathDurationNanos, pred, columnPathDurationNanos))
		case traceql.IntrinsicTraceStartTime:
			if start == 0 && end == 0 {
				traceIters = append(traceIters, makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano))
			}
		case traceql.IntrinsicTraceRootSpan:
			var pred parquetquery.Predicate
			if allConditions {
				pred, err = createStringPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, err
				}
			}
			traceIters = append(traceIters, makeIter(columnPathRootSpanName, pred, columnPathRootSpanName))
		case traceql.IntrinsicTraceRootService:
			var pred parquetquery.Predicate
			if allConditions {
				pred, err = createStringPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, err
				}
			}
			traceIters = append(traceIters, makeIter(columnPathRootServiceName, pred, columnPathRootServiceName))
		}
	}

	// order is interesting here. would it be more efficient to grab the span/resource conditions first
	// or the time range filtering first?
	traceIters = append(traceIters, resourceIter)

	// evaluate time range
	// Time range filtering?
	if start > 0 && end > 0 {
		// Here's how we detect the span overlaps the time window:
		// Span start <= req.End
		// Span end >= req.Start
		var startFilter, endFilter parquetquery.Predicate
		startFilter = parquetquery.NewIntBetweenPredicate(0, int64(end))
		endFilter = parquetquery.NewIntBetweenPredicate(int64(start), math.MaxInt64)

		traceIters = append(traceIters, makeIter(columnPathStartTimeUnixNano, startFilter, columnPathStartTimeUnixNano))
		traceIters = append(traceIters, makeIter(columnPathEndTimeUnixNano, endFilter, columnPathEndTimeUnixNano))
	}

	// Final trace iterator
	// Join iterator means it requires matching resources to have been found
	// TraceCollor adds trace-level data to the spansets
	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &distinctTraceCollector{
		cb:  cb,
		key: key,
	}), nil
}

var _ parquetquery.GroupPredicate = (*distinctAttrCollector)(nil)

type distinctAttrCollector struct {
	// TODO: key is passed to every collector, can it be cleaned up?
	key string
	cb  traceql.AutocompleteCallback
}

func (d *distinctAttrCollector) String() string {
	return "distinctAttrCollector"
}

func (d *distinctAttrCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	tv := tempopb.TagValue{}

	for _, e := range result.Entries {
		// Ignore nulls, this leaves val as the remaining found value,
		// or nil if the key was found but no matching values
		if e.Value.Kind() < 0 {
			continue
		}
		switch e.Key {
		case "key":
			if d.key != e.Value.String() {
				return false
			}
		case "string", "int", "float", "bool":
			tv = pqValueToTagValue(e.Value)
		}
	}

	d.cb(tv) // TODO: What should we return here?

	return false
}

var _ parquetquery.GroupPredicate = (*distinctSpanCollector)(nil)

type distinctSpanCollector struct {
	key string
	cb  traceql.AutocompleteCallback
}

func (d distinctSpanCollector) String() string {
	return "distinctSpanCollector"
}

func (d distinctSpanCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	// Merge all individual columns into the span
	for _, e := range result.Entries {
		if key, tv := extractTagValue(e); key == d.key {
			d.cb(tv) // TODO: What should we return here?
		}
	}
	return false // TODO: What should we return here?
}

type entry struct {
	Key   string
	Value parquet.Value
}

func extractTagValue(e entry) (string, tempopb.TagValue) {
	switch e.Key {
	case columnPathSpanID,
		columnPathSpanParentID,
		columnPathSpanNestedSetLeft,
		columnPathSpanNestedSetRight:
		return "", tempopb.TagValue{}
	case columnPathSpanStartTime:
		return traceql.IntrinsicSpanStartTime.String(), pqValueToTagValue(e.Value)
	case columnPathSpanDuration:
		return traceql.IntrinsicDuration.String(), pqValueToTagValue(e.Value)
	case columnPathSpanName:
		return traceql.IntrinsicName.String(), pqValueToTagValue(e.Value)
	case columnPathSpanStatusCode:
		// Map OTLP status code back to TraceQL enum.
		// For other values, use the raw integer.
		var status traceql.Status
		switch e.Value.Uint64() {
		case uint64(v1.Status_STATUS_CODE_UNSET):
			status = traceql.StatusUnset
		case uint64(v1.Status_STATUS_CODE_OK):
			status = traceql.StatusOk
		case uint64(v1.Status_STATUS_CODE_ERROR):
			status = traceql.StatusError
		default:
			status = traceql.Status(e.Value.Uint64())
		}
		return traceql.IntrinsicStatus.String(), tempopb.TagValue{Type: "duration", Value: status.String()}
	case columnPathSpanStatusMessage:
		return traceql.IntrinsicStatusMessage.String(), tempopb.TagValue{Type: "keyword", Value: e.Value.String()}
	case columnPathSpanKind:
		var kind traceql.Kind
		switch e.Value.Uint64() {
		case uint64(v1.Span_SPAN_KIND_UNSPECIFIED):
			kind = traceql.KindUnspecified
		case uint64(v1.Span_SPAN_KIND_INTERNAL):
			kind = traceql.KindInternal
		case uint64(v1.Span_SPAN_KIND_SERVER):
			kind = traceql.KindServer
		case uint64(v1.Span_SPAN_KIND_CLIENT):
			kind = traceql.KindClient
		case uint64(v1.Span_SPAN_KIND_PRODUCER):
			kind = traceql.KindProducer
		case uint64(v1.Span_SPAN_KIND_CONSUMER):
			kind = traceql.KindConsumer
		default:
			kind = traceql.Kind(e.Value.Uint64())
		}
		return traceql.IntrinsicKind.String(), tempopb.TagValue{Type: "int", Value: kind.String()}
	default:
		// TODO - This exists for span-level dedicated columns like http.status_code
		// Are nils possible here?
		return e.Key, pqValueToTagValue(e.Value)
	}
}

var _ parquetquery.GroupPredicate = (*distinctBatchCollector)(nil)

type distinctBatchCollector struct {
	key string
	cb  traceql.AutocompleteCallback
}

func (d *distinctBatchCollector) String() string {
	return "distinctBatchCollector"
}

func (d *distinctBatchCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	// Gather Attributes from dedicated resource-level columns
	for _, e := range result.Entries {
		if e.Key != d.key {
			continue
		}
		switch e.Value.Kind() {
		case parquet.Int64, parquet.ByteArray:
			d.cb(pqValueToTagValue(e.Value))
		}
	}
	return false
}

var _ parquetquery.GroupPredicate = (*distinctTraceCollector)(nil)

type distinctTraceCollector struct {
	key string
	cb  traceql.AutocompleteCallback
}

func (d *distinctTraceCollector) String() string {
	return "distinctTraceCollector"
}

func (d *distinctTraceCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		switch e.Key {
		case columnPathTraceID:
		case columnPathStartTimeUnixNano:
			if traceql.IntrinsicTraceStartTime.String() == d.key {
				d.cb(pqValueToTagValue(e.Value))
			}
		case columnPathDurationNanos:
			if traceql.IntrinsicTraceDuration.String() == d.key {
				d.cb(pqValueToTagValue(e.Value))
			}
		case columnPathRootSpanName:
			if traceql.IntrinsicTraceRootSpan.String() == d.key {
				d.cb(pqValueToTagValue(e.Value))
			}
		case columnPathRootServiceName:
			if traceql.IntrinsicTraceRootService.String() == d.key {
				d.cb(pqValueToTagValue(e.Value))
			}
		}
	}
	return false
}

func pqValueToTagValue(v parquet.Value) tempopb.TagValue {
	switch v.Kind() {
	case parquet.Boolean:
		return tempopb.TagValue{Type: "bool", Value: v.String()}
	case parquet.Int32, parquet.Int64:
		return tempopb.TagValue{Type: "int", Value: v.String()}
	case parquet.Float:
		return tempopb.TagValue{Type: "float", Value: v.String()}
	case parquet.ByteArray:
		return tempopb.TagValue{Type: "string", Value: v.String()}
	default:
		return tempopb.TagValue{}
	}
}
