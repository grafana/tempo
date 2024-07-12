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

type tagRequest struct {
	// applies to tag names and tag values. the conditions by which to return the filtered data
	conditions []traceql.Condition
	// scope requested. only used for tag names. A scope of None means all scopes.
	scope traceql.AttributeScope
	// tag requested.  only used for tag values. if populated then return tag values for this tag, otherwise return tag names.
	tag traceql.Attribute
}

func (r tagRequest) keysRequested(scope traceql.AttributeScope) bool {
	if r.tag != (traceql.Attribute{}) {
		return false
	}

	// none scope means return all scopes
	if r.scope == traceql.AttributeScopeNone {
		return true
	}

	return r.scope == scope
}

func (b *backendBlock) FetchTagNames(ctx context.Context, req traceql.FetchTagsRequest, cb traceql.FetchTagsCallback, opts common.SearchOptions) error {
	err := checkConditions(req.Conditions)
	if err != nil {
		return errors.Wrap(err, "conditions invalid")
	}

	_, mingledConditions, err := categorizeConditions(req.Conditions)
	if err != nil {
		return err
	}

	// Last check. No conditions, use old path. It's much faster.
	if len(req.Conditions) < 1 || mingledConditions {
		return b.SearchTags(ctx, req.Scope, func(t string, scope traceql.AttributeScope) {
			cb(t, scope)
		}, opts)
	}

	pf, _, err := b.openForSearch(ctx, opts)
	if err != nil {
		return err
	}

	tr := tagRequest{
		conditions: req.Conditions,
		scope:      req.Scope,
	}

	iter, err := autocompleteIter(ctx, tr, pf, opts, b.meta.DedicatedColumns)
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
			if cb(oe.Key, oe.Value.(traceql.AttributeScope)) {
				return nil // We have enough values
			}
		}
	}

	tagNamesForSpecialColumns(req.Scope, pf, b.meta.DedicatedColumns, cb)

	return nil
}

func tagNamesForSpecialColumns(scope traceql.AttributeScope, pf *parquet.File, dcs backend.DedicatedColumns, cb traceql.FetchTagsCallback) {
	// currently just seeing if any row groups have values. future improvements:
	// - only check those row groups that otherwise have a match in the iterators above
	// - use rep/def levels to determine if a value exists at a row w/o actually testing values.
	//   atm i believe this requires reading the pages themselves b/c the rep/def lvls come w/ the page
	hasValues := func(path string, pf *parquet.File) bool {
		idx, _ := parquetquery.GetColumnIndexByPath(pf, path)
		md := pf.Metadata()
		for _, rg := range md.RowGroups {
			col := rg.Columns[idx]
			if col.MetaData.NumValues-col.MetaData.Statistics.NullCount > 0 {
				return true
			}
		}

		return false
	}

	// add all well known columns that have values
	for name, entry := range wellKnownColumnLookups {
		if entry.level != scope && scope != traceql.AttributeScopeNone {
			continue
		}

		if hasValues(entry.columnPath, pf) {
			if cb(name, entry.level) {
				return
			}
		}
	}

	// add all span dedicated columns that have values
	if scope == traceql.AttributeScopeNone || scope == traceql.AttributeScopeSpan {
		dedCols := dedicatedColumnsToColumnMapping(dcs, backend.DedicatedColumnScopeSpan)
		for name, col := range dedCols.mapping {
			if hasValues(col.ColumnPath, pf) {
				if cb(name, traceql.AttributeScopeSpan) {
					return
				}
			}
		}
	}

	// add all resource dedicated columns that have values
	if scope == traceql.AttributeScopeNone || scope == traceql.AttributeScopeResource {
		dedCols := dedicatedColumnsToColumnMapping(dcs, backend.DedicatedColumnScopeResource)
		for name, col := range dedCols.mapping {
			if hasValues(col.ColumnPath, pf) {
				if cb(name, traceql.AttributeScopeResource) {
					return
				}
			}
		}
	}
}

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

	tr := tagRequest{
		conditions: req.Conditions,
		tag:        req.TagName,
	}

	iter, err := autocompleteIter(ctx, tr, pf, opts, b.meta.DedicatedColumns)
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
func autocompleteIter(ctx context.Context, tr tagRequest, pf *parquet.File, opts common.SearchOptions, dc backend.DedicatedColumns) (parquetquery.Iterator, error) {
	// categorizeConditions conditions into span-level or resource-level
	catConditions, _, err := categorizeConditions(tr.conditions)
	if err != nil {
		return nil, err
	}

	rgs := rowGroupsFromFile(pf, opts)
	makeIter := makeIterFunc(ctx, rgs, pf)

	var currentIter parquetquery.Iterator

	if len(catConditions.event) > 0 || tr.keysRequested(traceql.AttributeScopeEvent) {
		currentIter, err = createDistinctEventIterator(makeIter, tr, currentIter, catConditions.event)
		if err != nil {
			return nil, errors.Wrap(err, "creating event iterator")
		}
	}

	if len(catConditions.link) > 0 || tr.keysRequested(traceql.AttributeScopeLink) {
		currentIter, err = createDistinctLinkIterator(makeIter, tr, currentIter, catConditions.link)
		if err != nil {
			return nil, errors.Wrap(err, "creating link iterator")
		}
	}

	if len(catConditions.span) > 0 || tr.keysRequested(traceql.AttributeScopeSpan) {
		currentIter, err = createDistinctSpanIterator(makeIter, tr, currentIter, catConditions.span, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating span iterator")
		}
	}

	if len(catConditions.resource) > 0 || tr.keysRequested(traceql.AttributeScopeResource) {
		currentIter, err = createDistinctResourceIterator(makeIter, tr, currentIter, catConditions.resource, dc)
		if err != nil {
			return nil, errors.Wrap(err, "creating resource iterator")
		}
	}

	if len(catConditions.trace) > 0 {
		currentIter, err = createDistinctTraceIterator(makeIter, tr, currentIter, catConditions.trace)
		if err != nil {
			return nil, errors.Wrap(err, "creating trace iterator")
		}
	}

	return currentIter, nil
}

