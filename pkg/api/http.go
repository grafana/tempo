package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/common/model"

	"github.com/grafana/dskit/httpgrpc"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
)

const (
	URLParamTraceID = "traceID"
	// search
	urlParamQuery           = "q"
	urlParamTags            = "tags"
	urlParamMinDuration     = "minDuration"
	urlParamMaxDuration     = "maxDuration"
	urlParamLimit           = "limit"
	urlParamStart           = "start"
	urlParamEnd             = "end"
	urlParamSpansPerSpanSet = "spss"
	urlParamStep            = "step"
	urlParamShard           = "shard"
	urlParamShardCount      = "shardCount"
	urlParamSince           = "since"

	// backend search (querier/serverless)
	urlParamStartPage        = "startPage"
	urlParamPagesToSearch    = "pagesToSearch"
	urlParamBlockID          = "blockID"
	urlParamEncoding         = "encoding"
	urlParamIndexPageSize    = "indexPageSize"
	urlParamTotalRecords     = "totalRecords"
	urlParamDataEncoding     = "dataEncoding"
	urlParamVersion          = "version"
	urlParamSize             = "size"
	urlParamFooterSize       = "footerSize"
	urlParamDedicatedColumns = "dc"

	// maxBytes (serverless only)
	urlParamMaxBytes = "maxBytes"

	// search tags
	urlParamScope = "scope"

	// generator summary
	urlParamGroupBy = "groupBy"
	// urlParamMetric  = "metric"

	HeaderAccept         = "Accept"
	HeaderContentType    = "Content-Type"
	HeaderAcceptProtobuf = "application/protobuf"
	HeaderAcceptJSON     = "application/json"

	PathPrefixQuerier   = "/querier"
	PathPrefixGenerator = "/generator"

	PathTraces              = "/api/traces/{traceID}"
	PathSearch              = "/api/search"
	PathSearchTags          = "/api/search/tags"
	PathSearchTagValues     = "/api/search/tag/{" + MuxVarTagName + "}/values"
	PathEcho                = "/api/echo"
	PathBuildInfo           = "/api/status/buildinfo"
	PathUsageStats          = "/status/usage-stats"
	PathSpanMetrics         = "/api/metrics"
	PathSpanMetricsSummary  = "/api/metrics/summary"
	PathMetricsQueryInstant = "/api/metrics/query"
	PathMetricsQueryRange   = "/api/metrics/query_range"

	// PathOverrides user configurable overrides
	PathOverrides = "/api/overrides"

	PathSearchTagValuesV2 = "/api/v2/search/tag/{" + MuxVarTagName + "}/values"
	PathSearchTagsV2      = "/api/v2/search/tags"

	QueryModeKey       = "mode"
	QueryModeIngesters = "ingesters"
	QueryModeBlocks    = "blocks"
	QueryModeAll       = "all"
	BlockStartKey      = "blockStart"
	BlockEndKey        = "blockEnd"

	defaultLimit           = 20
	defaultSpansPerSpanSet = 3
	defaultSince           = 1 * time.Hour
)

func ParseTraceID(r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)
	traceID, ok := vars[URLParamTraceID]
	if !ok {
		return nil, fmt.Errorf("please provide a traceID")
	}

	byteID, err := util.HexStringToTraceID(traceID)
	if err != nil {
		return nil, err
	}

	return byteID, nil
}

