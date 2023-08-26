package vparquet

import (
	"context"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/pkg/parquetquery"
	pq "github.com/grafana/tempo/pkg/parquetquery"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// span implements traceql.Span
type span struct {
	attributes         map[traceql.Attribute]traceql.Static
	id                 []byte
	startTimeUnixNanos uint64
	endtimeUnixNanos   uint64

	// metadata used to track the span in the parquet file
	rowNum         parquetquery.RowNumber
	cbSpansetFinal bool
	cbSpanset      *traceql.Spanset
}

func (s *span) Attributes() map[traceql.Attribute]traceql.Static {
	return s.attributes
}

func (s *span) ID() []byte {
	return s.id
}

func (s *span) StartTimeUnixNanos() uint64 {
	return s.startTimeUnixNanos
}

func (s *span) DurationNanos() uint64 {
	return s.endtimeUnixNanos - s.startTimeUnixNanos
}

func (*span) DescendantOf(traceql.Span) bool { return false }
func (*span) SiblingOf(traceql.Span) bool    { return false }
func (*span) ChildOf(traceql.Span) bool      { return false }

// attributesMatched counts all attributes in the map as well as metadata fields like start/end/id
func (s *span) attributesMatched() int {
	count := 0
	for _, v := range s.attributes {
		if v.Type != traceql.TypeNil {
			count++
		}
	}
	if s.endtimeUnixNanos != 0 {
		count++
	}
	if s.startTimeUnixNanos != 0 {
		count++
	}
	if len(s.id) > 0 {
		count++
	}

	return count
}

// todo: this sync pool currently massively reduces allocations by pooling spans for certain queries.
// it currently catches spans discarded:
// - in the span collector
// - in the batch collector
// - while converting to spanmeta
// to be fully effective it needs to catch spans thrown away in the query engine. perhaps filter spans
// can return a slice of dropped and kept spansets?
var spanPool = sync.Pool{
	New: func() interface{} {
		return &span{
			attributes: make(map[traceql.Attribute]traceql.Static),
		}
	},
}

func putSpan(s *span) {
	s.id = nil
	s.endtimeUnixNanos = 0
	s.startTimeUnixNanos = 0
	s.rowNum = parquetquery.EmptyRowNumber()
	s.cbSpansetFinal = false
	s.cbSpanset = nil

	// clear attributes
	for k := range s.attributes {
		delete(s.attributes, k)
	}

	spanPool.Put(s)
}

func getSpan() *span {
	return spanPool.Get().(*span)
}

// Helper function to create an iterator, that abstracts away
// context like file and rowgroups.
type makeIterFn func(columnName string, predicate parquetquery.Predicate, selectAs string) parquetquery.Iterator

const (
	columnPathTraceID                  = "TraceID"
	columnPathStartTimeUnixNano        = "StartTimeUnixNano"
	columnPathEndTimeUnixNano          = "EndTimeUnixNano"
	columnPathDurationNanos            = "DurationNanos"
	columnPathRootSpanName             = "RootSpanName"
	columnPathRootServiceName          = "RootServiceName"
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

	columnPathSpanID        = "rs.ils.Spans.ID"
	columnPathSpanName      = "rs.ils.Spans.Name"
	columnPathSpanStartTime = "rs.ils.Spans.StartUnixNanos"
	columnPathSpanEndTime   = "rs.ils.Spans.EndUnixNanos"
	columnPathSpanKind      = "rs.ils.Spans.Kind"
	// columnPathSpanDuration       = "rs.ils.Spans.DurationNanos"
	columnPathSpanStatusCode     = "rs.ils.Spans.StatusCode"
	columnPathSpanStatusMessage  = "rs.ils.Spans.StatusMessage"
	columnPathSpanAttrKey        = "rs.ils.Spans.Attrs.Key"
	columnPathSpanAttrString     = "rs.ils.Spans.Attrs.Value"
	columnPathSpanAttrInt        = "rs.ils.Spans.Attrs.ValueInt"
	columnPathSpanAttrDouble     = "rs.ils.Spans.Attrs.ValueDouble"
	columnPathSpanAttrBool       = "rs.ils.Spans.Attrs.ValueBool"
	columnPathSpanHTTPStatusCode = "rs.ils.Spans.HttpStatusCode"
	columnPathSpanHTTPMethod     = "rs.ils.Spans.HttpMethod"
	columnPathSpanHTTPURL        = "rs.ils.Spans.HttpUrl"

	otherEntrySpansetKey = "spanset"
	otherEntrySpanKey    = "span"

	// a fake intrinsic scope at the trace lvl
	intrinsicScopeTrace = -1
	intrinsicScopeSpan  = -2
)

var intrinsicColumnLookups = map[traceql.Intrinsic]struct {
	scope      traceql.AttributeScope
	typ        traceql.StaticType
	columnPath string
}{
	traceql.IntrinsicName:          {intrinsicScopeSpan, traceql.TypeString, columnPathSpanName},
	traceql.IntrinsicStatus:        {intrinsicScopeSpan, traceql.TypeStatus, columnPathSpanStatusCode},
	traceql.IntrinsicStatusMessage: {intrinsicScopeSpan, traceql.TypeString, columnPathSpanStatusMessage},
	traceql.IntrinsicDuration:      {intrinsicScopeSpan, traceql.TypeDuration, ""},
	traceql.IntrinsicKind:          {intrinsicScopeSpan, traceql.TypeKind, columnPathSpanKind},
	traceql.IntrinsicSpanID:        {intrinsicScopeSpan, traceql.TypeString, columnPathSpanID},
	traceql.IntrinsicSpanStartTime: {intrinsicScopeSpan, traceql.TypeString, columnPathSpanStartTime},

	traceql.IntrinsicTraceRootService: {intrinsicScopeTrace, traceql.TypeString, columnPathRootServiceName},
	traceql.IntrinsicTraceRootSpan:    {intrinsicScopeTrace, traceql.TypeString, columnPathRootSpanName},
	traceql.IntrinsicTraceDuration:    {intrinsicScopeTrace, traceql.TypeString, columnPathDurationNanos},
	traceql.IntrinsicTraceID:          {intrinsicScopeTrace, traceql.TypeDuration, columnPathTraceID},
	traceql.IntrinsicTraceStartTime:   {intrinsicScopeTrace, traceql.TypeDuration, columnPathStartTimeUnixNano},
}

// Lookup table of all well-known attributes with dedicated columns
var wellKnownColumnLookups = map[string]struct {
	columnPath string                 // path.to.column
	level      traceql.AttributeScope // span or resource level
	typ        traceql.StaticType     // Data type
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
	LabelHTTPStatusCode: {columnPathSpanHTTPStatusCode, traceql.AttributeScopeSpan, traceql.TypeInt},
	LabelHTTPMethod:     {columnPathSpanHTTPMethod, traceql.AttributeScopeSpan, traceql.TypeString},
	LabelHTTPUrl:        {columnPathSpanHTTPURL, traceql.AttributeScopeSpan, traceql.TypeString},
}

// Fetch spansets from the block for the given TraceQL FetchSpansRequest. The request is checked for
// internal consistencies:  operand count matches the operation, all operands in each condition are identical
// types, and the operand type is compatible with the operation.
func (b *backendBlock) Fetch(ctx context.Context, req traceql.FetchSpansRequest, opts common.SearchOptions) (traceql.FetchSpansResponse, error) {
	err := checkConditions(req.Conditions)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "conditions invalid")
	}

	pf, rr, err := b.openForSearch(ctx, opts)
	if err != nil {
		return traceql.FetchSpansResponse{}, err
	}

	iter, err := fetch(ctx, req, pf, opts)
	if err != nil {
		return traceql.FetchSpansResponse{}, errors.Wrap(err, "creating fetch iter")
	}

	return traceql.FetchSpansResponse{
		Results: iter,
		Bytes:   func() uint64 { return rr.BytesRead() },
	}, nil
}

