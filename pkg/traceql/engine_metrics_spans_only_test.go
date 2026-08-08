package traceql

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

type batchTestSpanIterator struct {
	spans []Span
	next  int
	batch []Span

	secondPass        SecondPassFn
	beforeNext        func() error
	beforeNextBatch   func() error
	afterReleaseBatch func()
	errOnExhaustion   error

	nextCalls       int
	batchCalls      int
	releaseCalls    int
	secondPassCalls int
}

var _ SpanBatchIterator = (*batchTestSpanIterator)(nil)

func (i *batchTestSpanIterator) Next(ctx context.Context) (Span, error) {
	i.nextCalls++
	if i.beforeNext != nil {
		if err := i.beforeNext(); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if i.secondPass != nil {
		_, err := i.secondPass(&Spanset{Spans: []Span{i.spans[0]}})
		if err != nil {
			return nil, err
		}
		i.secondPass = nil
		i.secondPassCalls++
	}

	if i.next == len(i.spans) {
		return nil, i.errOnExhaustion
	}

	span := i.spans[i.next]
	i.next++
	return span, nil
}

func (i *batchTestSpanIterator) NextBatch(ctx context.Context, size int) ([]Span, error) {
	i.batchCalls++
	if i.beforeNextBatch != nil {
		if err := i.beforeNextBatch(); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if i.secondPass != nil {
		_, err := i.secondPass(&Spanset{Spans: []Span{i.spans[0]}})
		if err != nil {
			return nil, err
		}
		i.secondPass = nil
		i.secondPassCalls++
	}

	if i.next == len(i.spans) {
		return nil, i.errOnExhaustion
	}

	end := min(i.next+size, len(i.spans))
	i.batch = append(i.batch[:0], i.spans[i.next:end]...)
	i.next = end
	return i.batch, nil
}

func (i *batchTestSpanIterator) ReleaseBatch() {
	i.releaseCalls++
	clear(i.batch)
	i.batch = i.batch[:0]
	if i.afterReleaseBatch != nil {
		i.afterReleaseBatch()
	}
}

func (*batchTestSpanIterator) Close() {}

type singleTestSpanIterator struct {
	spans []Span
	next  int
}

var _ SpanIterator = (*singleTestSpanIterator)(nil)

func (i *singleTestSpanIterator) Next(ctx context.Context) (Span, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if i.next == len(i.spans) {
		return nil, nil
	}

	span := i.spans[i.next]
	i.next++
	return span, nil
}

func (*singleTestSpanIterator) Close() {}

type batchTestSpanFetcher struct {
	iter *batchTestSpanIterator
}

var _ SpansetFetcher = (*batchTestSpanFetcher)(nil)

func (*batchTestSpanFetcher) Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error) {
	return FetchSpansResponse{}, util.ErrUnsupported
}

func (f *batchTestSpanFetcher) FetchSpans(_ context.Context, req FetchSpansRequest) (FetchSpansOnlyResponse, error) {
	f.iter.secondPass = req.SecondPass
	return FetchSpansOnlyResponse{
		Results: f.iter,
		Stats:   func() FetchSpansStats { return FetchSpansStats{} },
	}, nil
}

type singleTestSpanFetcher struct {
	iter SpanIterator
}

var _ SpansetFetcher = (*singleTestSpanFetcher)(nil)

func (*singleTestSpanFetcher) Fetch(context.Context, FetchSpansRequest) (FetchSpansResponse, error) {
	return FetchSpansResponse{}, util.ErrUnsupported
}

func (f *singleTestSpanFetcher) FetchSpans(context.Context, FetchSpansRequest) (FetchSpansOnlyResponse, error) {
	return FetchSpansOnlyResponse{
		Results: f.iter,
		Stats:   func() FetchSpansStats { return FetchSpansStats{} },
	}, nil
}

func newBatchTestSpan(service, traceID string, start uint64, summary bool) *mockSpan {
	span := newMockSpan(nil).
		WithStartTime(start).
		WithSpanString("service", service)
	span.attributes[IntrinsicTraceIDAttribute] = NewStaticString(traceID)
	if summary {
		span.attributes[NewAttribute(SpanPruningAttribute)] = NewStaticBool(true)
	}
	return span
}

func newBatchTestEvaluator(t *testing.T, exemplars uint32) MetricsEvaluator {
	t.Helper()

	eval, err := NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
		Query:     "{} | count_over_time() by (span.service)",
		Start:     1,
		End:       1_000,
		Step:      100,
		Exemplars: exemplars,
	}, WithWatchers(NewSpanPruningWatcher()))
	require.NoError(t, err)
	return eval
}