// ParseSearchRequest takes an http.Request and decodes query params to create a tempopb.SearchRequest
func ParseSearchRequest(r *http.Request) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:            map[string]string{},
		SpansPerSpanSet: defaultSpansPerSpanSet,
	}

	vals := r.URL.Query()

	if s, ok := extractQueryParam(vals, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(vals, urlParamEnd); ok {
		end, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		req.End = uint32(end)
	}

	query, queryFound := extractQueryParam(vals, urlParamQuery)
	if queryFound {
		// TODO hacky fix: we don't validate {} since this isn't handled correctly yet
		if query != "{}" {
			_, err := traceql.Parse(query)
			if err != nil {
				return nil, fmt.Errorf("invalid TraceQL query: %w", err)
			}
		}
		req.Query = query
	}

	encodedTags, tagsFound := extractQueryParam(vals, urlParamTags)
	if tagsFound {
		// tags and traceQL API are mutually exclusive
		if queryFound {
			return nil, fmt.Errorf("invalid request: can't specify tags and q in the same query")
		}

		decoder := logfmt.NewDecoder(strings.NewReader(encodedTags))

		for decoder.ScanRecord() {
			for decoder.ScanKeyval() {
				key := string(decoder.Key())
				if _, ok := req.Tags[key]; ok {
					return nil, fmt.Errorf("invalid tags: tag %s has been set twice", key)
				}
				req.Tags[key] = string(decoder.Value())
			}
		}

		if err := decoder.Err(); err != nil {
			var syntaxErr *logfmt.SyntaxError
			if ok := errors.As(err, &syntaxErr); ok {
				return nil, fmt.Errorf("invalid tags: %s at pos %d", syntaxErr.Msg, syntaxErr.Pos)
			}
			return nil, fmt.Errorf("invalid tags: %w", err)
		}
	}

	// if we don't have a query or tags, and we don't see start or end treat this like an old style search
	// if we have no tags but we DO have start/end we have to treat this like a range search with no
	// tags specified.
	if !queryFound && !tagsFound && req.Start == 0 && req.End == 0 {
		// Passing tags as individual query parameters is not supported anymore, clients should use the tags
		// query parameter instead. We still parse these tags since the initial Grafana implementation uses this.
		// As Grafana gets updated and/or versions using this get old we can remove this section.
		for k, v := range vals {
			// Skip reserved keywords
			if k == urlParamQuery || k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == urlParamLimit || k == urlParamSpansPerSpanSet || k == urlParamStart || k == urlParamEnd {
				continue
			}

			if len(v) > 0 && v[0] != "" {
				req.Tags[k] = v[0]
			}
		}
	}

	if s, ok := extractQueryParam(vals, urlParamMinDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid minDuration: %w", err)
		}
		req.MinDurationMs = uint32(dur.Milliseconds())
	}

	if s, ok := extractQueryParam(vals, urlParamMaxDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid maxDuration: %w", err)
		}
		req.MaxDurationMs = uint32(dur.Milliseconds())

		if req.MinDurationMs != 0 && req.MinDurationMs > req.MaxDurationMs {
			return nil, errors.New("invalid maxDuration: must be greater than minDuration")
		}
	}

	if s, ok := extractQueryParam(vals, urlParamLimit); ok {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 {
			return nil, errors.New("invalid limit: must be a positive number")
		}
		req.Limit = uint32(limit)
	}

	if s, ok := extractQueryParam(vals, urlParamSpansPerSpanSet); ok {
		spansPerSpanSet, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid spss: %w", err)
		}
		if spansPerSpanSet <= 0 {
			return nil, errors.New("invalid spss: must be a positive number")
		}
		req.SpansPerSpanSet = uint32(spansPerSpanSet)
	}

	// start and end == 0 is fine
	if req.End == 0 && req.Start == 0 {
		return req, nil
	}

	// if start or end are non-zero do some checks
	if req.End <= req.Start {
		return nil, fmt.Errorf("http parameter start must be before end. received start=%d end=%d", req.Start, req.End)
	}
	return req, nil
}

func ParseSpanMetricsRequest(r *http.Request) (*tempopb.SpanMetricsRequest, error) {
	req := &tempopb.SpanMetricsRequest{}
	vals := r.URL.Query()

	groupBy := vals.Get(urlParamGroupBy)
	req.GroupBy = groupBy

	query := vals.Get(urlParamQuery)
	req.Query = query

	l := vals.Get(urlParamLimit)
	if l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		req.Limit = uint64(limit)
	}

	if s, ok := extractQueryParam(vals, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(vals, urlParamEnd); ok {
		end, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		req.End = uint32(end)
	}

	return req, nil
}

