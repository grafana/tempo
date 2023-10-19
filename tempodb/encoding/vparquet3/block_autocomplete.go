package vparquet3

import (
	"context"
	"fmt"
	"math"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
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

	iter, err := autocompleteIter(ctx, req, pf, opts, b.meta.DedicatedColumns)
	if err != nil {
		return errors.Wrap(err, "creating fetch iter")
	}

	// TODO: The iter shouldn't be exhausted here, it should be returned to the caller
	for {
		// Exhaust the iterator
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if res == nil {
			break
		}
		for _, e := range res.Entries {
			if e.Key == req.TagName {
				cb(pqValueToTagValue(e.Value))
			}
		}
	}

	return nil
}

// autocompleteIter creates an iterator that will collect values for a given attribute/tag.
func autocompleteIter(ctx context.Context, req traceql.AutocompleteRequest, pf *parquet.File, opts common.SearchOptions, dc backend.DedicatedColumns) (parquetquery.Iterator, error) {
	iter, err := createDistinctIterator(ctx, nil, req.Conditions, true, 0, math.MaxInt64, req.TagName, pf, opts, dc)
	if err != nil {
		return nil, fmt.Errorf("error creating iterator: %w", err)
	}

	_ = level.Info(log.Logger).Log("msg", "created iterator", "conditions", fmt.Sprintf("%+v", req.Conditions))
	_ = level.Info(log.Logger).Log("iter", iter.String())

	return iter, nil
}

func createDistinctIterator(
	ctx context.Context,
	primaryIter parquetquery.Iterator,
	conds []traceql.Condition,
	allConditions bool,
	start, end uint64,
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
	keep := func(s string) bool { return key == s }

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

	fmt.Println("spanConditions", len(spanConditions), "resourceConditions", len(resourceConditions), "traceConditions", len(traceConditions))

	if len(spanConditions) > 0 {
		spanIter, err = createDistinctSpanIterator(makeIter, keep, primaryIter, spanConditions, spanRequireAtLeastOneMatch, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating span iterator")
		}
	}

	if len(resourceConditions) > 0 {
		resourceIter, err = createDistinctResourceIterator(makeIter, keep, spanIter, resourceConditions, batchRequireAtLeastOneMatch, batchRequireAtLeastOneMatchOverall, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating resource iterator")
		}
	}

	if len(traceConditions) > 0 {
		traceIter, err = createDistinctTraceIterator(makeIter, keep, resourceIter, traceConditions, start, end, allConditions)
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
		return nil, fmt.Errorf("no conditions")
	}
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createDistinctSpanIterator(makeIter makeIterFn, keep keepFn, primaryIter parquetquery.Iterator, conditions []traceql.Condition, _, allConditions bool, dedicatedColumns backend.DedicatedColumns) (parquetquery.Iterator, error) {
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

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, keep, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
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

	if len(required) == 0 {
		return attrIter, nil
	}

	spanCol := &distinctSpanCollector{keep: keep}

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, spanCol), nil
}

func createDistinctAttributeIterator(makeIter makeIterFn, keep keepFn, conditions []traceql.Condition,
	definitionLevel int,
	keyPath, strPath, intPath, floatPath, boolPath string,
	allConditions bool,
) (parquetquery.Iterator, error) {
	var (
		attrKeys                                               []string
		attrStringPreds, attrIntPreds, attrFltPreds, boolPreds []parquetquery.Predicate
		allIters                                               []parquetquery.Iterator
	)
	for _, cond := range conditions {

		if cond.Op == traceql.OpNone {
			// This means we have to scan all values, we don't know what type
			// to expect
			attrKeys = append(attrKeys, cond.Attribute.Name)
			attrStringPreds = append(attrStringPreds, nil)
			attrIntPreds = append(attrIntPreds, nil)
			attrFltPreds = append(attrFltPreds, nil)
			boolPreds = append(boolPreds, nil)
			continue
		}

		var keyIter, valIter parquetquery.Iterator

		switch cond.Operands[0].Type {
		case traceql.TypeString:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), "key")
			valIter = makeIter(strPath, pred, "string")

		case traceql.TypeInt:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), "key")
			valIter = makeIter(intPath, pred, "int")

		case traceql.TypeFloat:
			pred, err := createFloatPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), "key")
			valIter = makeIter(floatPath, pred, "float")

		case traceql.TypeBoolean:
			pred, err := createBoolPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), "key")
			valIter = makeIter(boolPath, pred, "bool")
		}

		allIters = append(allIters, parquetquery.NewJoinIterator(definitionLevel, []parquetquery.Iterator{keyIter, valIter}, &distinctAttrCollector{keep: keep}))
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

	// TODO: Super hacky, clean up!!
	if len(allIters) > 0 {
		if len(attrKeys) > 0 {
			allIters = append(allIters, makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key"))
		}
		return parquetquery.NewLeftJoinIterator(definitionLevel, allIters, valueIters, &distinctAttrCollector{keep: keep}), nil
	}

	if len(valueIters) > 0 {
		// LeftJoin means only look at rows where the key is what we want.
		// Bring in any of the typed values as needed.

		// if all conditions must be true we can use a simple join iterator to test the values one column at a time.
		// len(valueIters) must be 1 to handle queries like `{ span.foo = "x" && span.bar > 1}`
		if allConditions && len(valueIters) == 1 {
			iters := append([]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")}, valueIters...)
			return parquetquery.NewJoinIterator(
				definitionLevel,
				iters,
				&distinctAttrCollector{keep: keep},
			), nil
		}

		return parquetquery.NewLeftJoinIterator(
			definitionLevel,
			[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
			valueIters,
			&distinctAttrCollector{keep: keep},
		), nil
	}

	return nil, nil
}

func createDistinctResourceIterator(makeIter makeIterFn, keep keepFn, spanIterator parquetquery.Iterator, conditions []traceql.Condition, requireAtLeastOneMatch, _, allConditions bool, dedicatedColumns backend.DedicatedColumns) (parquetquery.Iterator, error) {
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

	attrIter, err := createDistinctAttributeIterator(makeIter, keep, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	batchCol := &distinctBatchCollector{keep: keep}

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
	if spanIterator != nil {
		required = append(required, spanIterator)
	}

	// Left join here means the span iterator + 1 are required,
	// and all other resource conditions are optional. Whatever matches
	// is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpans, required, iters, batchCol), nil
}

func createDistinctTraceIterator(makeIter makeIterFn, keep keepFn, resourceIter parquetquery.Iterator, conds []traceql.Condition, start, end uint64, allConditions bool) (parquetquery.Iterator, error) {
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
		keep: keep,
	}), nil
}

type keepFn func(string) bool

var _ parquetquery.GroupPredicate = (*distinctAttrCollector)(nil)

type distinctAttrCollector struct {
	// TODO: key is passed to every collector, can it be cleaned up?
	keep keepFn
}

func (d *distinctAttrCollector) String() string {
	return "distinctAttrCollector"
}

func (d *distinctAttrCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	var key string
	var val parquet.Value

	for _, e := range result.Entries {
		// Ignore nulls, this leaves val as the remaining found value,
		// or nil if the key was found but no matching values
		if e.Value.Kind() < 0 {
			continue
		}

		switch e.Key {
		case "key":
			key = e.Value.String()
		case "string":
			val = e.Value
		case "int":
			val = e.Value
		case "float":
			val = e.Value
		case "bool":
			val = e.Value
		}
	}

	result.Entries = result.Entries[:0]
	if d.keep(key) {
		result.AppendValue(key, val)
	}

	return true
}

var _ parquetquery.GroupPredicate = (*distinctSpanCollector)(nil)

type distinctSpanCollector struct {
	keep keepFn
}

func (d distinctSpanCollector) String() string {
	return "distinctSpanCollector"
}

func (d distinctSpanCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		if key, v := extractTagValue(e); d.keep(key) {
			result.AppendValue(key, v)
		}
	}
	return true // TODO: What should we return here?
}