func TestMetricsEvaluatorSampleExemplarDisablesMetadataFetchAtLimit(t *testing.T) {
	eval := &metricsEvaluator{
		maxExemplars: 2,
		exemplarMap:  make(map[string]struct{}),
	}
	eval.exemplarsAvailable.Store(true)

	eval.mtx.Lock()
	firstSampled := eval.sampleExemplar([]byte("trace-a"))
	availableAfterFirst := eval.exemplarsAvailable.Load()
	secondSampled := eval.sampleExemplar([]byte("trace-b"))
	eval.mtx.Unlock()

	require.True(t, firstSampled)
	require.True(t, availableAfterFirst)
	require.True(t, secondSampled)
	require.False(t, eval.exemplarsAvailable.Load())
}

func TestMetricsEvaluatorDoSpansOnlyBatchesEngineWork(t *testing.T) {
	iter := &batchTestSpanIterator{spans: []Span{
		newBatchTestSpan("api", "trace-a", 100, false),
		newBatchTestSpan("worker", "trace-b", 200, true),
		newBatchTestSpan("api", "trace-c", 300, false),
	}}
	eval := newBatchTestEvaluator(t, 2)
	me, err := singleBatchMetricsEvaluator(eval)
	require.NoError(t, err)
	iter.beforeNextBatch = func() error {
		if !me.mtx.TryLock() {
			return errors.New("NextBatch ran while the evaluator lock was held")
		}
		me.mtx.Unlock()
		return nil
	}

	require.NoError(t, eval.Do(t.Context(), &batchTestSpanFetcher{iter: iter}, 0, 0, 0))
	require.Zero(t, iter.nextCalls, "batch-capable storage must not fall back to per-span Next")
	require.Equal(t, 2, iter.batchCalls, "the final empty batch confirms iterator exhaustion")
	require.Equal(t, 1, iter.releaseCalls)
	require.Equal(t, 1, iter.secondPassCalls, "SecondPass must run while advancing storage, before the evaluator lock")

	require.Equal(t, uint64(len(iter.spans)), me.spansTotal)

	results := eval.Results()
	require.Len(t, results, 2)

	var totalSamples float64
	var totalExemplars int
	for _, series := range results {
		for _, value := range series.Values {
			if !math.IsNaN(value) {
				totalSamples += value
			}
		}
		totalExemplars += len(series.Exemplars)
	}
	require.Equal(t, float64(len(iter.spans)), totalSamples)
	require.Equal(t, 2, totalExemplars)
	require.Equal(t, int64(1), eval.Metrics().AdditionalMetrics[SpanPruningAttribute])
}

func TestMetricsEvaluatorSecondPassDoesNotWaitForMetricsAggregation(t *testing.T) {
	eval := newBatchTestEvaluator(t, 0)
	me, err := singleBatchMetricsEvaluator(eval)
	require.NoError(t, err)
	require.NotNil(t, me.storageReq.SecondPass)

	me.mtx.Lock()
	done := make(chan error, 1)
	go func() {
		_, err := me.storageReq.SecondPass(&Spanset{Spans: []Span{newBatchTestSpan("api", "trace-a", 100, false)}})
		done <- err
	}()

	select {
	case err := <-done:
		me.mtx.Unlock()
		require.NoError(t, err)
	case <-time.After(time.Second):
		me.mtx.Unlock()
		require.NoError(t, <-done, "SecondPass must not wait for metrics aggregation")
		t.Fatal("SecondPass waited for metrics aggregation")
	}
}