func ParseSpanMetricsSummaryRequest(r *http.Request) (*tempopb.SpanMetricsSummaryRequest, error) {
	req := &tempopb.SpanMetricsSummaryRequest{}
	vals := r.URL.Query()

	groupBy := vals.Get(urlParamGroupBy)
	req.GroupBy = groupBy

	query := vals.Get(urlParamQuery)
	req.Query = query

	l := vals.Get(urlParamLimit)
	if l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		req.Limit = uint64(limit)
	}

	if s, ok := extractQueryParam(vals, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(vals, urlParamEnd); ok {
		end, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		req.End = uint32(end)
	}

	return req, nil
}

func ParseQueryInstantRequest(r *http.Request) (*tempopb.QueryInstantRequest, error) {
	req := &tempopb.QueryInstantRequest{}
	vals := r.URL.Query()

	// check "query" first. this was originally added for prom compatibility and Grafana still uses it.
	if s, ok := extractQueryParam(vals, "query"); ok {
		req.Query = s
	}

	// also check the `q` parameter. this is what all other Tempo endpoints take for a TraceQL query.
	if s, ok := extractQueryParam(vals, urlParamQuery); ok {
		req.Query = s
	}

	start, end, _ := bounds(vals)
	req.Start = uint64(start.UnixNano())
	req.End = uint64(end.UnixNano())

	return req, nil
}

func ParseQueryRangeRequest(r *http.Request) (*tempopb.QueryRangeRequest, error) {
	req := &tempopb.QueryRangeRequest{}
	vals := r.URL.Query()

	// check "query" first. this was originally added for prom compatibility and Grafana still uses it.
	if s, ok := extractQueryParam(vals, "query"); ok {
		req.Query = s
	}

	// also check the `q` parameter. this is what all other Tempo endpoints take for a TraceQL query.
	if s, ok := extractQueryParam(vals, urlParamQuery); ok {
		req.Query = s
	}

	if s, ok := extractQueryParam(vals, QueryModeKey); ok {
		req.QueryMode = s
	}

	start, end, _ := bounds(vals)
	req.Start = uint64(start.UnixNano())
	req.End = uint64(end.UnixNano())

	step, err := step(vals, start, end)
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusBadRequest, err.Error())
	}
	req.Step = uint64(step.Nanoseconds())

	shardCount, _ := extractQueryParam(vals, urlParamShardCount)
	if shardCount, err := strconv.Atoi(shardCount); err == nil {
		req.ShardCount = uint32(shardCount)
	}
	shard, _ := extractQueryParam(vals, urlParamShard)
	if shard, err := strconv.Atoi(shard); err == nil {
		req.ShardID = uint32(shard)
	}

	// New RF1 params
	blockID, _ := extractQueryParam(vals, urlParamBlockID)
	if blockID, err := uuid.Parse(blockID); err == nil {
		req.BlockID = blockID.String()
	}

	startPage, _ := extractQueryParam(vals, urlParamStartPage)
	if startPage, err := strconv.Atoi(startPage); err == nil {
		req.StartPage = uint32(startPage)
	}

	pagesToSearch, _ := extractQueryParam(vals, urlParamPagesToSearch)
	if of, err := strconv.Atoi(pagesToSearch); err == nil {
		req.PagesToSearch = uint32(of)
	}

	version, _ := extractQueryParam(vals, urlParamVersion)
	req.Version = version

	encoding, _ := extractQueryParam(vals, urlParamEncoding)
	req.Encoding = encoding

	size, _ := extractQueryParam(vals, urlParamSize)
	if size, err := strconv.Atoi(size); err == nil {
		req.Size_ = uint64(size)
	}

	footerSize, _ := extractQueryParam(vals, urlParamFooterSize)
	if footerSize, err := strconv.Atoi(footerSize); err == nil {
		req.FooterSize = uint32(footerSize)
	}

	dedicatedColumns, _ := extractQueryParam(vals, urlParamDedicatedColumns)
	if len(dedicatedColumns) > 0 {
		err := json.Unmarshal([]byte(dedicatedColumns), &req.DedicatedColumns)
		if err != nil {
			return nil, httpgrpc.Errorf(http.StatusBadRequest, fmt.Errorf("failed to parse dedicated columns: %w", err).Error())
		}
	}

	return req, nil
}