func checkConditions(conditions []traceql.Condition) error {
	for _, cond := range conditions {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicStructuralDescendant, traceql.IntrinsicStructuralChild, traceql.IntrinsicStructuralSibling:
			return common.ErrUnsupported
		}

		opCount := len(cond.Operands)

		switch cond.Op {

		case traceql.OpNone:
			if opCount != 0 {
				return fmt.Errorf("operanion none must have 0 arguments. condition: %+v", cond)
			}

		case traceql.OpEqual, traceql.OpNotEqual,
			traceql.OpGreater, traceql.OpGreaterEqual,
			traceql.OpLess, traceql.OpLessEqual,
			traceql.OpRegex, traceql.OpNotRegex:
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

var _ pq.Iterator = (*bridgeIterator)(nil)

// bridgeIterator creates a bridge between one iterator pass and the next
type bridgeIterator struct {
	iter parquetquery.Iterator
	cb   traceql.SecondPassFn

	nextSpans []*span
}

func newBridgeIterator(iter parquetquery.Iterator, cb traceql.SecondPassFn) *bridgeIterator {
	return &bridgeIterator{
		iter: iter,
		cb:   cb,
	}
}

func (i *bridgeIterator) String() string {
	return fmt.Sprintf("bridgeIterator: \n\t%s", util.TabOut(i.iter))
}

func (i *bridgeIterator) Next() (*pq.IteratorResult, error) {
	// drain current buffer
	if len(i.nextSpans) > 0 {
		ret := i.nextSpans[0]
		i.nextSpans = i.nextSpans[1:]
		return spanToIteratorResult(ret), nil
	}

	for {
		res, err := i.iter.Next()
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}

		// The spanset is in the OtherEntries
		iface := res.OtherValueFromKey(otherEntrySpansetKey)
		if iface == nil {
			return nil, fmt.Errorf("engine assumption broken: spanset not found in other entries in bridge")
		}
		spanset, ok := iface.(*traceql.Spanset)
		if !ok {
			return nil, fmt.Errorf("engine assumption broken: spanset is not of type *traceql.Spanset in bridge")
		}

		filteredSpansets, err := i.cb(spanset)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		// if the filter removed all spansets then let's release all back to the pool
		// no reason to try anything more nuanced than this. it will handle nearly all cases
		if len(filteredSpansets) == 0 {
			for _, s := range spanset.Spans {
				putSpan(s.(*span))
			}
		}

		// flatten spans into i.currentSpans
		for _, ss := range filteredSpansets {
			for idx, s := range ss.Spans {
				span := s.(*span)

				// use otherEntryCallbackSpansetKey to indicate to the rebatchIterator that either
				// 1) this is the last span in the spanset, or 2) there are more spans in the spanset
				span.cbSpansetFinal = idx == len(ss.Spans)-1
				span.cbSpanset = ss
				i.nextSpans = append(i.nextSpans, span)
			}
		}

		// spans returned from the filter are not guaranteed to be in file order
		// we need them to be so that the meta iterators work correctly. sort here
		sort.Slice(i.nextSpans, func(j, k int) bool {
			return parquetquery.CompareRowNumbers(DefinitionLevelResourceSpans, i.nextSpans[j].rowNum, i.nextSpans[k].rowNum) == -1
		})

		// found something!
		if len(i.nextSpans) > 0 {
			ret := i.nextSpans[0]
			i.nextSpans = i.nextSpans[1:]
			return spanToIteratorResult(ret), nil
		}
	}
}

func spanToIteratorResult(s *span) *pq.IteratorResult {
	res := &pq.IteratorResult{RowNumber: s.rowNum}
	res.AppendOtherValue(otherEntrySpanKey, s)

	return res
}

func (i *bridgeIterator) SeekTo(to pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	var at *pq.IteratorResult

	for at, _ = i.Next(); i != nil && at != nil && pq.CompareRowNumbers(definitionLevel, at.RowNumber, to) < 0; {
		at, _ = i.Next()
	}

	return at, nil
}

func (i *bridgeIterator) Close() {
	i.iter.Close()
}

// confirm rebatchIterator implements pq.Iterator
var _ pq.Iterator = (*rebatchIterator)(nil)

// rebatchIterator either passes spansets through directly OR rebatches them based on metadata
// in OtherEntries
type rebatchIterator struct {
	iter parquetquery.Iterator

	nextSpans []*span
}

func newRebatchIterator(iter parquetquery.Iterator) *rebatchIterator {
	return &rebatchIterator{
		iter: iter,
	}
}

func (i *rebatchIterator) String() string {
	return fmt.Sprintf("rebatchIterator: \n\t%s", util.TabOut(i.iter))
}

// Next has to handle two different style results. First is an initial set of spans
// that does not have a callback spanset. These can be passed directly through.
// Second is a set of spans that have spansets imposed by the callback (i.e. for grouping)
// these must be regrouped into the callback spansets
func (i *rebatchIterator) Next() (*pq.IteratorResult, error) {
	for {
		// see if we have a queue
		res := i.resultFromNextSpans()
		if res != nil {
			return res, nil
		}

		// check the iterator for anything
		res, err := i.iter.Next()
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}

		// get the spanset and see if we should pass it through or buffer for rebatching
		iface := res.OtherValueFromKey(otherEntrySpansetKey)
		if iface == nil {
			return nil, fmt.Errorf("engine assumption broken: spanset not found in other entries in rebatch")
		}
		ss, ok := iface.(*traceql.Spanset)
		if !ok {
			return nil, fmt.Errorf("engine assumption broken: spanset is not of type *traceql.Spanset in rebatch")
		}

		// if this has no call back spanset just pass it on
		if len(ss.Spans) > 0 && ss.Spans[0].(*span).cbSpanset == nil {
			return res, nil
		}

		// dump all spans into our buffer
		for _, s := range ss.Spans {
			sp := s.(*span)
			if !sp.cbSpansetFinal {
				continue
			}

			// copy trace level data from the current iteration spanset into the rebatch spanset. only do this if
			// we don't have current data
			if sp.cbSpanset.DurationNanos == 0 {
				sp.cbSpanset.DurationNanos = ss.DurationNanos
			}
			if len(sp.cbSpanset.TraceID) == 0 {
				sp.cbSpanset.TraceID = ss.TraceID
			}
			if len(sp.cbSpanset.RootSpanName) == 0 {
				sp.cbSpanset.RootSpanName = ss.RootSpanName
			}
			if len(sp.cbSpanset.RootServiceName) == 0 {
				sp.cbSpanset.RootServiceName = ss.RootServiceName
			}
			if sp.cbSpanset.StartTimeUnixNanos == 0 {
				sp.cbSpanset.StartTimeUnixNanos = ss.StartTimeUnixNanos
			}

			i.nextSpans = append(i.nextSpans, sp)
		}

		res = i.resultFromNextSpans()
		if res != nil {
			return res, nil
		}
		// if we don't find anything in that spanset, start over
	}
}

