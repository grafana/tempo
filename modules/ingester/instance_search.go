package ingester

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	ot_log "github.com/opentracing/opentracing-go/log"
	"go.uber.org/atomic"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/search"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
)

func (i *instance) Search(ctx context.Context, req *tempopb.SearchRequest) (*tempopb.SearchResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.Search")
	defer span.Finish()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxResults := int(req.Limit)
	// if limit is not set, use a safe default
	if maxResults == 0 {
		maxResults = 20
	}

	span.LogFields(ot_log.String("SearchRequest", req.String()))

	sr := search.NewResults()
	defer sr.Close() // signal all running workers to quit

	// Lock headblock separately from other blocks and release it as soon as this
	// subtask is finished.
	// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
	// To avoid deadlocks this function and all others must acquire them in
	// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
	// then headblockMtx. Even if the likelihood is low it is a statistical certainly
	// that eventually a deadlock will occur.
	i.headBlockMtx.RLock()
	i.searchBlock(ctx, req, sr, i.headBlock.BlockMeta(), i.headBlock, i.headBlockMtx.RUnlock)

	// Lock blocks mutex until all search tasks are finished and this function exists. This avoids
	// deadlocking with other activity (ingest, flushing), caused by releasing
	// and then attempting to retake the lock.
	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, b := range i.completingBlocks {
		i.searchBlock(ctx, req, sr, b.BlockMeta(), b, nil)
	}

	for _, b := range i.completeBlocks {
		i.searchBlock(ctx, req, sr, b.BlockMeta(), b, nil)
	}

	sr.AllWorkersStarted()

	// read and combine search results
	combiner := traceql.NewMetadataCombiner()

	// collect results from all the goroutines via sr.Results channel.
	// range loop will exit when sr.Results channel is closed.
	for result := range sr.Results() {
		if combiner.Count() >= maxResults {
			sr.Close() // signal pending workers to exit
			continue
		}

		combiner.AddMetadata(result)
	}

	if sr.Error() != nil {
		return nil, sr.Error()
	}

	return &tempopb.SearchResponse{
		Traces: combiner.Metadata(),
		Metrics: &tempopb.SearchMetrics{
			InspectedTraces: sr.TracesInspected(),
			InspectedBytes:  sr.BytesInspected(),
		},
	}, nil
}

// searchBlock starts a search task for the given block. The block must already be under lock,
// and this method calls cleanup to unlock the block when done.
func (i *instance) searchBlock(ctx context.Context, req *tempopb.SearchRequest, sr *search.Results, meta *backend.BlockMeta, block common.Searcher, cleanup func()) {
	// confirm block should be included in search
	if !includeBlock(meta, req) {
		if cleanup != nil {
			cleanup()
		}
		return
	}

	blockID := meta.BlockID

	sr.StartWorker()
	go func(e common.Searcher, cleanup func()) {
		if cleanup != nil {
			defer cleanup()
		}
		defer sr.FinishWorker()

		span, ctx := opentracing.StartSpanFromContext(ctx, "instance.searchBlock")
		defer span.Finish()

		span.LogFields(ot_log.Event("block entry mtx acquired"))
		span.SetTag("blockID", blockID)

		var resp *tempopb.SearchResponse
		var err error

		opts := common.DefaultSearchOptions()
		if api.IsTraceQLQuery(req) {
			// note: we are creating new engine for each wal block,
			// and engine.ExecuteSearch is parsing the query for each block
			resp, err = traceql.NewEngine().ExecuteSearch(ctx, req, traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return e.Fetch(ctx, req, opts)
			}))
		} else {
			resp, err = e.Search(ctx, req, opts)
		}

		if errors.Is(err, common.ErrUnsupported) {
			level.Warn(log.Logger).Log("msg", "block does not support search", "blockID", blockID)
			return
		}
		if err != nil {
			level.Error(log.Logger).Log("msg", "error searching block", "blockID", blockID, "err", err)
			sr.SetError(err)
			return
		}

		for _, t := range resp.Traces {
			sr.AddResult(ctx, t)
		}
		sr.AddBlockInspected()

		sr.AddBytesInspected(resp.Metrics.InspectedBytes)
		sr.AddTraceInspected(resp.Metrics.InspectedTraces)
	}(block, cleanup)
}

func (i *instance) SearchTags(ctx context.Context, scope string) (*tempopb.SearchTagsResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	// check if it's the special intrinsic scope
	if scope == api.ParamScopeIntrinsic {
		return &tempopb.SearchTagsResponse{
			TagNames: search.GetVirtualIntrinsicValues(),
		}, nil
	}

	// parse for normal scopes
	attributeScope := traceql.AttributeScopeFromString(scope)
	if attributeScope == traceql.AttributeScopeUnknown {
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}
		err = s.SearchTags(ctx, attributeScope, dv.Collect, common.DefaultSearchOptions())
		if err != nil && !errors.Is(err, common.ErrUnsupported) {
			return fmt.Errorf("unexpected error searching tags: %w", err)
		}

		return nil
	}

	i.headBlockMtx.RLock()
	err = search(i.headBlock, distinctValues)
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, b := range i.completingBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}
	for _, b := range i.completeBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tags in instance exceeded limit, reduce cardinality or size of tags", "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return &tempopb.SearchTagsResponse{
		TagNames: distinctValues.Strings(),
	}, nil
}