func BuildQueryInstantRequest(req *http.Request, searchReq *tempopb.QueryInstantRequest) *http.Request {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req
	}

	qb := newQueryBuilder("")
	qb.addParam(urlParamStart, strconv.FormatUint(searchReq.Start, 10))
	qb.addParam(urlParamEnd, strconv.FormatUint(searchReq.End, 10))
	qb.addParam(urlParamQuery, searchReq.Query)

	req.URL.RawQuery = qb.query()

	return req
}

func BuildQueryRangeRequest(req *http.Request, searchReq *tempopb.QueryRangeRequest) *http.Request {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req
	}

	qb := newQueryBuilder("")
	qb.addParam(urlParamStart, strconv.FormatUint(searchReq.Start, 10))
	qb.addParam(urlParamEnd, strconv.FormatUint(searchReq.End, 10))
	qb.addParam(urlParamStep, time.Duration(searchReq.Step).String())
	qb.addParam(urlParamShard, strconv.FormatUint(uint64(searchReq.ShardID), 10))
	qb.addParam(urlParamShardCount, strconv.FormatUint(uint64(searchReq.ShardCount), 10))
	qb.addParam(QueryModeKey, searchReq.QueryMode)
	// New RF1 params
	qb.addParam(urlParamBlockID, searchReq.BlockID)
	qb.addParam(urlParamStartPage, strconv.Itoa(int(searchReq.StartPage)))
	qb.addParam(urlParamPagesToSearch, strconv.Itoa(int(searchReq.PagesToSearch)))
	qb.addParam(urlParamVersion, searchReq.Version)
	qb.addParam(urlParamEncoding, searchReq.Encoding)
	qb.addParam(urlParamSize, strconv.Itoa(int(searchReq.Size_)))
	qb.addParam(urlParamFooterSize, strconv.Itoa(int(searchReq.FooterSize)))
	if len(searchReq.DedicatedColumns) > 0 {
		columnsJSON, _ := json.Marshal(searchReq.DedicatedColumns)
		qb.addParam(urlParamDedicatedColumns, string(columnsJSON))
	}

	if len(searchReq.Query) > 0 {
		qb.addParam(urlParamQuery, searchReq.Query)
	}

	req.URL.RawQuery = qb.query()

	return req
}

func bounds(vals url.Values) (time.Time, time.Time, error) {
	var (
		now      = time.Now()
		start, _ = extractQueryParam(vals, urlParamStart)
		end, _   = extractQueryParam(vals, urlParamEnd)
		since, _ = extractQueryParam(vals, urlParamSince)
	)

	return determineBounds(now, start, end, since)
}

func determineBounds(now time.Time, startString, endString, sinceString string) (time.Time, time.Time, error) {
	since := defaultSince
	if sinceString != "" {
		d, err := model.ParseDuration(sinceString)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("could not parse 'since' parameter: %w", err)
		}
		since = time.Duration(d)
	}

	end, err := parseTimestamp(endString, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("could not parse 'end' parameter: %w", err)
	}

	// endOrNow is used to apply a default for the start time or an offset if 'since' is provided.
	// we want to use the 'end' time so long as it's not in the future as this should provide
	// a more intuitive experience when end time is in the future.
	endOrNow := end
	if end.After(now) {
		endOrNow = now
	}

	start, err := parseTimestamp(startString, endOrNow.Add(-since))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("could not parse 'start' parameter: %w", err)
	}

	return start, end, nil
}

