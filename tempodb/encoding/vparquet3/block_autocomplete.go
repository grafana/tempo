package vparquet3

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/parquetquery"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/parquet-go/parquet-go"
	"github.com/pkg/errors"
)

func (b *backendBlock) FetchTagValues(ctx context.Context, req traceql.AutocompleteRequest, cb traceql.AutocompleteCallback, opts common.SearchOptions) error {
	err := checkConditions(req.Conditions)
	if err != nil {
		return errors.Wrap(err, "conditions invalid")
	}

	if len(req.Conditions) <= 1 { // Last check. No conditions, use old path. It's much faster.
		return b.SearchTagValuesV2(ctx, req.TagName, common.TagCallbackV2(cb), opts)
	}

	pf, _, err := b.openForSearch(ctx, opts)
	if err != nil {
		return err
	}

	iter, err := autocompleteIter(ctx, req, pf, opts, b.meta.DedicatedColumns)
	if err != nil {
		return errors.Wrap(err, "creating fetch iter")
	}
	defer iter.Close()

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
		for _, oe := range res.OtherEntries {
			if oe.Key == req.TagName.String() {
				v := oe.Value.(traceql.Static)
				if cb(v) {
					return nil // We have enough values
				}
			}
		}
	}

	return nil
}

// autocompleteIter creates an iterator that will collect values for a given attribute/tag.
func autocompleteIter(ctx context.Context, req traceql.AutocompleteRequest, pf *parquet.File, opts common.SearchOptions, dc backend.DedicatedColumns) (parquetquery.Iterator, error) {
	iter, err := createDistinctIterator(ctx, nil, req.Conditions, req.TagName, true, pf, opts, dc)
	if err != nil {
		return nil, fmt.Errorf("error creating iterator: %w", err)
	}

	return iter, nil
}

func createDistinctIterator(
	ctx context.Context,
	primaryIter parquetquery.Iterator,
	conds []traceql.Condition,
	tag traceql.Attribute,
	allConditions bool,
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

	// Safeguard. Shouldn't be needed since we only use collect the tag we want.
	keep := func(attr traceql.Attribute) bool { return tag == attr }

	// Optimization for queries like {resource.x... && span.y ...}
	// Requires no mingled scopes like .foo=x, which could be satisfied one either resource or span.
	allConditions = allConditions && !mingledConditions

	// TODO: Return early if there are mingled conditions? They're not supported yet (and possibly never will be).

	var (
		currentIter parquetquery.Iterator
		err         error
	)

	if len(spanConditions) > 0 {
		currentIter, err = createDistinctSpanIterator(makeIter, keep, tag, primaryIter, spanConditions, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating span iterator")
		}
	}

	if len(resourceConditions) > 0 {
		currentIter, err = createDistinctResourceIterator(makeIter, keep, tag, currentIter, resourceConditions, allConditions, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating resource iterator")
		}
	}

	if len(traceConditions) > 0 {
		currentIter, err = createDistinctTraceIterator(makeIter, keep, currentIter, traceConditions, allConditions)
		if err != nil {
			return nil, errors.Wrap(err, "creating trace iterator")
		}
	}

	return currentIter, nil
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createDistinctSpanIterator(
	makeIter makeIterFn,
	keep keepFn,
	tag traceql.Attribute,
	primaryIter parquetquery.Iterator,
	conditions []traceql.Condition,
	allConditions bool,
	dedicatedColumns backend.DedicatedColumns,
) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		columnPredicates  = map[string][]parquetquery.Predicate{}
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
		columnMapping     = dedicatedColumnsToColumnMapping(dedicatedColumns, backend.DedicatedColumnScopeSpan)
	)

	// TODO: Potentially problematic when wanted attribute is also part of a condition
	//     e.g. { span.foo =~ ".*" && span.foo = }
	addSelectAs := func(attr traceql.Attribute, columnPath string, selectAs string) {
		if attr == tag {
			columnSelectAs[columnPath] = selectAs
		} else {
			columnSelectAs[columnPath] = "" // Don't select, just filter
		}
	}

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {

		case traceql.IntrinsicSpanID,
			traceql.IntrinsicSpanStartTime:
			// Metadata conditions not necessary, we don't need to fetch them
			// TODO: Add support if they're added to TraceQL
			continue

		case traceql.IntrinsicName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanName, pred)
			addSelectAs(cond.Attribute, columnPathSpanName, columnPathSpanName)
			continue

		case traceql.IntrinsicKind:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanKind, pred)
			addSelectAs(cond.Attribute, columnPathSpanKind, columnPathSpanKind)
			continue

		case traceql.IntrinsicDuration:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanDuration, pred)
			addSelectAs(cond.Attribute, columnPathSpanDuration, columnPathSpanDuration)
			continue

		case traceql.IntrinsicStatus:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanStatusCode, pred)
			addSelectAs(cond.Attribute, columnPathSpanStatusCode, columnPathSpanStatusCode)
			continue

		case traceql.IntrinsicStatusMessage:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanStatusMessage, pred)
			addSelectAs(cond.Attribute, columnPathSpanStatusMessage, columnPathSpanStatusMessage)
			continue

		// TODO: Support structural operators
		case traceql.IntrinsicStructuralDescendant,
			traceql.IntrinsicStructuralChild,
			traceql.IntrinsicStructuralSibling:
			continue
		}

		// Well-known attribute?
		if entry, ok := wellKnownColumnLookups[cond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeResource {
			if cond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				addSelectAs(cond.Attribute, entry.columnPath, cond.Attribute.Name)
				continue
			}

			// Compatible type?
			if entry.typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				addPredicate(entry.columnPath, pred)
				addSelectAs(cond.Attribute, entry.columnPath, cond.Attribute.Name)
				continue
			}
		}

		// Attributes stored in dedicated columns
		if c, ok := columnMapping.get(cond.Attribute.Name); ok {
			if cond.Op == traceql.OpNone {
				addPredicate(c.ColumnPath, nil) // No filtering
				addSelectAs(cond.Attribute, c.ColumnPath, cond.Attribute.Name)
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
				addSelectAs(cond.Attribute, c.ColumnPath, cond.Attribute.Name)
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, orIfNeeded(predicates), columnSelectAs[columnPath]))
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, keep, tag, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
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

	// TODO: Unsure how this works when there are unscoped conditions like .foo=x
	if len(required) == 0 {
		return attrIter, nil
	}

	spanCol := &distinctSpanCollector{keep: keep}

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, spanCol), nil
}