func TestMetricsEvaluatorDoSpansOnlyBatchMatchesSingleSpanResults(t *testing.T) {
	spans := func() []Span {
		return []Span{
			newBatchTestSpan("api", "trace-a", 100, false),
			newBatchTestSpan("worker", "trace-b", 200, true),
			newBatchTestSpan("api", "trace-c", 300, false),
		}
	}

	batched := newBatchTestEvaluator(t, 2)
	require.NoError(t, batched.Do(t.Context(), &batchTestSpanFetcher{
		iter: &batchTestSpanIterator{spans: spans()},
	}, 0, 0, 0))

	single := newBatchTestEvaluator(t, 2)
	require.NoError(t, single.Do(t.Context(), &singleTestSpanFetcher{
		iter: &singleTestSpanIterator{spans: spans()},
	}, 0, 0, 0))

	toSlice := func(series SeriesSet) []TimeSeries {
		out := make([]TimeSeries, 0, len(series))
		for _, value := range series {
			out = append(out, value)
		}
		return out
	}
	requireEqualSeriesSets(t, toSlice(single.Results()), batched.Results())
	for key, singleSeries := range single.Results() {
		batchedSeries, ok := batched.Results()[key]
		require.True(t, ok)
		require.Len(t, batchedSeries.Exemplars, len(singleSeries.Exemplars))
		for i, singleExemplar := range singleSeries.Exemplars {
			batchedExemplar := batchedSeries.Exemplars[i]
			labelMap := func(labels Labels) map[string]Static {
				out := make(map[string]Static, len(labels))
				for _, label := range labels {
					out[label.Name] = label.Value
				}
				return out
			}
			require.Equal(t, labelMap(singleExemplar.Labels), labelMap(batchedExemplar.Labels))
			require.Equal(t, singleExemplar.TimestampMs, batchedExemplar.TimestampMs)
			if math.IsNaN(singleExemplar.Value) {
				require.True(t, math.IsNaN(batchedExemplar.Value))
			} else {
				require.Equal(t, singleExemplar.Value, batchedExemplar.Value)
			}
		}
	}
	require.Equal(t, single.Metrics(), batched.Metrics())
}

func TestMetricsEvaluatorDoSpansOnlyBatchRetainsArrayLabels(t *testing.T) {
	values := []int{1, 2}
	attr := NewScopedAttribute(AttributeScopeSpan, false, "array")
	span := newBatchTestSpan("api", "trace-a", 100, false)
	span.attributes[attr] = NewStaticIntArray(values)

	spans := make([]Span, spansOnlyBatchSize+1)
	for i := range spans {
		spans[i] = span
	}

	mutated := false
	iter := &batchTestSpanIterator{
		spans: spans,
		afterReleaseBatch: func() {
			if !mutated {
				values[0], values[1] = 3, 4
				mutated = true
			}
		},
	}
	eval, err := NewEngine().CompileMetricsQueryRange(&tempopb.QueryRangeRequest{
		Query:     "{} | count_over_time() by (span.array)",
		Start:     1,
		End:       1_000,
		Step:      100,
		Exemplars: 1,
	})
	require.NoError(t, err)

	require.NoError(t, eval.Do(t.Context(), &batchTestSpanFetcher{iter: iter}, 0, 0, 0))
	require.True(t, mutated)

	counts := map[[2]int]float64{}
	for _, series := range eval.Results() {
		var array []int
		for _, label := range series.Labels {
			if values, ok := label.Value.IntArray(); ok {
				array = values
			}
		}
		require.Len(t, array, 2)

		total := 0.0
		for _, value := range series.Values {
			if !math.IsNaN(value) {
				total += value
			}
		}
		key := [2]int{array[0], array[1]}
		counts[key] = total

		if key == [2]int{1, 2} {
			require.Len(t, series.Exemplars, 1)
			var exemplarArray []int
			for _, label := range series.Exemplars[0].Labels {
				if values, ok := label.Value.IntArray(); ok {
					exemplarArray = values
				}
			}
			require.Equal(t, []int{1, 2}, exemplarArray)
		}
	}
	require.Equal(t, map[[2]int]float64{{1, 2}: spansOnlyBatchSize, {3, 4}: 1}, counts)
}