func createDistinctEventIterator(
	makeIter makeIterFn,
	tr tagRequest,
	primaryIter parquetquery.Iterator,
	conditions []traceql.Condition,
) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
	)

	// TODO: Potentially problematic when wanted attribute is also part of a condition
	//     e.g. { span.foo =~ ".*" && span.foo = }
	addSelectAs := func(attr traceql.Attribute, columnPath, selectAs string) {
		if attr == tr.tag {
			columnSelectAs[columnPath] = selectAs
		} else {
			columnSelectAs[columnPath] = "" // Don't select, just filter
		}
	}

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicEventName:
			pred, err := createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			iters = append(iters, makeIter(columnPathEventName, pred, columnSelectAs[columnPathEventName]))
			addSelectAs(cond.Attribute, columnPathEventName, columnPathEventName)
			continue
		}
		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, tr, genericConditions, DefinitionLevelResourceSpansILSSpanEventAttrs,
		columnPathEventAttrKey, columnPathEventAttrString, columnPathEventAttrInt, columnPathEventAttrDouble, columnPathEventAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating event attribute iterator")
	}

	// if no intrinsics and no primary then we can just return the attribute iterator
	if len(iters) == 0 && primaryIter == nil {
		return attrIter, nil
	}

	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	if primaryIter != nil {
		iters = append(iters, primaryIter)
	}

	eventCol := newDistinctValueCollector(mapEventAttr, "event")

	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpanEvent, iters, eventCol), nil
}