func createDistinctAttributeIterator(
	makeIter makeIterFn,
	keep keepFn,
	tag traceql.Attribute,
	conditions []traceql.Condition,
	definitionLevel int,
	keyPath, strPath, intPath, floatPath, boolPath string,
	allConditions bool,
) (parquetquery.Iterator, error) {
	var (
		attrKeys                                               []string
		attrStringPreds, attrIntPreds, attrFltPreds, boolPreds []parquetquery.Predicate
		allIters                                               []parquetquery.Iterator
	)

	selectAs := func(key string, attr traceql.Attribute) string {
		if tag == attr {
			return key
		}
		return ""
	}

	for _, cond := range conditions {

		if cond.Op == traceql.OpNone {
			// This means we have to scan all values, we don't know what type to expect
			if tag == cond.Attribute {
				// If it's not the tag we're looking for, we can skip it
				attrKeys = append(attrKeys, cond.Attribute.Name)
				attrStringPreds = append(attrStringPreds, nil)
				attrIntPreds = append(attrIntPreds, nil)
				attrFltPreds = append(attrFltPreds, nil)
				boolPreds = append(boolPreds, nil)
				continue
			}
		}

		var keyIter, valIter parquetquery.Iterator

		switch cond.Operands[0].Type {
		case traceql.TypeString:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), selectAs("key", cond.Attribute))
			valIter = makeIter(strPath, pred, selectAs("string", cond.Attribute))

		case traceql.TypeInt:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), selectAs("key", cond.Attribute))
			valIter = makeIter(intPath, pred, selectAs("int", cond.Attribute))

		case traceql.TypeFloat:
			pred, err := createFloatPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), selectAs("key", cond.Attribute))
			valIter = makeIter(floatPath, pred, selectAs("float", cond.Attribute))

		case traceql.TypeBoolean:
			pred, err := createBoolPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, fmt.Errorf("creating attribute predicate: %w", err)
			}
			keyIter = makeIter(keyPath, parquetquery.NewStringInPredicate([]string{cond.Attribute.Name}), selectAs("key", cond.Attribute))
			valIter = makeIter(boolPath, pred, selectAs("bool", cond.Attribute))
		}

		allIters = append(allIters, parquetquery.NewJoinIterator(definitionLevel, []parquetquery.Iterator{keyIter, valIter}, &distinctAttrCollector{
			scope: scopeFromDefinitionLevel(definitionLevel),
			keep:  keep,
		}))
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
				&distinctAttrCollector{
					scope: scopeFromDefinitionLevel(definitionLevel),
					keep:  keep,
				},
			), nil
		}

		return parquetquery.NewLeftJoinIterator(
			definitionLevel,
			[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
			valueIters,
			&distinctAttrCollector{
				scope: scopeFromDefinitionLevel(definitionLevel),
				keep:  keep,
			},
		), nil
	}

	return nil, nil
}