func (i *rebatchIterator) resultFromNextSpans() *pq.IteratorResult {
	for len(i.nextSpans) > 0 {
		ret := i.nextSpans[0]
		i.nextSpans = i.nextSpans[1:]

		if ret.cbSpansetFinal && ret.cbSpanset != nil {
			res := &pq.IteratorResult{}
			res.AppendOtherValue(otherEntrySpansetKey, ret.cbSpanset)
			return res
		}
	}

	return nil
}

func (i *rebatchIterator) SeekTo(to pq.RowNumber, definitionLevel int) (*pq.IteratorResult, error) {
	return i.iter.SeekTo(to, definitionLevel)
}

func (i *rebatchIterator) Close() {
	i.iter.Close()
}

// spansetIterator turns the parquet iterator into the final
// traceql iterator.  Every row it receives is one spanset.
type spansetIterator struct {
	iter parquetquery.Iterator
}

var _ traceql.SpansetIterator = (*spansetIterator)(nil)

func newSpansetIterator(iter parquetquery.Iterator) *spansetIterator {
	return &spansetIterator{
		iter: iter,
	}
}

func (i *spansetIterator) Next(context.Context) (*traceql.Spanset, error) {
	res, err := i.iter.Next()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// The spanset is in the OtherEntries
	iface := res.OtherValueFromKey(otherEntrySpansetKey)
	if iface == nil {
		return nil, fmt.Errorf("engine assumption broken: spanset not found in other entries in spansetIterator")
	}
	ss, ok := iface.(*traceql.Spanset)
	if !ok {
		return nil, fmt.Errorf("engine assumption broken: spanset is not of type *traceql.Spanset in spansetIterator")
	}

	return ss, nil
}