// SearchTagsV2 calls SearchTags for each scope and returns the results.
func (i *instance) SearchTagsV2(ctx context.Context, scope string) (*tempopb.SearchTagsV2Response, error) {
	scopes := []string{scope}
	if scope == "" {
		// start with intrinsic scope and all traceql attribute scopes
		atts := traceql.AllAttributeScopes()
		scopes = make([]string, 0, len(atts)+1) // +1 for intrinsic

		scopes = append(scopes, api.ParamScopeIntrinsic)
		for _, att := range atts {
			scopes = append(scopes, att.String())
		}
	}
	resps := make([]*tempopb.SearchTagsResponse, len(scopes))

	overallError := atomic.NewError(nil)
	wg := sync.WaitGroup{}
	for idx := range scopes {
		resps[idx] = &tempopb.SearchTagsResponse{}

		wg.Add(1)
		go func(scope string, ret **tempopb.SearchTagsResponse) {
			defer wg.Done()

			resp, err := i.SearchTags(ctx, scope)
			if err != nil {
				overallError.Store(fmt.Errorf("error searching tags: %s: %w", scope, err))
				return
			}

			*ret = resp
		}(scopes[idx], &resps[idx])
	}
	wg.Wait()

	err := overallError.Load()
	if err != nil {
		return nil, err
	}

	// build response
	resp := &tempopb.SearchTagsV2Response{}
	for idx := range resps {
		resp.Scopes = append(resp.Scopes, &tempopb.SearchTagsV2Scope{
			Name: scopes[idx],
			Tags: resps[idx].TagNames,
		})
	}

	return resp, nil
}

func (i *instance) SearchTagValues(ctx context.Context, tagName string) (*tempopb.SearchTagValuesResponse, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	distinctValues := util.NewDistinctStringCollector(limit)

	var inspectedBlocks, maxBlocks int
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = limit
	}

	search := func(s common.Searcher, dv *util.DistinctStringCollector) error {
		if maxBlocks > 0 && inspectedBlocks >= maxBlocks {
			return nil
		}

		if s == nil {
			return nil
		}
		if dv.Exceeded() {
			return nil
		}

		inspectedBlocks++
		err = s.SearchTagValues(ctx, tagName, dv.Collect, common.DefaultSearchOptions())
		if err != nil && !errors.Is(err, common.ErrUnsupported) {
			return fmt.Errorf("unexpected error searching tag values (%s): %w", tagName, err)
		}

		return nil
	}

	i.headBlockMtx.RLock()
	err = search(i.headBlock, distinctValues)
	i.headBlockMtx.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err)
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	for _, b := range i.completingBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}
	for _, b := range i.completeBlocks {
		if err = search(b, distinctValues); err != nil {
			return nil, fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err)
		}
	}

	if distinctValues.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", tagName, "userID", userID, "limit", limit, "total", distinctValues.TotalDataSize())
	}

	return &tempopb.SearchTagValuesResponse{
		TagValues: distinctValues.Strings(),
	}, nil
}