func createDistinctResourceIterator(
	makeIter makeIterFn,
	keep keepFn,
	tag traceql.Attribute,
	spanIterator parquetquery.Iterator,
	conditions []traceql.Condition,
	allConditions bool,
	dedicatedColumns backend.DedicatedColumns,
) (parquetquery.Iterator, error) {
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

	addSelectAs := func(attr traceql.Attribute, columnPath string, selectAs string) {
		if attr == tag {
			columnSelectAs[columnPath] = selectAs
		} else {
			columnSelectAs[columnPath] = "" // Don't select, just filter
		}
	}

	for _, cond := range conditions {
		// Well-known selector?
		if entry, ok := wellKnownColumnLookups[cond.Attribute.Name]; ok && entry.level != traceql.AttributeScopeSpan {
			if cond.Op == traceql.OpNone {
				addPredicate(entry.columnPath, nil) // No filtering
				addSelectAs(cond.Attribute, entry.columnPath, cond.Attribute.Name)
				continue
			}

			// Compatible type?
			if entry.typ == operandType(cond.Operands) {
				pred, err := createPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, errors.Wrap(err, "creating predicate")
				}
				selectAs := cond.Attribute.Name
				if tag != cond.Attribute {
					selectAs = ""
				}
				iters = append(iters, makeIter(entry.columnPath, pred, selectAs))
				continue
			}
		}

		// Attributes stored in dedicated columns
		if c, ok := columnMapping.get(cond.Attribute.Name); ok {
			if cond.Op == traceql.OpNone {
				addPredicate(c.ColumnPath, nil) // No filtering
				addSelectAs(cond.Attribute, c.ColumnPath, cond.Attribute.Name)
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
				addSelectAs(cond.Attribute, c.ColumnPath, cond.Attribute.Name)
				continue
			}
		}

		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, orIfNeeded(predicates), columnSelectAs[columnPath]))
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, keep, tag, genericConditions, DefinitionLevelResourceAttrs,
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

func createDistinctTraceIterator(
	makeIter makeIterFn,
	keep keepFn,
	resourceIter parquetquery.Iterator,
	conds []traceql.Condition,
	allConditions bool,
) (parquetquery.Iterator, error) {
	var err error
	traceIters := make([]parquetquery.Iterator, 0, 3)

	// add conditional iterators first. this way if someone searches for { traceDuration > 1s && span.foo = "bar"} the query will
	// be sped up by searching for traceDuration first. note that we can only set the predicates if all conditions is true.
	// otherwise we just pass the info up to the engine to make a choice
	for _, cond := range conds {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicTraceID, traceql.IntrinsicTraceStartTime:
			// metadata conditions not necessary, we don't need to fetch them

		case traceql.IntrinsicTraceDuration:
			var pred parquetquery.Predicate
			if allConditions {
				pred, err = createIntPredicate(cond.Op, cond.Operands)
				if err != nil {
					return nil, err
				}
			}
			traceIters = append(traceIters, makeIter(columnPathDurationNanos, pred, columnPathDurationNanos))

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
	if resourceIter != nil {
		traceIters = append(traceIters, resourceIter)
	}

	// Final trace iterator
	// Join iterator means it requires matching resources to have been found
	// TraceCollor adds trace-level data to the spansets
	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &distinctTraceCollector{
		keep: keep,
	}), nil
}

type keepFn func(attr traceql.Attribute) bool

var _ parquetquery.GroupPredicate = (*distinctAttrCollector)(nil)

type distinctAttrCollector struct {
	scope traceql.AttributeScope
	keep  keepFn
}

func (d *distinctAttrCollector) String() string {
	return "distinctAttrCollector"
}

func (d *distinctAttrCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	var key string
	var val traceql.Static

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
			val = traceql.NewStaticString(e.Value.String())
		case "int":
			val = traceql.NewStaticInt(int(e.Value.Int64()))
		case "float":
			val = traceql.NewStaticFloat(e.Value.Double())
		case "bool":
			val = traceql.NewStaticBool(e.Value.Boolean())
		}
	}

	attr := traceql.NewScopedAttribute(d.scope, false, key)
	if d.keep(attr) {
		result.AppendOtherValue(attr.String(), val)
	}

	result.Entries = result.Entries[:0]

	return true
}

type entry struct {
	Key   string
	Value parquet.Value
}

var _ parquetquery.GroupPredicate = (*distinctSpanCollector)(nil)