// parseTimestamp parses a ns unix timestamp from a string
// if the value is empty it returns a default value passed as second parameter
func parseTimestamp(value string, def time.Time) (time.Time, error) {
	if value == "" {
		return def, nil
	}

	if strings.Contains(value, ".") {
		if t, err := strconv.ParseFloat(value, 64); err == nil {
			s, ns := math.Modf(t)
			ns = math.Round(ns*1000) / 1000
			return time.Unix(int64(s), int64(ns*float64(time.Second))), nil
		}
	}
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return ts, nil
		}
		return time.Time{}, err
	}
	if len(value) <= 10 {
		return time.Unix(nanos, 0), nil
	}
	return time.Unix(0, nanos), nil
}

func step(vals url.Values, start, end time.Time) (time.Duration, error) {
	value, _ := extractQueryParam(vals, urlParamStep)
	if value == "" {
		return time.Duration(traceql.DefaultQueryRangeStep(uint64(start.UnixNano()), uint64(end.UnixNano()))), nil
	}
	return parseSecondsOrDuration(value)
}

func parseSecondsOrDuration(value string) (time.Duration, error) {
	if d, err := strconv.ParseFloat(value, 64); err == nil {
		ts := d * float64(time.Second)
		if ts > float64(math.MaxInt64) || ts < float64(math.MinInt64) {
			return 0, fmt.Errorf("cannot parse %q to a valid duration. It overflows int64", value)
		}
		return time.Duration(ts), nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		return time.Duration(d), nil
	}
	return 0, fmt.Errorf("cannot parse %q to a valid duration", value)
}

// BuildSearchRequest takes a tempopb.SearchRequest and populates the passed http.Request
// with the appropriate params. If no http.Request is provided a new one is created.
func BuildSearchRequest(req *http.Request, searchReq *tempopb.SearchRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req, nil
	}

	qb := newQueryBuilder("")
	qb.addParam(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	qb.addParam(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	if searchReq.Limit != 0 {
		qb.addParam(urlParamLimit, strconv.FormatUint(uint64(searchReq.Limit), 10))
	}
	if searchReq.MaxDurationMs != 0 {
		qb.addParam(urlParamMaxDuration, strconv.FormatUint(uint64(searchReq.MaxDurationMs), 10)+"ms")
	}
	if searchReq.MinDurationMs != 0 {
		qb.addParam(urlParamMinDuration, strconv.FormatUint(uint64(searchReq.MinDurationMs), 10)+"ms")
	}
	if searchReq.SpansPerSpanSet != 0 {
		qb.addParam(urlParamSpansPerSpanSet, strconv.FormatUint(uint64(searchReq.SpansPerSpanSet), 10))
	}

	if len(searchReq.Query) > 0 {
		qb.addParam(urlParamQuery, searchReq.Query)
	}

	if len(searchReq.Tags) > 0 {
		builder := &strings.Builder{}
		encoder := logfmt.NewEncoder(builder)

		for k, v := range searchReq.Tags {
			err := encoder.EncodeKeyval(k, v)
			if err != nil {
				return nil, err
			}
		}

		qb.addParam(urlParamTags, builder.String())
	}

	req.URL.RawQuery = qb.query()

	return req, nil
}

// BuildSearchBlockRequest takes a tempopb.SearchBlockRequest and populates the passed http.Request
// with the appropriate params. If no http.Request is provided a new one is created.
func BuildSearchBlockRequest(req *http.Request, searchReq *tempopb.SearchBlockRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	req, err := BuildSearchRequest(req, searchReq.SearchReq)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder(req.URL.RawQuery)
	qb.addParam(urlParamBlockID, searchReq.BlockID)
	qb.addParam(urlParamPagesToSearch, strconv.FormatUint(uint64(searchReq.PagesToSearch), 10))
	qb.addParam(urlParamSize, strconv.FormatUint(searchReq.Size_, 10))
	qb.addParam(urlParamStartPage, strconv.FormatUint(uint64(searchReq.StartPage), 10))
	qb.addParam(urlParamEncoding, searchReq.Encoding)
	qb.addParam(urlParamIndexPageSize, strconv.FormatUint(uint64(searchReq.IndexPageSize), 10))
	qb.addParam(urlParamTotalRecords, strconv.FormatUint(uint64(searchReq.TotalRecords), 10))
	qb.addParam(urlParamDataEncoding, searchReq.DataEncoding)
	qb.addParam(urlParamVersion, searchReq.Version)
	qb.addParam(urlParamFooterSize, strconv.FormatUint(uint64(searchReq.FooterSize), 10))
	if len(searchReq.DedicatedColumns) > 0 {
		columnsJSON, err := json.Marshal(searchReq.DedicatedColumns)
		if err != nil {
			return nil, err
		}
		qb.addParam(urlParamDedicatedColumns, string(columnsJSON))
	}

	req.URL.RawQuery = qb.query()

	return req, nil
}

// AddServerlessParams takes an already existing http.Request and adds maxBytes
// to it
func AddServerlessParams(req *http.Request, maxBytes int) *http.Request {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	qb := newQueryBuilder(req.URL.RawQuery)
	qb.addParam(urlParamMaxBytes, strconv.FormatInt(int64(maxBytes), 10))
	req.URL.RawQuery = qb.query()

	return req
}

// ExtractServerlessParams extracts params for the serverless functions from
// an http.Request
func ExtractServerlessParams(req *http.Request) (int, error) {
	s, exists := extractQueryParam(req.URL.Query(), urlParamMaxBytes)
	if !exists {
		return 0, nil
	}
	maxBytes, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid maxBytes: %w", err)
	}

	return int(maxBytes), nil
}