func (i *spansetIterator) Close() {
	i.iter.Close()
}

// mergeSpansetIterator iterates through a slice of spansetIterators exhausting them
// in order
type mergeSpansetIterator struct {
	iters []traceql.SpansetIterator
}

var _ traceql.SpansetIterator = (*mergeSpansetIterator)(nil)

func (i *mergeSpansetIterator) Next(ctx context.Context) (*traceql.Spanset, error) {
	for len(i.iters) > 0 {
		spanset, err := i.iters[0].Next(ctx)
		if err != nil {
			return nil, err
		}
		if spanset == nil {
			// This iter is exhausted, pop it
			i.iters[0].Close()
			i.iters = i.iters[1:]
			continue
		}
		return spanset, nil
	}

	return nil, nil
}

func (i *mergeSpansetIterator) Close() {
	// Close any outstanding iters
	for _, iter := range i.iters {
		iter.Close()
	}
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

func fetch(ctx context.Context, req traceql.FetchSpansRequest, pf *parquet.File, opts common.SearchOptions) (*spansetIterator, error) {
	iter, err := createAllIterator(ctx, nil, req.Conditions, req.AllConditions, req.StartTimeUnixNanos, req.EndTimeUnixNanos, pf, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating iterator: %w", err)
	}

	if req.SecondPass != nil {
		iter = newBridgeIterator(newRebatchIterator(iter), req.SecondPass)

		iter, err = createAllIterator(ctx, iter, req.SecondPassConditions, false, 0, 0, pf, opts)
		if err != nil {
			return nil, fmt.Errorf("error creating second pass iterator: %w", err)
		}
	}

	return newSpansetIterator(newRebatchIterator(iter)), nil
}

func createAllIterator(ctx context.Context, primaryIter parquetquery.Iterator, conds []traceql.Condition, allConditions bool, start, end uint64, pf *parquet.File, opts common.SearchOptions) (parquetquery.Iterator, error) {
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
		batchRequireAtLeastOneMatchOverall = len(conds) > 0 && len(traceConditions) == 0
	)

	// Optimization for queries like {resource.x... && span.y ...}
	// Requires no mingled scopes like .foo=x, which could be satisfied
	// one either resource or span.
	allConditions = allConditions && !mingledConditions

	spanIter, err := createSpanIterator(makeIter, primaryIter, spanConditions, spanRequireAtLeastOneMatch, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span iterator")
	}

	resourceIter, err := createResourceIterator(makeIter, spanIter, resourceConditions, batchRequireAtLeastOneMatch, batchRequireAtLeastOneMatchOverall, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating resource iterator")
	}

	return createTraceIterator(makeIter, resourceIter, traceConditions, start, end), nil
}

