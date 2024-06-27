package vparquet4

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

func (b *backendBlock) FetchTagValues(ctx context.Context, req traceql.FetchTagValuesRequest, cb traceql.FetchTagValuesCallback, opts common.SearchOptions) error {
	err := checkConditions(req.Conditions)
	if err != nil {
		return errors.Wrap(err, "conditions invalid")
	}

	_, mingledConditions, err := categorizeConditions(req.Conditions)
	if err != nil {
		return err
	}

	// Last check. No conditions, use old path. It's much faster.
	if len(req.Conditions) <= 1 || mingledConditions { // <= 1 because we always have a "OpNone" condition for the tag name
		return b.SearchTagValuesV2(ctx, req.TagName, common.TagValuesCallbackV2(cb), common.DefaultSearchOptions())
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
			v := oe.Value.(traceql.Static)
			if cb(v) {
				return nil // We have enough values
			}
		}
	}

	return nil
}

// autocompleteIter creates an iterator that will collect values for a given attribute/tag.
func autocompleteIter(ctx context.Context, req traceql.FetchTagValuesRequest, pf *parquet.File, opts common.SearchOptions, dc backend.DedicatedColumns) (parquetquery.Iterator, error) {
	iter, err := createDistinctIterator(ctx, req.Conditions, req.TagName, pf, opts, dc)
	if err != nil {
		return nil, fmt.Errorf("error creating iterator: %w", err)
	}

	return iter, nil
}