type entry struct {
	Key   string
	Value parquet.Value
}

func extractTagValue(e entry) (string, parquet.Value) {
	switch e.Key {
	case columnPathSpanID,
		columnPathSpanParentID,
		columnPathSpanNestedSetLeft,
		columnPathSpanNestedSetRight:
		return "", parquet.Value{}
	case columnPathSpanStartTime:
		return traceql.IntrinsicSpanStartTime.String(), e.Value
	case columnPathSpanDuration:
		return traceql.IntrinsicDuration.String(), e.Value
	case columnPathSpanName:
		return traceql.IntrinsicName.String(), e.Value
	case columnPathSpanStatusCode:
		// TODO: Translate to TraceQL status code (string)
		return traceql.IntrinsicStatus.String(), e.Value
	case columnPathSpanStatusMessage:
		return traceql.IntrinsicStatusMessage.String(), e.Value
	case columnPathSpanKind:
		// TODO: Translate to TraceQL kind (string)
		return traceql.IntrinsicKind.String(), e.Value
	default:
		// TODO - This exists for span-level dedicated columns like http.status_code
		// Are nils possible here?
		return e.Key, e.Value
	}
}

var _ parquetquery.GroupPredicate = (*distinctBatchCollector)(nil)

type distinctBatchCollector struct {
	keep keepFn
}

func (d *distinctBatchCollector) String() string {
	return "distinctBatchCollector"
}

func (d *distinctBatchCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	// Gather Attributes from dedicated resource-level columns
	for _, e := range result.Entries {
		if !d.keep(e.Key) {
			continue
		}
		switch e.Value.Kind() {
		case parquet.Int64, parquet.ByteArray:
			result.AppendValue(e.Key, e.Value)
		}
	}
	// TODO: Clean up result.Entries?
	return true
}

var _ parquetquery.GroupPredicate = (*distinctTraceCollector)(nil)

type distinctTraceCollector struct {
	keep keepFn
}

func (d *distinctTraceCollector) String() string {
	return "distinctTraceCollector"
}

func (d *distinctTraceCollector) KeepGroup(_ *parquetquery.IteratorResult) bool {
	//for _, e := range result.Entries {
	//	switch e.Key {
	//	case columnPathTraceID:
	//	case columnPathStartTimeUnixNano:
	//		if traceql.IntrinsicTraceStartTime.String() == d.key {
	//			d.cb(pqValueToTagValue(e.Value))
	//		}
	//	case columnPathDurationNanos:
	//		if traceql.IntrinsicTraceDuration.String() == d.key {
	//			d.cb(pqValueToTagValue(e.Value))
	//		}
	//	case columnPathRootSpanName:
	//		if traceql.IntrinsicTraceRootSpan.String() == d.key {
	//			d.cb(pqValueToTagValue(e.Value))
	//		}
	//	case columnPathRootServiceName:
	//		if traceql.IntrinsicTraceRootService.String() == d.key {
	//			d.cb(pqValueToTagValue(e.Value))
	//		}
	//	}
	//}
	return true
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