// createSpanIterator iterates through all span-level columns, groups them into rows representing
// one span each.  Spans are returned that match any of the given conditions.
func createSpanIterator(makeIter makeIterFn, primaryIter parquetquery.Iterator, conditions []traceql.Condition, requireAtLeastOneMatch, allConditions bool) (parquetquery.Iterator, error) {
	var (
		columnSelectAs     = map[string]string{}
		columnPredicates   = map[string][]parquetquery.Predicate{}
		iters              []parquetquery.Iterator
		genericConditions  []traceql.Condition
		durationPredicates []*parquetquery.GenericPredicate[int64]
	)

	addPredicate := func(columnPath string, p parquetquery.Predicate) {
		columnPredicates[columnPath] = append(columnPredicates[columnPath], p)
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
			// may have been added via duration
			if _, ok := columnSelectAs[columnPathSpanStartTime]; ok {
				continue
			}

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
			if pred, ok := pred.(*parquetquery.GenericPredicate[int64]); ok {
				durationPredicates = append(durationPredicates, pred)
			} else {
				durationPredicates = append(durationPredicates, nil)
			}
			addPredicate(columnPathSpanEndTime, nil)
			columnSelectAs[columnPathSpanEndTime] = columnPathSpanEndTime

			// may have been added above
			if _, ok := columnSelectAs[columnPathSpanStartTime]; ok {
				continue
			}
			addPredicate(columnPathSpanStartTime, nil)
			columnSelectAs[columnPathSpanStartTime] = columnPathSpanStartTime
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

	minCount := 0
	if requireAtLeastOneMatch {
		minCount = 1
	}
	if allConditions {
		// The final number of expected attributes.
		distinct := map[string]struct{}{}
		for _, cond := range conditions {
			distinct[cond.Attribute.Name] = struct{}{}
		}
		minCount = len(distinct)
	}
	spanCol := &spanCollector{
		minCount,
		durationPredicates,
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
	if requireAtLeastOneMatch && len(iters) > 0 && len(durationPredicates) == 0 {
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

	// Left join here means the span id/start/end iterators + 1 are required,
	// and all other conditions are optional. Whatever matches is returned.
	return parquetquery.NewLeftJoinIterator(DefinitionLevelResourceSpansILSSpan, required, iters, spanCol), nil
}

// createResourceIterator iterates through all resourcespans-level (batch-level) columns, groups them into rows representing
// one batch each. It builds on top of the span iterator, and turns the groups of spans and resource-level values into
// spansets.  Spansets are returned that match any of the given conditions.
func createResourceIterator(makeIter makeIterFn, spanIterator parquetquery.Iterator, conditions []traceql.Condition, requireAtLeastOneMatch, requireAtLeastOneMatchOverall, allConditions bool) (parquetquery.Iterator, error) {
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

	for columnPath, predicates := range columnPredicates {
		iters = append(iters, makeIter(columnPath, parquetquery.NewOrPredicate(predicates...), columnSelectAs[columnPath]))
	}

	attrIter, err := createAttributeIterator(makeIter, genericConditions, DefinitionLevelResourceAttrs,
		columnPathResourceAttrKey, columnPathResourceAttrString, columnPathResourceAttrInt, columnPathResourceAttrDouble, columnPathResourceAttrBool, allConditions)
	if err != nil {
		return nil, errors.Wrap(err, "creating span attribute iterator")
	}
	if attrIter != nil {
		iters = append(iters, attrIter)
	}

	minCount := 0
	if requireAtLeastOneMatch {
		minCount = 1
	}
	if allConditions {
		// The final number of expected attributes
		distinct := map[string]struct{}{}
		for _, cond := range conditions {
			distinct[cond.Attribute.Name] = struct{}{}
		}
		minCount = len(distinct)
	}
	batchCol := &batchCollector{
		requireAtLeastOneMatchOverall: requireAtLeastOneMatchOverall,
		minAttributes:                 minCount,
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

func createTraceIterator(makeIter makeIterFn, resourceIter parquetquery.Iterator, conds []traceql.Condition, start, end uint64) parquetquery.Iterator {
	traceIters := make([]parquetquery.Iterator, 0, 3)

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

	for _, cond := range conds {
		switch cond.Attribute.Intrinsic {
		case traceql.IntrinsicTraceID:
			traceIters = append(traceIters, makeIter(columnPathTraceID, nil, columnPathTraceID))
		case traceql.IntrinsicTraceDuration:
			traceIters = append(traceIters, makeIter(columnPathDurationNanos, nil, columnPathDurationNanos))
		case traceql.IntrinsicTraceStartTime:
			if start == 0 && end == 0 {
				traceIters = append(traceIters, makeIter(columnPathStartTimeUnixNano, nil, columnPathStartTimeUnixNano))
			}
		case traceql.IntrinsicTraceRootSpan:
			traceIters = append(traceIters, makeIter(columnPathRootSpanName, nil, columnPathRootSpanName))
		case traceql.IntrinsicTraceRootService:
			traceIters = append(traceIters, makeIter(columnPathRootServiceName, nil, columnPathRootServiceName))
		}
	}

	// Final trace iterator
	// Join iterator means it requires matching resources to have been found
	// TraceCollor adds trace-level data to the spansets
	return parquetquery.NewJoinIterator(DefinitionLevelTrace, traceIters, &traceCollector{})
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

	for _, op := range operands {
		if op.Type != traceql.TypeString {
			return nil, fmt.Errorf("operand is not string: %+v", op)
		}
	}

	s := operands[0].S

	switch op {
	case traceql.OpNotEqual:
		return parquetquery.NewGenericPredicate(
			func(v string) bool {
				return v != s
			},
			func(min, max string) bool {
				return min != s || max != s
			},
			func(v parquet.Value) string {
				return v.String()
			},
		), nil

	case traceql.OpRegex:
		return parquetquery.NewRegexInPredicate([]string{s})
	case traceql.OpNotRegex:
		return parquetquery.NewRegexNotInPredicate([]string{s})
	case traceql.OpEqual:
		return parquetquery.NewStringInPredicate([]string{s}), nil
	case traceql.OpGreater:
		return parquetquery.NewGenericPredicate(
			func(v string) bool {
				return strings.Compare(v, s) > 0
			},
			func(min, max string) bool {
				return strings.Compare(max, s) > 0
			},
			func(v parquet.Value) string {
				return v.String()
			},
		), nil
	case traceql.OpGreaterEqual:
		return parquetquery.NewGenericPredicate(
			func(v string) bool {
				return strings.Compare(v, s) >= 0
			},
			func(min, max string) bool {
				return strings.Compare(max, s) >= 0
			},
			func(v parquet.Value) string {
				return v.String()
			},
		), nil
	case traceql.OpLess:
		return parquetquery.NewGenericPredicate(
			func(v string) bool {
				return strings.Compare(v, s) < 0
			},
			func(min, max string) bool {
				return strings.Compare(min, s) < 0
			},
			func(v parquet.Value) string {
				return v.String()
			},
		), nil
	case traceql.OpLessEqual:
		return parquetquery.NewGenericPredicate(
			func(v string) bool {
				return strings.Compare(v, s) <= 0
			},
			func(min, max string) bool {
				return strings.Compare(min, s) <= 0
			},
			func(v parquet.Value) string {
				return v.String()
			},
		), nil
	default:
		return nil, fmt.Errorf("operand not supported for strings: %+v", op)
	}
}

func createIntPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	var i int64
	switch operands[0].Type {
	case traceql.TypeInt:
		i = int64(operands[0].N)
	case traceql.TypeDuration:
		i = operands[0].D.Nanoseconds()
	case traceql.TypeStatus:
		i = int64(StatusCodeMapping[operands[0].Status.String()])
	case traceql.TypeKind:
		i = int64(KindMapping[operands[0].Kind.String()])
	default:
		return nil, fmt.Errorf("operand is not int, duration, status or kind: %+v", operands[0])
	}

	var fn func(v int64) bool
	var rangeFn func(min, max int64) bool

	switch op {
	case traceql.OpEqual:
		fn = func(v int64) bool { return v == i }
		rangeFn = func(min, max int64) bool { return min <= i && i <= max }
	case traceql.OpNotEqual:
		fn = func(v int64) bool { return v != i }
		rangeFn = func(min, max int64) bool { return min != i || max != i }
	case traceql.OpGreater:
		fn = func(v int64) bool { return v > i }
		rangeFn = func(min, max int64) bool { return max > i }
	case traceql.OpGreaterEqual:
		fn = func(v int64) bool { return v >= i }
		rangeFn = func(min, max int64) bool { return max >= i }
	case traceql.OpLess:
		fn = func(v int64) bool { return v < i }
		rangeFn = func(min, max int64) bool { return min < i }
	case traceql.OpLessEqual:
		fn = func(v int64) bool { return v <= i }
		rangeFn = func(min, max int64) bool { return min <= i }
	default:
		return nil, fmt.Errorf("operand not supported for integers: %+v", op)
	}

	return parquetquery.NewIntPredicate(fn, rangeFn), nil
}

func createFloatPredicate(op traceql.Operator, operands traceql.Operands) (parquetquery.Predicate, error) {
	if op == traceql.OpNone {
		return nil, nil
	}

	// Ensure operand is float
	if operands[0].Type != traceql.TypeFloat {
		return nil, fmt.Errorf("operand is not float: %+v", operands[0])
	}

	i := operands[0].F

	var fn func(v float64) bool
	var rangeFn func(min, max float64) bool

	switch op {
	case traceql.OpEqual:
		fn = func(v float64) bool { return v == i }
		rangeFn = func(min, max float64) bool { return min <= i && i <= max }
	case traceql.OpNotEqual:
		fn = func(v float64) bool { return v != i }
		rangeFn = func(min, max float64) bool { return min != i || max != i }
	case traceql.OpGreater:
		fn = func(v float64) bool { return v > i }
		rangeFn = func(min, max float64) bool { return max > i }
	case traceql.OpGreaterEqual:
		fn = func(v float64) bool { return v >= i }
		rangeFn = func(min, max float64) bool { return max >= i }
	case traceql.OpLess:
		fn = func(v float64) bool { return v < i }
		rangeFn = func(min, max float64) bool { return min < i }
	case traceql.OpLessEqual:
		fn = func(v float64) bool { return v <= i }
		rangeFn = func(min, max float64) bool { return min <= i }
	default:
		return nil, fmt.Errorf("operand not supported for floats: %+v", op)
	}

	return parquetquery.NewFloatPredicate(fn, rangeFn), nil
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

	case traceql.OpNotEqual:
		return parquetquery.NewBoolPredicate(!operands[0].B), nil

	default:
		return nil, fmt.Errorf("operand not supported for booleans: %+v", op)
	}
}

func createAttributeIterator(makeIter makeIterFn, conditions []traceql.Condition,
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
				&attributeCollector{}), nil
		}

		return parquetquery.NewLeftJoinIterator(definitionLevel,
			[]parquetquery.Iterator{makeIter(keyPath, parquetquery.NewStringInPredicate(attrKeys), "key")},
			valueIters,
			&attributeCollector{}), nil
	}

	return nil, nil
}