func createDistinctIterator(
	ctx context.Context,
	conds []traceql.Condition,
	tag traceql.Attribute,
	pf *parquet.File,
	opts common.SearchOptions,
	dc backend.DedicatedColumns,
) (parquetquery.Iterator, error) {
	// categorize conditions by scope
	catConditions, _, err := categorizeConditions(conds)
	if err != nil {
		return nil, err
	}

	rgs := rowGroupsFromFile(pf, opts)
	makeIter := makeIterFunc(ctx, rgs, pf)

	var currentIter parquetquery.Iterator

	if len(catConditions.span) > 0 {
		currentIter, err = createDistinctSpanIterator(makeIter, tag, currentIter, catConditions.span, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating span iterator")
		}
	}

	if len(catConditions.resource) > 0 {
		currentIter, err = createDistinctResourceIterator(makeIter, tag, currentIter, catConditions.resource, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating resource iterator")
		}
	}

	if len(catConditions.trace) > 0 {
		currentIter, err = createDistinctTraceIterator(makeIter, currentIter, catConditions.trace)
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
	tag traceql.Attribute,
	primaryIter parquetquery.Iterator,
	conditions []traceql.Condition,
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
	addSelectAs := func(attr traceql.Attribute, columnPath, selectAs string) {
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
			traceql.IntrinsicStructuralSibling,
			// nested set intrinsics should not be considered when autocompleting
			traceql.IntrinsicNestedSetLeft,
			traceql.IntrinsicNestedSetRight,
			traceql.IntrinsicNestedSetParent:
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

	attrIter, err := createDistinctAttributeIterator(makeIter, tag, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	if primaryIter != nil {
		iters = append([]parquetquery.Iterator{primaryIter}, iters...)
	}

	if len(columnPredicates) == 0 {
		// If no special+intrinsic+dedicated columns are being searched,
		// we can iterate over the generic attributes directly.
		return attrIter, nil
	}

	spanCol := newDistinctValueCollector(mapSpanAttr)

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, iters, spanCol), nil
}

func createDistinctAttributeIterator(
	makeIter makeIterFn,
	tag traceql.Attribute,
	conditions []traceql.Condition,
	definitionLevel int,
	keyPath, strPath, intPath, floatPath, boolPath string,
) (parquetquery.Iterator, error) {
	var (
		attrKeys                                               []string
		attrStringPreds, attrIntPreds, attrFltPreds, boolPreds []parquetquery.Predicate
		iters                                                  []parquetquery.Iterator
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
			}
			continue
		}

		var keyIter, valIter parquetquery.Iterator

		switch cond.Operands[0].Type() {
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
		default:
			// Generic attributes don't support special types (e.g. duration, status, kind)
			// If we get here, it means we're trying to search for a special type in a generic attribute
			// e.g. { span.foo = 1s }
			// This is not supported. Condition will be ignored.
			continue
		}

		iters = append(iters, parquetquery.NewJoinIterator(definitionLevel, []parquetquery.Iterator{keyIter, valIter}, nil))
	}

	var valueIters []parquetquery.Iterator
	if len(attrStringPreds) > 0 {
		valueIters = append(valueIters, makeIter(strPath, orIfNeeded(attrStringPreds), "string"))
	}
	if len(attrIntPreds) > 0 {
		valueIters = append(valueIters, makeIter(intPath, orIfNeeded(attrIntPreds), "int"))
	}
	if len(attrFltPreds) > 0 {
		valueIters = append(valueIters, makeIter(floatPath, orIfNeeded(attrFltPreds), "float"))
	}
	if len(boolPreds) > 0 {
		valueIters = append(valueIters, makeIter(boolPath, orIfNeeded(boolPreds), "bool"))
	}

	if len(valueIters) > 0 || len(iters) > 0 {
		if len(valueIters) > 0 {
			tagIter, err := parquetquery.NewLeftJoinIterator(
				definitionLevel,
				[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
				valueIters,
				newDistinctAttrCollector(scopeFromDefinitionLevel(definitionLevel)),
			)
			if err != nil {
				return nil, fmt.Errorf("creating left join iterator: %w", err)
			}
			iters = append(iters, tagIter)
		}
		return parquetquery.NewJoinIterator(
			oneLevelUp(definitionLevel),
			iters,
			nil,
		), nil
	}

	return nil, nil
}

func oneLevelUp(definitionLevel int) int {
	switch definitionLevel {
	case DefinitionLevelResourceSpansILSSpanAttrs:
		return DefinitionLevelResourceSpansILSSpan
	case DefinitionLevelResourceAttrs:
		return DefinitionLevelResourceSpans
	}
	return definitionLevel
}

func createDistinctResourceIterator(
	makeIter makeIterFn,
	tag traceql.Attribute,
	spanIterator parquetquery.Iterator,
	conditions []traceql.Condition,
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

	addSelectAs := func(attr traceql.Attribute, columnPath, selectAs string) {
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

	attrIter, err := createDistinctAttributeIterator(makeIter, tag, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	batchCol := newDistinctValueCollector(mapResourceAttr)

	// Put span iterator last, so it is only read when
	// the resource conditions are met.
	if spanIterator != nil {
		iters = append(iters, spanIterator)
	}

	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, iters, batchCol), nil
}

func createDistinctTraceIterator(
	makeIter makeIterFn,
	resourceIter parquetquery.Iterator,
	conds []traceql.Condition,
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
			pred, err = createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			traceIters = append(traceIters, makeIter(columnPathDurationNanos, pred, columnPathDurationNanos))

		case traceql.IntrinsicTraceRootSpan:
			var pred parquetquery.Predicate
			pred, err = createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			traceIters = append(traceIters, makeIter(columnPathRootSpanName, pred, columnPathRootSpanName))

		case traceql.IntrinsicTraceRootService:
			var pred parquetquery.Predicate
			pred, err = createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
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
	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, newDistinctValueCollector(mapTraceAttr)), nil
}

var _ parquetquery.GroupPredicate = (*distinctAttrCollector)(nil)

type distinctAttrCollector struct {
	scope traceql.AttributeScope

	sentVals map[traceql.StaticMapKey]struct{}
}

func newDistinctAttrCollector(scope traceql.AttributeScope) *distinctAttrCollector {
	return &distinctAttrCollector{
		scope:    scope,
		sentVals: make(map[traceql.StaticMapKey]struct{}),
	}
}

func (d *distinctAttrCollector) String() string {
	return "distinctAttrCollector"
}

func (d *distinctAttrCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	var val traceql.Static

	for _, e := range result.Entries {
		// Ignore nulls, this leaves val as the remaining found value,
		// or nil if the key was found but no matching values
		if e.Value.Kind() < 0 {
			continue
		}

		switch e.Key {
		case "string":
			val = traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
		case "int":
			val = traceql.NewStaticInt(int(e.Value.Int64()))
		case "float":
			val = traceql.NewStaticFloat(e.Value.Double())
		case "bool":
			val = traceql.NewStaticBool(e.Value.Boolean())
		}
	}

	if val != nil {
		if _, ok := d.sentVals[val.MapKey()]; !ok {
			result.AppendOtherValue("", val)
			d.sentVals[val.MapKey()] = struct{}{}
		}
	}

	result.Entries = result.Entries[:0]

	return true
}

type entry struct {
	Key   string
	Value parquet.Value
}

var _ parquetquery.GroupPredicate = (*distinctValueCollector)(nil)

type distinctValueCollector struct {
	mapToStatic func(entry) traceql.Static
	sentVals    map[traceql.StaticMapKey]struct{}
}

func newDistinctValueCollector(mapToStatic func(entry) traceql.Static) *distinctValueCollector {
	return &distinctValueCollector{
		mapToStatic: mapToStatic,
		sentVals:    make(map[traceql.StaticMapKey]struct{}),
	}
}

func (d distinctValueCollector) String() string { return "distinctValueCollector" }

func (d distinctValueCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		if e.Value.IsNull() {
			continue
		}
		static := d.mapToStatic(e)

		if _, ok := d.sentVals[static.MapKey()]; !ok {
			result.AppendOtherValue("", static)
			d.sentVals[static.MapKey()] = struct{}{}
		}
	}
	result.Entries = result.Entries[:0]
	return true
}