type distinctSpanCollector struct {
	keep keepFn
}

func (d distinctSpanCollector) String() string { return "distinctSpanCollector" }

func (d distinctSpanCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		if attr, static := mapSpanAttr(e); d.keep(attr) {
			result.AppendOtherValue(attr.String(), static)
		}
	}
	result.Entries = result.Entries[:0]
	return true
}

func mapSpanAttr(e entry) (traceql.Attribute, traceql.Static) {
	switch e.Key {
	case columnPathSpanID,
		columnPathSpanParentID,
		columnPathSpanNestedSetLeft,
		columnPathSpanNestedSetRight,
		columnPathSpanStartTime:
	case columnPathSpanDuration:
		return traceql.IntrinsicDurationAttribute, traceql.NewStaticDuration(time.Duration(e.Value.Int64()))
	case columnPathSpanName:
		return traceql.IntrinsicNameAttribute, traceql.NewStaticString(e.Value.String())
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
		return traceql.IntrinsicStatusAttribute, traceql.NewStaticStatus(status)
	case columnPathSpanStatusMessage:
		return traceql.IntrinsicStatusMessageAttribute, traceql.NewStaticString(e.Value.String())
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
		return traceql.IntrinsicKindAttribute, traceql.NewStaticKind(kind)
	default:
		// This exists for span-level dedicated columns like http.status_code
		switch e.Value.Kind() {
		case parquet.Boolean:
			return newSpanAttr(e.Key), traceql.NewStaticBool(e.Value.Boolean())
		case parquet.Int32, parquet.Int64:
			return newSpanAttr(e.Key), traceql.NewStaticInt(int(e.Value.Int64()))
		case parquet.Float:
			return newSpanAttr(e.Key), traceql.NewStaticFloat(e.Value.Double())
		case parquet.ByteArray, parquet.FixedLenByteArray:
			return newSpanAttr(e.Key), traceql.NewStaticString(e.Value.String())
		}
	}
	return traceql.Attribute{}, traceql.Static{}
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
		if attr, static := mapResourceAttr(e); d.keep(attr) {
			result.AppendOtherValue(attr.String(), static)
		}
	}
	result.Entries = result.Entries[:0]
	return true
}

func mapResourceAttr(e entry) (traceql.Attribute, traceql.Static) {
	switch e.Value.Kind() {
	case parquet.Boolean:
		return newResAttr(e.Key), traceql.NewStaticBool(e.Value.Boolean())
	case parquet.Int32, parquet.Int64:
		return newResAttr(e.Key), traceql.NewStaticInt(int(e.Value.Int64()))
	case parquet.Float:
		return newResAttr(e.Key), traceql.NewStaticFloat(e.Value.Double())
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return newResAttr(e.Key), traceql.NewStaticString(e.Value.String())
	default:
		return traceql.Attribute{}, traceql.Static{}
	}
}

var _ parquetquery.GroupPredicate = (*distinctTraceCollector)(nil)

type distinctTraceCollector struct {
	keep keepFn
}

func (d *distinctTraceCollector) String() string {
	return "distinctTraceCollector"
}

func (d *distinctTraceCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		if attr, static := mapTraceAttr(e); d.keep(attr) {
			result.AppendOtherValue(attr.String(), static)
		}
	}
	result.Entries = result.Entries[:0]
	return true
}

func mapTraceAttr(e entry) (traceql.Attribute, traceql.Static) {
	switch e.Key {
	case columnPathTraceID, columnPathEndTimeUnixNano, columnPathStartTimeUnixNano: // No TraceQL intrinsics for these
	case columnPathDurationNanos:
		return traceql.IntrinsicTraceDurationAttribute, traceql.NewStaticDuration(time.Duration(e.Value.Int64()))
	case columnPathRootSpanName:
		return traceql.IntrinsicTraceRootSpanAttribute, traceql.NewStaticString(e.Value.String())
	case columnPathRootServiceName:
		return traceql.IntrinsicTraceRootServiceAttribute, traceql.NewStaticString(e.Value.String())
	}
	return traceql.Attribute{}, traceql.Static{}
}

func scopeFromDefinitionLevel(lvl int) traceql.AttributeScope {
	switch lvl {
	case DefinitionLevelResourceSpansILSSpan,
		DefinitionLevelResourceSpansILSSpanAttrs:
		return traceql.AttributeScopeSpan
	case DefinitionLevelResourceAttrs,
		DefinitionLevelResourceSpans:
		return traceql.AttributeScopeResource
	default:
		return traceql.AttributeScopeNone
	}
}