// This turns groups of span values into Span objects
type spanCollector struct {
	minAttributes   int
	durationFilters []*parquetquery.GenericPredicate[int64]
}

var _ parquetquery.GroupPredicate = (*spanCollector)(nil)

func (c *spanCollector) String() string {
	return fmt.Sprintf("spanCollector(%d, %v)", c.minAttributes, c.durationFilters)
}

func (c *spanCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	var sp *span

	// look for existing span first. this occurs on the second pass
	for _, e := range res.OtherEntries {
		if e.Key == otherEntrySpanKey {
			sp = e.Value.(*span)
			break
		}
	}

	// if not found create a new one
	if sp == nil {
		sp = getSpan()
		sp.rowNum = res.RowNumber
	}

	for _, e := range res.OtherEntries {
		if e.Key == otherEntrySpanKey {
			continue
		}
		sp.attributes[newSpanAttr(e.Key)] = e.Value.(traceql.Static)
	}

	var startTimeUnixNanos, endTimeUnixNanos uint64

	// Merge all individual columns into the span
	for _, kv := range res.Entries {
		switch kv.Key {
		case columnPathSpanID:
			sp.id = kv.Value.ByteArray()
		case columnPathSpanStartTime:
			startTimeUnixNanos = kv.Value.Uint64()
			sp.startTimeUnixNanos = startTimeUnixNanos
		case columnPathSpanEndTime:
			endTimeUnixNanos = kv.Value.Uint64()
			sp.endtimeUnixNanos = endTimeUnixNanos
		case columnPathSpanName:
			sp.attributes[traceql.NewIntrinsic(traceql.IntrinsicName)] = traceql.NewStaticString(kv.Value.String())
		// case columnPathSpanDuration:
		// 	span.Attributes[traceql.NewIntrinsic(traceql.IntrinsicDuration)] = traceql.NewStaticDuration(time.Duration(kv.Value.Uint64()))
		case columnPathSpanStatusCode:
			// Map OTLP status code back to TraceQL enum.
			// For other values, use the raw integer.
			var status traceql.Status
			switch kv.Value.Uint64() {
			case uint64(v1.Status_STATUS_CODE_UNSET):
				status = traceql.StatusUnset
			case uint64(v1.Status_STATUS_CODE_OK):
				status = traceql.StatusOk
			case uint64(v1.Status_STATUS_CODE_ERROR):
				status = traceql.StatusError
			default:
				status = traceql.Status(kv.Value.Uint64())
			}
			sp.attributes[traceql.NewIntrinsic(traceql.IntrinsicStatus)] = traceql.NewStaticStatus(status)
		case columnPathSpanStatusMessage:
			sp.attributes[traceql.NewIntrinsic(traceql.IntrinsicStatusMessage)] = traceql.NewStaticString(kv.Value.String())
		case columnPathSpanKind:
			var kind traceql.Kind
			switch kv.Value.Uint64() {
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
				kind = traceql.Kind(kv.Value.Uint64())
			}
			sp.attributes[traceql.NewIntrinsic(traceql.IntrinsicKind)] = traceql.NewStaticKind(kind)
		default:
			// TODO - This exists for span-level dedicated columns like http.status_code
			// Are nils possible here?
			switch kv.Value.Kind() {
			case parquet.Boolean:
				sp.attributes[newSpanAttr(kv.Key)] = traceql.NewStaticBool(kv.Value.Boolean())
			case parquet.Int32, parquet.Int64:
				sp.attributes[newSpanAttr(kv.Key)] = traceql.NewStaticInt(int(kv.Value.Int64()))
			case parquet.Float:
				sp.attributes[newSpanAttr(kv.Key)] = traceql.NewStaticFloat(kv.Value.Double())
			case parquet.ByteArray:
				sp.attributes[newSpanAttr(kv.Key)] = traceql.NewStaticString(kv.Value.String())
			}
		}
	}

	// Save computed duration if any filters present and at least one is passed.
	if len(c.durationFilters) > 0 {
		duration := endTimeUnixNanos - startTimeUnixNanos
		for _, f := range c.durationFilters {
			if f == nil || f.Fn(int64(duration)) {
				sp.attributes[traceql.NewIntrinsic(traceql.IntrinsicDuration)] = traceql.NewStaticDuration(time.Duration(duration))
				break
			}
		}
	}

	if c.minAttributes > 0 {
		count := sp.attributesMatched()
		if count < c.minAttributes {
			putSpan(sp)
			return false
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(otherEntrySpanKey, sp)

	return true
}

// batchCollector receives rows of matching resource-level
// This turns groups of batch values and Spans into SpanSets
type batchCollector struct {
	requireAtLeastOneMatchOverall bool
	minAttributes                 int

	// shared static spans used in KeepGroup. done for memory savings, but won't
	// work if the batchCollector is accessed concurrently
	buffer []*span
}

var _ parquetquery.GroupPredicate = (*batchCollector)(nil)

func (c *batchCollector) String() string {
	return fmt.Sprintf("batchCollector{%v, %d}", c.requireAtLeastOneMatchOverall, c.minAttributes)
}

func (c *batchCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	// TODO - This wraps everything up in a spanset per batch.
	// We probably don't need to do this, since the traceCollector
	// flattens it into 1 spanset per trace.  All we really need
	// todo is merge the resource-level attributes onto the spans
	// and filter out spans that didn't match anything.
	c.buffer = c.buffer[:0]

	resAttrs := make(map[traceql.Attribute]traceql.Static)
	for _, kv := range res.OtherEntries {
		if span, ok := kv.Value.(*span); ok {
			c.buffer = append(c.buffer, span)
			continue
		}

		// Attributes show up here
		resAttrs[newResAttr(kv.Key)] = kv.Value.(traceql.Static)
	}

	// Throw out batches without any spans
	if len(c.buffer) == 0 {
		return false
	}

	// Gather Attributes from dedicated resource-level columns
	for _, e := range res.Entries {
		switch e.Value.Kind() {
		case parquet.Int64:
			resAttrs[newResAttr(e.Key)] = traceql.NewStaticInt(int(e.Value.Int64()))
		case parquet.ByteArray:
			resAttrs[newResAttr(e.Key)] = traceql.NewStaticString(e.Value.String())
		}
	}

	if c.minAttributes > 0 {
		if len(resAttrs) < c.minAttributes {
			return false
		}
	}

	// Copy resource-level attributes to the individual spans now
	for k, v := range resAttrs {
		for _, span := range c.buffer {
			if _, alreadyExists := span.attributes[k]; !alreadyExists {
				span.attributes[k] = v
			}
		}
	}

	// Remove unmatched attributes
	for _, span := range c.buffer {
		for k, v := range span.attributes {
			if v.Type == traceql.TypeNil {
				delete(span.attributes, k)
			}
		}
	}

	var filteredSpans []traceql.Span

	// Copy over only spans that met minimum criteria
	if c.requireAtLeastOneMatchOverall {
		for _, span := range c.buffer {
			if span.attributesMatched() > 0 {
				filteredSpans = append(filteredSpans, span)
				continue
			}
			putSpan(span)
		}
	} else {
		filteredSpans = make([]traceql.Span, 0, len(c.buffer))
		for _, span := range c.buffer {
			filteredSpans = append(filteredSpans, span)
		}
	}

	// Throw out batches without any spans
	if len(filteredSpans) == 0 {
		return false
	}

	sp := &traceql.Spanset{
		Spans: filteredSpans,
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(otherEntrySpansetKey, sp)

	return true
}

// traceCollector receives rows from the resource-level matches.
// It adds trace-level attributes into the spansets before
// they are returned
type traceCollector struct {
	// traceAttrs is a map reused by KeepGroup to reduce allocations
	traceAttrs map[traceql.Attribute]traceql.Static
}

var _ parquetquery.GroupPredicate = (*traceCollector)(nil)

func (c *traceCollector) String() string {
	return "traceCollector{}"
}

func (c *traceCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	finalSpanset := &traceql.Spanset{}
	// init the map and clear out if necessary
	if c.traceAttrs == nil {
		c.traceAttrs = make(map[traceql.Attribute]traceql.Static)
	}
	for k := range c.traceAttrs {
		delete(c.traceAttrs, k)
	}

	for _, e := range res.Entries {
		switch e.Key {
		case columnPathTraceID:
			finalSpanset.TraceID = e.Value.ByteArray()
		case columnPathStartTimeUnixNano:
			finalSpanset.StartTimeUnixNanos = e.Value.Uint64()
		case columnPathDurationNanos:
			finalSpanset.DurationNanos = e.Value.Uint64()
			c.traceAttrs[traceql.NewIntrinsic(traceql.IntrinsicTraceDuration)] = traceql.NewStaticDuration(time.Duration(finalSpanset.DurationNanos))
		case columnPathRootSpanName:
			finalSpanset.RootSpanName = e.Value.String()
			c.traceAttrs[traceql.NewIntrinsic(traceql.IntrinsicTraceRootSpan)] = traceql.NewStaticString(finalSpanset.RootSpanName)
		case columnPathRootServiceName:
			finalSpanset.RootServiceName = e.Value.String()
			c.traceAttrs[traceql.NewIntrinsic(traceql.IntrinsicTraceRootService)] = traceql.NewStaticString(finalSpanset.RootServiceName)
		}
	}

	for _, e := range res.OtherEntries {
		if spanset, ok := e.Value.(*traceql.Spanset); ok {
			finalSpanset.Spans = append(finalSpanset.Spans, spanset.Spans...)

			// loop over all spans and add the trace-level attributes
			for k, v := range c.traceAttrs {
				for _, sp := range spanset.Spans {
					s := sp.(*span)
					if _, alreadyExists := s.attributes[k]; !alreadyExists {
						s.attributes[k] = v
					}
				}
			}
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(otherEntrySpansetKey, finalSpanset)

	return true
}

// attributeCollector receives rows from the individual key/string/int/etc
// columns and joins them together into map[key]value entries with the
// right type.
type attributeCollector struct{}

var _ parquetquery.GroupPredicate = (*attributeCollector)(nil)

func (c *attributeCollector) String() string {
	return "attributeCollector{}"
}

func (c *attributeCollector) KeepGroup(res *parquetquery.IteratorResult) bool {
	var key string
	var val traceql.Static

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
			val = traceql.NewStaticString(e.Value.String())
		case "int":
			val = traceql.NewStaticInt(int(e.Value.Int64()))
		case "float":
			val = traceql.NewStaticFloat(e.Value.Double())
		case "bool":
			val = traceql.NewStaticBool(e.Value.Boolean())
		}
	}

	res.Entries = res.Entries[:0]
	res.OtherEntries = res.OtherEntries[:0]
	res.AppendOtherValue(key, val)

	return true
}

func newSpanAttr(name string) traceql.Attribute {
	return traceql.NewScopedAttribute(traceql.AttributeScopeSpan, false, name)
}

func newResAttr(name string) traceql.Attribute {
	return traceql.NewScopedAttribute(traceql.AttributeScopeResource, false, name)
}