func mapSpanAttr(e entry) traceql.Static {
	switch e.Key {
	case columnPathSpanID,
		columnPathSpanParentID,
		columnPathSpanNestedSetLeft,
		columnPathSpanNestedSetRight,
		columnPathSpanStartTime:
	case columnPathSpanDuration:
		return traceql.NewStaticDuration(time.Duration(e.Value.Int64()))
	case columnPathSpanName:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
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
		return traceql.NewStaticStatus(status)
	case columnPathSpanStatusMessage:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
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
		return traceql.NewStaticKind(kind)
	default:
		// This exists for span-level dedicated columns like http.status_code
		switch e.Value.Kind() {
		case parquet.Boolean:
			return traceql.NewStaticBool(e.Value.Boolean())
		case parquet.Int32, parquet.Int64:
			return traceql.NewStaticInt(int(e.Value.Int64()))
		case parquet.Float:
			return traceql.NewStaticFloat(e.Value.Double())
		case parquet.ByteArray, parquet.FixedLenByteArray:
			return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
		}
	}
	return traceql.NewStaticNil()
}

func mapResourceAttr(e entry) traceql.Static {
	switch e.Value.Kind() {
	case parquet.Boolean:
		return traceql.NewStaticBool(e.Value.Boolean())
	case parquet.Int32, parquet.Int64:
		return traceql.NewStaticInt(int(e.Value.Int64()))
	case parquet.Float:
		return traceql.NewStaticFloat(e.Value.Double())
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
	default:
		return traceql.NewStaticNil()
	}
}

func mapTraceAttr(e entry) traceql.Static {
	switch e.Key {
	case columnPathTraceID, columnPathEndTimeUnixNano, columnPathStartTimeUnixNano: // No TraceQL intrinsics for these
	case columnPathDurationNanos:
		return traceql.NewStaticDuration(time.Duration(e.Value.Int64()))
	case columnPathRootSpanName:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
	case columnPathRootServiceName:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
	}
	return traceql.NewStaticNil()
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