func extractQueryParam(v url.Values, param string) (string, bool) {
	value := v.Get(param)
	return value, value != ""
}

// ValidateAndSanitizeRequest validates params for trace by id api
// return values are (blockStart, blockEnd, queryMode, start, end, error)
func ValidateAndSanitizeRequest(r *http.Request) (string, string, string, int64, int64, error) {
	vals := r.URL.Query()

	q, _ := extractQueryParam(vals, QueryModeKey)

	// validate queryMode. it should either be empty or one of (QueryModeIngesters|QueryModeBlocks|QueryModeAll)
	var queryMode string
	var startTime int64
	var endTime int64
	var blockStart string
	var blockEnd string
	if len(q) == 0 || q == QueryModeAll {
		queryMode = QueryModeAll
	} else if q == QueryModeIngesters {
		queryMode = QueryModeIngesters
	} else if q == QueryModeBlocks {
		queryMode = QueryModeBlocks
	} else {
		return "", "", "", 0, 0, fmt.Errorf("invalid value for mode %s", q)
	}

	// no need to validate/sanitize other parameters if queryMode == QueryModeIngesters
	if queryMode == QueryModeIngesters {
		return "", "", queryMode, 0, 0, nil
	}

	if start, ok := extractQueryParam(vals, BlockStartKey); ok {
		_, err := uuid.Parse(start)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid value for blockstart: %w", err)
		}
		blockStart = start
	} else {
		blockStart = tempodb.BlockIDMin
	}

	if end, ok := extractQueryParam(vals, BlockEndKey); ok {
		_, err := uuid.Parse(end)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid value for blockEnd: %w", err)
		}
		blockEnd = end
	} else {
		blockEnd = tempodb.BlockIDMax
	}

	if s, ok := extractQueryParam(vals, urlParamStart); ok {
		var err error
		startTime, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid start: %w", err)
		}
	} else {
		startTime = 0
	}

	if s, ok := extractQueryParam(vals, urlParamEnd); ok {
		var err error
		endTime, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid end: %w", err)
		}
	} else {
		endTime = 0
	}

	if startTime != 0 && endTime != 0 && endTime <= startTime {
		return "", "", "", 0, 0, fmt.Errorf("http parameter start must be before end. received start=%d end=%d", startTime, endTime)
	}
	return blockStart, blockEnd, queryMode, startTime, endTime, nil
}
