package vparquet

import (
	"context"
	"fmt"
	"io"

	pq "github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/segmentio/parquet-go"
)

func (b *backendBlock) SearchTags(ctx context.Context, cb common.TagCallback, opts common.SearchOptions) error {
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
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	return searchTags(derivedCtx, cb, pf)
}

func searchTags(_ context.Context, cb common.TagCallback, pf *parquet.File) error {
	// find indexes of generic attribute columns
	resourceKeyIdx, _ := pq.GetColumnIndexByPath(pf, FieldResourceAttrKey)
	spanKeyIdx, _ := pq.GetColumnIndexByPath(pf, FieldSpanAttrKey)
	if resourceKeyIdx == -1 || spanKeyIdx == -1 {
		return fmt.Errorf("resource or span attributes col not found (%d, %d)", resourceKeyIdx, spanKeyIdx)
	}
	standardAttrIdxs := []int{
		resourceKeyIdx,
		spanKeyIdx,
	}

	// find indexes of all special columns
	specialAttrIdxs := map[int]string{}
	for lbl, col := range labelMappings {
		idx, _ := pq.GetColumnIndexByPath(pf, col)
		if idx == -1 {
			continue
		}

		specialAttrIdxs[idx] = lbl
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
					if err == io.EOF || pg == nil {
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
				for {
					pg, err := pgs.ReadPage()
					if err == io.EOF || pg == nil {
						break
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
				}
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
	span, derivedCtx := opentracing.StartSpanFromContext(ctx, "parquet.backendBlock.SearchTagValues",
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
	defer func() { span.SetTag("inspectedBytes", rr.TotalBytesRead.Load()) }()

	return searchTagValues(ctx, tag, cb, pf)
}

func searchTagValues(ctx context.Context, tag string, cb common.TagCallback, pf *parquet.File) error {
	// labelMappings will indicate whether this is a search for a special or standard
	// column
	column := labelMappings[tag]
	if column == "" {
		err := searchStandardTagValues(ctx, tag, pf, cb)
		if err != nil {
			return fmt.Errorf("unexpected error searching standard tags: %w", err)
		}
		return nil
	}

	err := searchSpecialTagValues(ctx, column, pf, cb)
	if err != nil {
		return fmt.Errorf("unexpected error searching special tags: %w", err)
	}
	return nil
}

// searchStandardTagValues searches a parquet file for "standard" tags. i.e. tags that don't have unique
// columns and are contained in labelMappings
func searchStandardTagValues(ctx context.Context, tag string, pf *parquet.File, cb common.TagCallback) error {
	rgs := pf.RowGroups()
	makeIter := makeIterFunc(ctx, rgs, pf)

	keyPred := pq.NewStringInPredicate([]string{tag})

	err := searchKeyValues(DefinitionLevelResourceAttrs, FieldResourceAttrKey, FieldResourceAttrVal, makeIter, keyPred, cb)
	if err != nil {
		return errors.Wrap(err, "search resource key values")
	}

	err = searchKeyValues(DefinitionLevelResourceSpansILSSpanAttrs, FieldSpanAttrKey, FieldSpanAttrVal, makeIter, keyPred, cb)
	if err != nil {
		return errors.Wrap(err, "search span key values")
	}

	return nil
}

func searchKeyValues(definitionLevel int, keyPath, valuePath string, makeIter makeIterFn, keyPred pq.Predicate, cb common.TagCallback) error {

	iter := pq.NewJoinIterator(definitionLevel, []pq.Iterator{
		makeIter(keyPath, keyPred, ""),
		makeIter(valuePath, nil, "values"),
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
			// We know that "values" is the only data selected above.
			cb(e.Value.String())
		}
	}

	return nil
}

// searchSpecialTagValues searches a parquet file for all values for the provided column. It first attempts
// to only pull all values from the column's dictionary. If this fails it falls back to scanning the entire path.
func searchSpecialTagValues(ctx context.Context, column string, pf *parquet.File, cb common.TagCallback) error {
	pred := newReportValuesPredicate(cb)
	rgs := pf.RowGroups()

	iter := makeIterFunc(ctx, rgs, pf)(column, pred, "")
	defer iter.Close()
	for {
		match, err := iter.Next()
		if err != nil {
			return errors.Wrap(err, "iter.Next failed")
		}
		if match == nil {
			break
		}
	}

	return nil
}