func TestMetricsEvaluatorDoSpansOnlyBatchHonorsMaxSeries(t *testing.T) {
	iter := &batchTestSpanIterator{spans: []Span{
		newBatchTestSpan("api", "trace-a", 100, false),
		newBatchTestSpan("worker", "trace-b", 200, false),
		newBatchTestSpan("database", "trace-c", 300, false),
	}}
	eval := newBatchTestEvaluator(t, 0)
	me, err := singleBatchMetricsEvaluator(eval)
	require.NoError(t, err)
	iter.beforeNext = func() error {
		if !me.mtx.TryLock() {
			return errors.New("Next ran while the evaluator lock was held")
		}
		me.mtx.Unlock()
		return nil
	}

	require.NoError(t, eval.Do(t.Context(), &batchTestSpanFetcher{iter: iter}, 0, 0, 2))
	require.Len(t, eval.Results(), 2)
	require.Equal(t, 2, iter.nextCalls, "the max-series cutoff must not prefetch more spans")
	require.Zero(t, iter.batchCalls, "bounded requests must preserve per-span iterator advancement")
	require.Zero(t, iter.releaseCalls)

	require.Equal(t, uint64(2), me.spansTotal, "spans after the max-series boundary must not be observed")
}

type cancelOnStartSpan struct {
	Span
	cancel    context.CancelFunc
	cancelled bool
}

func (s *cancelOnStartSpan) StartTimeUnixNanos() uint64 {
	if !s.cancelled {
		s.cancelled = true
		s.cancel()
	}
	return s.Span.StartTimeUnixNanos()
}

func TestMetricsEvaluatorDoSpansOnlyBatchStopsAtMaxSeriesBeforeStorageError(t *testing.T) {
	iter := &batchTestSpanIterator{
		spans: []Span{
			newBatchTestSpan("api", "trace-a", 100, false),
			newBatchTestSpan("worker", "trace-b", 200, false),
		},
		errOnExhaustion: errors.New("storage iterator failed after the max-series boundary"),
	}
	eval := newBatchTestEvaluator(t, 0)

	require.NoError(t, eval.Do(t.Context(), &batchTestSpanFetcher{iter: iter}, 0, 0, 1))
	require.Len(t, eval.Results(), 1)
	require.Equal(t, 1, iter.nextCalls)
	require.Zero(t, iter.batchCalls)
	require.Zero(t, iter.releaseCalls)

	me, err := singleBatchMetricsEvaluator(eval)
	require.NoError(t, err)
	require.Equal(t, uint64(1), me.spansTotal)
}

func TestMetricsEvaluatorDoSpansOnlyBatchPropagatesStorageErrors(t *testing.T) {
	storageErr := errors.New("storage iterator failed")
	iter := &batchTestSpanIterator{
		spans:           []Span{newBatchTestSpan("api", "trace-a", 100, false)},
		errOnExhaustion: storageErr,
	}
	eval := newBatchTestEvaluator(t, 0)

	err := eval.Do(t.Context(), &batchTestSpanFetcher{iter: iter}, 0, 0, 0)
	require.ErrorIs(t, err, storageErr)
	require.Equal(t, 1, iter.releaseCalls)

	me, extractErr := singleBatchMetricsEvaluator(eval)
	require.NoError(t, extractErr)
	require.Equal(t, uint64(1), me.spansTotal)
}

func TestMetricsEvaluatorDoSpansOnlyBatchObservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	first := &cancelOnStartSpan{
		Span:   newBatchTestSpan("api", "trace-a", 100, false),
		cancel: cancel,
	}
	iter := &batchTestSpanIterator{spans: []Span{
		first,
		newBatchTestSpan("worker", "trace-b", 200, false),
	}}
	eval := newBatchTestEvaluator(t, 0)

	err := eval.Do(ctx, &batchTestSpanFetcher{iter: iter}, 0, 0, 0)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, iter.releaseCalls)

	me, extractErr := singleBatchMetricsEvaluator(eval)
	require.NoError(t, extractErr)
	require.Equal(t, uint64(1), me.spansTotal, "cancellation must stop processing within a fetched batch")
}
