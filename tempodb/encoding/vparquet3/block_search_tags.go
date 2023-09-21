package vparquet3

import (
	"context"
	"errors"
	"fmt"
	"io"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
	"github.com/parquet-go/parquet-go"
)

var translateTagToAttribute = map[string]traceql.Attribute{
	LabelName:                   traceql.NewIntrinsic(traceql.IntrinsicName),
	LabelStatusCode:             traceql.NewIntrinsic(traceql.IntrinsicStatus),
	LabelTraceQLRootName:        traceql.NewIntrinsic(traceql.IntrinsicTraceRootSpan),
	LabelTraceQLRootServiceName: traceql.NewIntrinsic(traceql.IntrinsicTraceRootService),

	// Preserve behavior of v1 tag lookups which directed some attributes
	// to dedicated columns.
	LabelServiceName:      traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelServiceName),
	LabelCluster:          traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelCluster),
	LabelNamespace:        traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelNamespace),
	LabelPod:              traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelPod),
	LabelContainer:        traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelContainer),
	LabelK8sNamespaceName: traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sNamespaceName),
	LabelK8sClusterName:   traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sClusterName),
	LabelK8sPodName:       traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sPodName),
	LabelK8sContainerName: traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, LabelK8sContainerName),
	LabelHTTPMethod:       traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPMethod),
	LabelHTTPUrl:          traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPUrl),
	LabelHTTPStatusCode:   traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, LabelHTTPStatusCode),
}

var nonTraceQLAttributes = map[string]string{
	LabelRootServiceName: columnPathRootServiceName,
	LabelRootSpanName:    columnPathRootSpanName,
}

func (b *backendBlock) SearchTags(ctx context.Context, scope traceql.AttributeScope, cb common.TagCallback, opts common.SearchOptions) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.SearchTags",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.BytesRead()) }()

	return searchTags(derivedCtx, scope, cb, pf, b.meta.DedicatedColumns)
}