func createDistinctLinkIterator(
	makeIter makeIterFn,
	tr tagRequest,
	primaryIter parquetquery.Iterator,
	conditions []traceql.Condition,
) (parquetquery.Iterator, error) {
	var (
		columnSelectAs    = map[string]string{}
		iters             []parquetquery.Iterator
		genericConditions []traceql.Condition
	)

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicLinkTraceID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, false)
			if err != nil {
				return nil, err
			}
			iters = append(iters, makeIter(columnPathLinkTraceID, pred, columnPathLinkTraceID))
			columnSelectAs[columnPathLinkTraceID] = "" // Don't select, just filter
			continue
		case traceql.IntrinsicLinkSpanID:
			pred, err := createBytesPredicate(cond.Op, cond.Operands, false)
			if err != nil {
				return nil, err
			}
			iters = append(iters, makeIter(columnPathLinkSpanID, pred, columnPathLinkSpanID))
			columnSelectAs[columnPathLinkTraceID] = "" // Don't select, just filter
			continue
		}
		// Else: generic attribute lookup
		genericConditions = append(genericConditions, cond)
	}

	attrIter, err := createDistinctAttributeIterator(makeIter, tr, genericConditions, DefinitionLevelResourceSpansILSSpanLinkAttrs,
		columnPathLinkAttrKey, columnPathLinkAttrString, columnPathLinkAttrInt, columnPathLinkAttrDouble, columnPathLinkAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating link attribute iterator")
	}

	// if no intrinsics and no events then we can just return the attribute iterator
	if len(iters) == 0 && primaryIter == nil {
		return attrIter, nil
	}

	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	if primaryIter != nil {
		iters = append(iters, primaryIter)
	}

	linkCol := newDistinctValueCollector(mapLinkAttr, "link")

	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpanEvent, iters, linkCol), nil
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createDistinctSpanIterator(
	makeIter makeIterFn,
	tr tagRequest,
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
	addSelectAs := func(attr traceql.Attribute, columnPath string, selectAs string) {
		if attr == tr.tag {
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

		case traceql.IntrinsicNestedSetLeft:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanNestedSetLeft, pred)
			addSelectAs(cond.Attribute, columnPathSpanNestedSetLeft, columnPathSpanNestedSetLeft)
			continue

		case traceql.IntrinsicNestedSetRight:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanNestedSetRight, pred)
			addSelectAs(cond.Attribute, columnPathSpanNestedSetRight, columnPathSpanNestedSetRight)
			continue

		case traceql.IntrinsicNestedSetParent:
			pred, err := createIntPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			addPredicate(columnPathSpanParentID, pred)
			addSelectAs(cond.Attribute, columnPathSpanParentID, columnPathSpanParentID)
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

	attrIter, err := createDistinctAttributeIterator(makeIter, tr, genericConditions, DefinitionLevelResourceSpansILSSpanAttrs,
		columnPathSpanAttrKey, columnPathSpanAttrString, columnPathSpanAttrInt, columnPathSpanAttrDouble, columnPathSpanAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}

	if len(columnPredicates) == 0 && primaryIter == nil {
		// If no special+intrinsic+dedicated columns + events/links are being searched,
		// we can iterate over the generic attributes directly.
		return attrIter, nil
	}

	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	if len(columnPredicates) == 0 && primaryIter == nil {
		// If no special+intrinsic+dedicated columns are being searched,
		// we can iterate over the generic attributes directly.
		return attrIter, nil
	}

	if primaryIter != nil {
		iters = append(iters, primaryIter)
	}

	spanCol := newDistinctValueCollector(mapSpanAttr, "span")

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpansILSSpan, iters, spanCol), nil
}