func (i *instance) SearchTagValuesV2(ctx context.Context, req *tempopb.SearchTagValuesRequest) (*tempopb.SearchTagValuesV2Response, error) {
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, err
	}

	limit := i.limiter.limits.MaxBytesPerTagValuesQuery(userID)
	valueCollector := util.NewDistinctValueCollector[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) })

	engine := traceql.NewEngine()

	wg := boundedwaitgroup.New(20) // TODO: Make configurable
	var anyErr atomic.Error
	var inspectedBlocks atomic.Int32
	var maxBlocks int32
	if limit := i.limiter.limits.MaxBlocksPerTagValuesQuery(userID); limit > 0 {
		maxBlocks = int32(limit)
	}

	tag, err := traceql.ParseIdentifier(req.TagName)
	if err != nil {
		return nil, err
	}

	query := extractMatchers(req.Query)

	var searchBlock func(common.Searcher) error
	if !i.autocompleteFilteringEnabled && isEmptyQuery(query) {
		// If filtering is disabled or query is empty,
		// we can use the more efficient SearchTagValuesV2 method.
		searchBlock = func(s common.Searcher) error {
			if anyErr.Load() != nil {
				return nil // Early exit if any error has occurred
			}

			if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
				return nil
			}

			return s.SearchTagValuesV2(ctx, tag, traceql.MakeCollectTagValueFunc(valueCollector.Collect), common.DefaultSearchOptions())
		}
	} else {
		searchBlock = func(s common.Searcher) error {
			if anyErr.Load() != nil {
				return nil // Early exit if any error has occurred
			}

			if maxBlocks > 0 && inspectedBlocks.Inc() > maxBlocks {
				return nil
			}

			fetcher := traceql.NewAutocompleteFetcherWrapper(func(ctx context.Context, req traceql.AutocompleteRequest, cb traceql.AutocompleteCallback) error {
				return s.FetchTagValues(ctx, req, cb, common.DefaultSearchOptions())
			})

			return engine.ExecuteTagValues(ctx, tag, query, traceql.MakeCollectTagValueFunc(valueCollector.Collect), fetcher)
		}
	}

	// head block
	// A warning about deadlocks!!  This area does a hard-acquire of both mutexes.
	// To avoid deadlocks this function and all others must acquire them in
	// the ** same_order ** or else!!! i.e. another function can't acquire blocksMtx
	// then headblockMtx. Even if the likelihood is low it is a statistical certainly
	// that eventually a deadlock will occur.
	i.headBlockMtx.RLock()
	if i.headBlock != nil {
		wg.Add(1)
		go func() {
			defer i.headBlockMtx.RUnlock()
			defer wg.Done()
			if err := searchBlock(i.headBlock); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching head block (%s): %w", i.headBlock.BlockMeta().BlockID, err))
			}
		}()
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	// completed blocks
	for _, b := range i.completeBlocks {
		wg.Add(1)
		go func(b *localBlock) {
			defer wg.Done()
			if err := searchBlock(b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching complete block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	// completing blocks
	for _, b := range i.completingBlocks {
		wg.Add(1)
		go func(b common.WALBlock) {
			defer wg.Done()
			if err := searchBlock(b); err != nil {
				anyErr.Store(fmt.Errorf("unexpected error searching completing block (%s): %w", b.BlockMeta().BlockID, err))
			}
		}(b)
	}

	wg.Wait()

	if err := anyErr.Load(); err != nil {
		return nil, err
	}

	if valueCollector.Exceeded() {
		level.Warn(log.Logger).Log("msg", "size of tag values in instance exceeded limit, reduce cardinality or size of tags", "tag", req.TagName, "userID", userID, "limit", limit, "total", valueCollector.TotalDataSize())
	}

	resp := &tempopb.SearchTagValuesV2Response{}

	for _, v := range valueCollector.Values() {
		v2 := v
		resp.TagValues = append(resp.TagValues, &v2)
	}

	return resp, nil
}

func isEmptyQuery(query string) bool {
	return query == emptyQuery || len(query) == 0
}

// TODO: Support spaces

// Regex to extract matchers from a query string
// This regular expression matches a string that contains three groups separated by operators.
// The first group is a string of alphabetical characters, dots, and underscores.
// The second group is a comparison operator, which can be one of several possibilities, including =, >, <, and !=.
// The third group is one of several possible values: a string enclosed in double quotes,
// a number with an optional time unit (such as "ns", "ms", "s", "m", or "h"),
// a plain number, or the boolean values "true" or "false".
// Example: "http.status_code = 200" from the query "{ .http.status_code = 200 && .http.method = }"
var matchersRegexp = regexp.MustCompile(`[a-zA-Z._]+\s*[=|<=|>=|=~|!=|>|<|!~]\s*(?:"[a-zA-Z./_0-9-]+"|[0-9smh]+|true|false)`)

// TODO: Merge into a single regular expression

// Regex to extract selectors from a query string
// This regular expression matches a string that contains a single spanset filter and no OR `||` conditions.
// Examples
//
//	Query                        |  Match
//
// { .bar = "foo" }                          |   Yes
// { .bar = "foo" && .foo = "bar" }          |   Yes
// { .bar = "foo" || .foo = "bar" }          |   No
// { .bar = "foo" } && { .foo = "bar" }      |   No
// { .bar = "foo" } || { .foo = "bar" }      |   No
var singleFilterRegexp = regexp.MustCompile(`^{[a-zA-Z._\s\-()/&=<>~!0-9"]*}$`)

const emptyQuery = "{}"

// TODO: Move to traceql package

// extractMatchers extracts matchers from a query string and returns a string that can be parsed by the storage layer.
func extractMatchers(query string) string {
	query = strings.TrimSpace(query)

	if len(query) == 0 {
		return emptyQuery
	}

	selector := singleFilterRegexp.FindString(query)
	if len(selector) == 0 {
		return emptyQuery
	}

	matchers := matchersRegexp.FindAllString(query, -1)

	var q strings.Builder
	q.WriteString("{")
	for i, m := range matchers {
		if i > 0 {
			q.WriteString(" && ")
		}
		q.WriteString(m)
	}
	q.WriteString("}")

	return q.String()
}

// includeBlock uses the provided time range to determine if the block should be included in the search.
func includeBlock(b *backend.BlockMeta, req *tempopb.SearchRequest) bool {
	start := int64(req.Start)
	end := int64(req.End)

	if start == 0 || end == 0 {
		return true
	}

	return b.StartTime.Unix() <= end && b.EndTime.Unix() >= start
}
