package vparquet5

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/tempo/pkg/parquetquery"
	"github.com/grafana/tempo/pkg/traceql"
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

	iter, err := fetchSpansOnly(ctx, req, pf, rgs, b.meta.DedicatedColumns)
	if err != nil {
		return traceql.FetchSpansOnlyResponse{}, fmt.Errorf("creating fetch iter: %w", err)
	}

	return traceql.FetchSpansOnlyResponse{
		Results: iter,
		Bytes:   func() uint64 { return rr.BytesRead() },
	}, nil
}

type spanOnlyIterator struct {
	iter parquetquery.Iterator
	sp   *span
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

	return i.sp, nil
}

func (i *spanOnlyIterator) Close() {
	i.iter.Close()
}

func fetchSpansOnly(ctx context.Context, req traceql.FetchSpansRequest, pf *parquet.File, rgs []parquet.RowGroup, dedicatedColumns backend.DedicatedColumns) (*spanOnlyIterator, error) {
	makeIter := makeIterFuncNoIntern(ctx, rgs, pf)

	iter, span, err := createSpanOnlyIterator(makeIter, req.Conditions, req.AllConditions, dedicatedColumns)
	if err != nil {
		return nil, err
	}
	return &spanOnlyIterator{
		iter: iter,
		sp:   span,
	}, nil
}

func createSpanOnlyIterator(makeIter makeIterFn, conditions []traceql.Condition, allConditions bool, dedicatedColumns backend.DedicatedColumns) (parquetquery.Iterator, *span, error) {
	var (
		columnSelectAs   = map[string]string{}
		columnPredicates = map[string][]parquetquery.Predicate{}
		iters            []parquetquery.Iterator
	)

	// todo: improve these methods. if addPredicate gets a nil predicate shouldn't it just wipe out the existing predicates instead of appending?
	// nil predicate matches everything. what's the point of also evaluating a "real" predicate?
	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
	}

	for _, cond := range conditions {
		// Intrinsic?
		switch cond.Attribute.Intrinsic {
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
		}
	}

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, orIfNeeded(predicates), columnSelectAs[columnPath]))
	}

	var required []parquetquery.Iterator
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

	spanCol := &spanCollector2{}

	// This is an optimization for when all of the span conditions must be met.
	// We simply move all iterators into the required list.
	if allConditions {
		required = append(iters, required...)
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
		var pred parquetquery.Predicate
		/*if sampler != nil {
			pred = newSamplingPredicate(sampler, nil)
		}*/
		required = []parquetquery.Iterator{makeIter(columnPathSpanStatusCode, pred, "")}
	}

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	iter, err := parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, nil, parquetquery.WithPool(pqSpanPool), parquetquery.WithCollector(spanCol))
	if err != nil {
		return nil, nil, err
	}
	return iter, &spanCol.at, nil
}

type spanCollector2 struct {
	minAttributes int

	at    span
	atRes parquetquery.IteratorResult
}

var _ parquetquery.Collector = (*spanCollector2)(nil)

func (c *spanCollector2) String() string {
	return fmt.Sprintf("spanCollector(%d)", c.minAttributes)
}

func NewSpanCollector2() *spanCollector2 {
	c := &spanCollector2{}
	c.atRes.AppendOtherValue(otherEntrySpanKey, &c.at)
	return c
}

func (c *spanCollector2) Reset(rowNumber parquetquery.RowNumber) {
	c.at.rowNum = rowNumber
	c.atRes.Reset()
	c.atRes.RowNumber = rowNumber
}

func (c *spanCollector2) Collect(res *parquetquery.IteratorResult) {
	sp := &c.at

	c.at.rowNum = res.RowNumber

	for _, e := range res.OtherEntries {
		switch v := e.Value.(type) {
		case traceql.Static:
			sp.addSpanAttr(newSpanAttr(e.Key), v)
		case *event:
			sp.setEventAttrs(v.attrs)
			putEvent(v)
		case *link:
			sp.setLinkAttrs(v.attrs)
			putLink(v)
		}
	}

	// var durationNanos uint64

	// Merge all individual columns into the span
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathSpanStartTime:
			sp.startTimeUnixNanos = kv.Value.Uint64()
		}
	}
}

func (c *spanCollector2) Result() *parquetquery.IteratorResult {
	return &c.atRes
}