func createDistinctAttributeIterator(
	makeIter makeIterFn,
	tr tagRequest,
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
		if tr.tag == attr {
			return key
		}
		return ""
	}

	for _, cond := range conditions {
		if cond.Op == traceql.OpNone {
			// This means we have to scan all values, we don't know what type to expect
			if tr.tag == cond.Attribute {
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

	scope := scopeFromDefinitionLevel(definitionLevel)
	if definitionLevel == DefinitionLevelResourceSpansILSSpanEventAttrs {
		switch keyPath {
		case columnPathEventAttrKey:
			scope = traceql.AttributeScopeEvent
		case columnPathLinkAttrKey:
			scope = traceql.AttributeScopeLink
		}
	}
	if len(valueIters) > 0 || len(iters) > 0 || tr.keysRequested(scope) {
		if len(valueIters) > 0 {
			tagIter, err := parquetquery.NewLeftJoinIterator(
				definitionLevel,
				[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
				valueIters,
				newDistinctAttrCollector(scope, false),
			)
			if err != nil {
				return nil, fmt.Errorf("creating left join iterator: %w", err)
			}
			iters = append(iters, tagIter)
		}

		if tr.keysRequested(scope) {
			return keyNameIterator(makeIter, definitionLevel, keyPath, iters)
		}

		return parquetquery.NewJoinIterator(
			oneLevelUp(definitionLevel),
			iters,
			nil,
		), nil
	}

	return nil, nil
}

func keyNameIterator(makeIter makeIterFn, definitionLevel int, keyPath string, attrIters []parquetquery.Iterator) (parquetquery.Iterator, error) {
	scope := scopeFromDefinitionLevel(definitionLevel)
	if definitionLevel == DefinitionLevelResourceSpansILSSpanEventAttrs {
		switch keyPath {
		case columnPathEventAttrKey:
			scope = traceql.AttributeScopeEvent
		case columnPathLinkAttrKey:
			scope = traceql.AttributeScopeLink
		}
	}

	if len(attrIters) == 0 {
		return parquetquery.NewJoinIterator(
			oneLevelUp(definitionLevel),
			[]parquetquery.Iterator{makeIter(keyPath, nil, "key")},
			newDistinctAttrCollector(scope, true),
		), nil
	}

	return parquetquery.NewLeftJoinIterator(
		oneLevelUp(definitionLevel),
		attrIters,
		[]parquetquery.Iterator{makeIter(keyPath, nil, "key")},
		newDistinctAttrCollector(scope, true),
	)
}

func oneLevelUp(definitionLevel int) int {
	switch definitionLevel {
	case DefinitionLevelResourceSpansILSSpanAttrs:
		return DefinitionLevelResourceSpansILSSpan
	case DefinitionLevelResourceAttrs:
		return DefinitionLevelResourceSpans
	case DefinitionLevelResourceSpansILSSpanEventAttrs: // should cover links as well
		return DefinitionLevelResourceSpansILSSpanEvent

	}
	return definitionLevel
}

func createDistinctResourceIterator(
	makeIter makeIterFn,
	tr tagRequest,
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

	addSelectAs := func(attr traceql.Attribute, columnPath string, selectAs string) {
		if attr == tr.tag {
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
				if tr.tag != cond.Attribute {
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

	attrIter, err := createDistinctAttributeIterator(makeIter, tr, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	batchCol := newDistinctValueCollector(mapResourceAttr, "resource")

	// Put span iterator last, so it is only read when
	// the resource conditions are met.
	if spanIterator != nil {
		iters = append(iters, spanIterator)
	}

	return parquetquery.NewJoinIterator(DefinitionLevelResourceSpans, iters, batchCol), nil
}

func createDistinctTraceIterator(
	makeIter makeIterFn,
	tr tagRequest,
	resourceIter parquetquery.Iterator,
	conds []traceql.Condition,
) (parquetquery.Iterator, error) {
	var err error
	traceIters := make([]parquetquery.Iterator, 0, 3)

	selectAs := func(attr traceql.Attribute, columnPath string) string {
		if attr == tr.tag {
			return columnPath
		}
		return ""
	}

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
			traceIters = append(traceIters, makeIter(columnPathDurationNanos, pred, selectAs(cond.Attribute, columnPathDurationNanos)))

		case traceql.IntrinsicTraceRootSpan:
			var pred parquetquery.Predicate
			pred, err = createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			traceIters = append(traceIters, makeIter(columnPathRootSpanName, pred, selectAs(cond.Attribute, columnPathRootSpanName)))

		case traceql.IntrinsicTraceRootService:
			var pred parquetquery.Predicate
			pred, err = createStringPredicate(cond.Op, cond.Operands)
			if err != nil {
				return nil, err
			}
			traceIters = append(traceIters, makeIter(columnPathRootServiceName, pred, selectAs(cond.Attribute, columnPathRootServiceName)))
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
	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, newDistinctValueCollector(mapTraceAttr, "trace")), nil
}

var _ parquetquery.GroupPredicate = (*distinctAttrCollector)(nil)

type distinctAttrCollector struct {
	scope     traceql.AttributeScope
	attrNames bool

	sentVals map[traceql.Static]struct{}
	sentKeys map[string]struct{}
}

func newDistinctAttrCollector(scope traceql.AttributeScope, attrNames bool) *distinctAttrCollector {
	return &distinctAttrCollector{
		scope:     scope,
		sentVals:  make(map[traceql.Static]struct{}),
		sentKeys:  make(map[string]struct{}),
		attrNames: attrNames,
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

		if d.attrNames {
			if e.Key == "key" {
				key := unsafeToString(e.Value.ByteArray())
				if _, ok := d.sentKeys[key]; !ok {
					result.AppendOtherValue(key, d.scope)
					d.sentKeys[key] = struct{}{}
				}
			}
		} else {
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
	}

	var empty traceql.Static
	if val != empty {
		if _, ok := d.sentVals[val]; !ok {
			result.AppendOtherValue("", val)
			d.sentVals[val] = struct{}{}
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
	sentVals    map[traceql.Static]struct{}
	name        string
}

func newDistinctValueCollector(mapToStatic func(entry) traceql.Static, name string) *distinctValueCollector {
	return &distinctValueCollector{
		mapToStatic: mapToStatic,
		sentVals:    make(map[traceql.Static]struct{}),
		name:        name,
	}
}

func (d distinctValueCollector) String() string { return "distinctValueCollector(" + d.name + ")" }

func (d distinctValueCollector) KeepGroup(result *parquetquery.IteratorResult) bool {
	for _, e := range result.Entries {
		if e.Value.IsNull() {
			continue
		}
		static := d.mapToStatic(e)

		if _, ok := d.sentVals[static]; !ok {
			result.AppendOtherValue("", static)
			d.sentVals[static] = struct{}{}
		}
	}
	result.Entries = result.Entries[:0]
	return true
}

func mapEventAttr(e entry) traceql.Static {
	switch e.Key {
	case columnPathEventName:
		return traceql.NewStaticString(unsafeToString(e.Value.ByteArray()))
	}
	return traceql.Static{}
}

func mapLinkAttr(_ entry) traceql.Static {
	return traceql.Static{}
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
	return traceql.Static{}
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
		return traceql.Static{}
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
	return traceql.Static{}
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