func searchTags(_ context.Context, scope traceql.AttributeScope, cb common.TagCallback, pf *parquet.File, dc backend.DedicatedColumns) error {
	standardAttrIdxs := make([]int, 0, 2) // the most we can have is 2, resource and span indexes depending on scope passed
	specialAttrIdxs := map[int]string{}

	addToIndexes := func(standardKeyPath string, specialMappings map[string]string, columnMapping dedicatedColumnMapping) error {
		// standard attributes
		resourceKeyIdx, _ := pq.GetColumnIndexByPath(pf, standardKeyPath)
		if resourceKeyIdx == -1 {
			return fmt.Errorf("resource attributes col not found (%d)", resourceKeyIdx)
		}
		standardAttrIdxs = append(standardAttrIdxs, resourceKeyIdx)

		// special attributes
		for lbl, col := range specialMappings {
			idx, _ := pq.GetColumnIndexByPath(pf, col)
			if idx == -1 {
				continue
			}

			specialAttrIdxs[idx] = lbl
		}

		// dedicated attributes
		columnMapping.forEach(func(lbl string, c dedicatedColumn) {
			idx, _ := pq.GetColumnIndexByPath(pf, c.ColumnPath)
			if idx == -1 {
				return
			}

			specialAttrIdxs[idx] = lbl
		})

		return nil
	}

	// resource
	if scope == traceql.AttributeScopeNone || scope == traceql.AttributeScopeResource {
		columnMapping := dedicatedColumnsToColumnMapping(dc, backend.DedicatedColumnScopeResource)
		err := addToIndexes(FieldResourceAttrKey, traceqlResourceLabelMappings, columnMapping)
		if err != nil {
			return err
		}
	}
	// span
	if scope == traceql.AttributeScopeNone || scope == traceql.AttributeScopeSpan {
		columnMapping := dedicatedColumnsToColumnMapping(dc, backend.DedicatedColumnScopeSpan)
		err := addToIndexes(FieldSpanAttrKey, traceqlSpanLabelMappings, columnMapping)
		if err != nil {
			return err
		}
	}

	// now search all row groups
	var err error
	rgs := pf.RowGroups()
	for _, rg := range rgs {
		// search all special attributes
		for idx, lbl := range specialAttrIdxs {
			cc := rg.ColumnChunks()[idx]
			err = func() error {
				pgs := cc.Pages()
				defer pgs.Close()
				for {
					pg, err := pgs.ReadPage()
					if errors.Is(err, io.EOF) || pg == nil {
						break
					}
					if err != nil {
						return err
					}

					stop := func(page parquet.Page) bool {
						defer parquet.Release(page)

						// if a special attribute has any non-null values, include it
						if page.NumNulls() < page.NumValues() {
							cb(lbl)
							delete(specialAttrIdxs, idx) // remove from map so we won't search again
							return true
						}
						return false
					}(pg)
					if stop {
						break
					}
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}

		// search other attributes
		for _, idx := range standardAttrIdxs {
			cc := rg.ColumnChunks()[idx]
			err = func() error {
				pgs := cc.Pages()
				defer pgs.Close()

				// normally we'd loop here calling read page for every page in the column chunk, but
				// there is only one dictionary per column chunk, so just read it from the first page
				// and be done.
				pg, err := pgs.ReadPage()
				if errors.Is(err, io.EOF) || pg == nil {
					return nil
				}
				if err != nil {
					return err
				}

				func(page parquet.Page) {
					defer parquet.Release(page)

					dict := page.Dictionary()
					if dict == nil {
						return
					}

					for i := 0; i < dict.Len(); i++ {
						s := dict.Index(int32(i)).String()
						cb(s)
					}
				}(pg)

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *backendBlock) SearchTagValues(ctx context.Context, tag string, cb common.TagCallback, opts common.SearchOptions) error {
	att, ok := translateTagToAttribute[tag]
	if !ok {
		att = traceql.NewAttribute(tag)
	}

	// Wrap to v2-style
	cb2 := func(v traceql.Static) bool {
		cb(v.EncodeToString(false))
		return false
	}

	return b.SearchTagValuesV2(ctx, att, cb2, opts)
}

func (b *backendBlock) SearchTagValuesV2(ctx context.Context, tag traceql.Attribute, cb common.TagCallbackV2, opts common.SearchOptions) error {
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.SearchTagValuesV2",
		opentracing.Tags{
			"blockID":   b.meta.BlockID,
			"tenantID":  b.meta.TenantID,
			"blockSize": b.meta.Size,
		})
	defer span.Finish()

	pf, rr, err := b.openForSearch(derivedCtx, opts)
	if err != nil {
		return fmt.Errorf("unexpected error opening parquet file: %w", err)
	}
	defer func() { span.SetTag("inspectedBytes", rr.BytesRead()) }()

	return searchTagValues(derivedCtx, tag, cb, pf, b.meta.DedicatedColumns)
}

func searchTagValues(ctx context.Context, tag traceql.Attribute, cb common.TagCallbackV2, pf *parquet.File, dc backend.DedicatedColumns) error {
	// Special handling for intrinsics
	if tag.Intrinsic != traceql.IntrinsicNone {
		lookup := intrinsicColumnLookups[tag.Intrinsic]
		if lookup.columnPath != "" {
			err := searchSpecialTagValues(ctx, lookup.columnPath, pf, cb)
			if err != nil {
				return fmt.Errorf("unexpected error searching special tags: %w", err)
			}
		}
		return nil
	}

	// Special handling for weird non-traceql things
	if columnPath := nonTraceQLAttributes[tag.Name]; columnPath != "" {
		err := searchSpecialTagValues(ctx, columnPath, pf, cb)
		if err != nil {
			return fmt.Errorf("unexpected error searching special tags: %s %w", columnPath, err)
		}
		return nil
	}

	// Search well-known attribute column if one exists and is a compatible scope.
	column := wellKnownColumnLookups[tag.Name]
	if column.columnPath != "" && (tag.Scope == column.level || tag.Scope == traceql.AttributeScopeNone) {
		err := searchSpecialTagValues(ctx, column.columnPath, pf, cb)
		if err != nil {
			return fmt.Errorf("unexpected error searching special tags: %w", err)
		}
	}

	// Search dynamic dedicated attribute columns
	if tag.Scope == traceql.AttributeScopeResource || tag.Scope == traceql.AttributeScopeNone {
		resourceColumnMapping := dedicatedColumnsToColumnMapping(dc, backend.DedicatedColumnScopeResource)
		if c, ok := resourceColumnMapping.get(tag.Name); ok {
			err := searchSpecialTagValues(ctx, c.ColumnPath, pf, cb)
			if err != nil {
				return fmt.Errorf("unexpected error searching special tags: %w", err)
			}
		}
	}
	if tag.Scope == traceql.AttributeScopeSpan || tag.Scope == traceql.AttributeScopeNone {
		spanColumnMapping := dedicatedColumnsToColumnMapping(dc, backend.DedicatedColumnScopeSpan)
		if c, ok := spanColumnMapping.get(tag.Name); ok {
			err := searchSpecialTagValues(ctx, c.ColumnPath, pf, cb)
			if err != nil {
				return fmt.Errorf("unexpected error searching special tags: %w", err)
			}
		}
	}

	// Finally also search generic key/values
	err := searchStandardTagValues(ctx, tag, pf, cb)
	if err != nil {
		return fmt.Errorf("unexpected error searching standard tags: %w", err)
	}

	return nil
}

// searchStandardTagValues searches a parquet file for "standard" tags. i.e. tags that don't have unique
// columns and are contained in labelMappings
func searchStandardTagValues(ctx context.Context, tag traceql.Attribute, pf *parquet.File, cb common.TagCallbackV2) error {
	rgs := pf.RowGroups()
	makeIter := makeIterFunc(ctx, rgs, pf)

	keyPred := pq.NewStringInPredicate([]string{tag.Name})

	if tag.Scope == traceql.AttributeScopeNone || tag.Scope == traceql.AttributeScopeResource {
		err := searchKeyValues(DefinitionLevelResourceAttrs,
			FieldResourceAttrKey,
			FieldResourceAttrVal,
			FieldResourceAttrValInt,
			FieldResourceAttrValDouble,
			FieldResourceAttrValBool,
			makeIter, keyPred, cb)
		if err != nil {
			return fmt.Errorf("search resource key values: %w", err)
		}
	}

	if tag.Scope == traceql.AttributeScopeNone || tag.Scope == traceql.AttributeScopeSpan {
		err := searchKeyValues(DefinitionLevelResourceSpansILSSpanAttrs,
			FieldSpanAttrKey,
			FieldSpanAttrVal,
			FieldSpanAttrValInt,
			FieldSpanAttrValDouble,
			FieldSpanAttrValBool,
			makeIter, keyPred, cb)
		if err != nil {
			return fmt.Errorf("search span key values: %w", err)
		}
	}

	return nil
}

func searchKeyValues(definitionLevel int, keyPath, stringPath, intPath, floatPath, boolPath string, makeIter makeIterFn, keyPred pq.Predicate, cb common.TagCallbackV2) error {
	skipNils := pq.NewSkipNilsPredicate()

	iter := pq.NewLeftJoinIterator(definitionLevel,
		// This is required
		[]pq.Iterator{makeIter(keyPath, keyPred, "")},
		[]pq.Iterator{
			// These are optional and we find matching values of all types
			makeIter(stringPath, skipNils, "string"),
			makeIter(intPath, skipNils, "int"),
			makeIter(floatPath, skipNils, "float"),
			makeIter(boolPath, skipNils, "bool"),
		}, nil)
	defer iter.Close()

	for {
		match, err := iter.Next()
		if err != nil {
			return err
		}
		if match == nil {
			break
		}
		for _, e := range match.Entries {
			if callback(cb, e.Value) {
				// Stop
				return nil
			}
		}
	}

	return nil
}

// searchSpecialTagValues searches a parquet file for all values for the provided column. It first attempts
// to only pull all values from the column's dictionary. If this fails it falls back to scanning the entire path.
func searchSpecialTagValues(ctx context.Context, column string, pf *parquet.File, cb common.TagCallbackV2) error {
	pred := newReportValuesPredicate(cb)
	rgs := pf.RowGroups()

	iter := makeIterFunc(ctx, rgs, pf)(column, pred, "")
	defer iter.Close()
	for {
		match, err := iter.Next()
		if err != nil {
			return fmt.Errorf("iter.Next failed: %w", err)
		}
		if match == nil {
			break
		}
	}

	return nil
}
