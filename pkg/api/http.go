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

	PathTraces             = "/api/traces/{traceID}"
	PathSearch             = "/api/search"
	PathSearchTags         = "/api/search/tags"
	PathSearchTagValues    = "/api/search/tag/{" + MuxVarTagName + "}/values"
	PathEcho               = "/api/echo"
	PathBuildInfo          = "/api/status/buildinfo"
	PathUsageStats         = "/status/usage-stats"
	PathSpanMetrics        = "/api/metrics"
	PathSpanMetricsSummary = "/api/metrics/summary"
	PathMetricsQueryRange  = "/api/metrics/query_range"

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

	if s, ok := extractQueryParam(r, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(r, urlParamEnd); ok {
		end, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		req.End = uint32(end)
	}

	query, queryFound := extractQueryParam(r, urlParamQuery)
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

	encodedTags, tagsFound := extractQueryParam(r, urlParamTags)
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
		for k, v := range r.URL.Query() {
			// Skip reserved keywords
			if k == urlParamQuery || k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == urlParamLimit || k == urlParamSpansPerSpanSet || k == urlParamStart || k == urlParamEnd {
				continue
			}

			if len(v) > 0 && v[0] != "" {
				req.Tags[k] = v[0]
			}
		}
	}

	if s, ok := extractQueryParam(r, urlParamMinDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid minDuration: %w", err)
		}
		req.MinDurationMs = uint32(dur.Milliseconds())
	}

	if s, ok := extractQueryParam(r, urlParamMaxDuration); ok {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid maxDuration: %w", err)
		}
		req.MaxDurationMs = uint32(dur.Milliseconds())

		if req.MinDurationMs != 0 && req.MinDurationMs > req.MaxDurationMs {
			return nil, errors.New("invalid maxDuration: must be greater than minDuration")
		}
	}

	if s, ok := extractQueryParam(r, urlParamLimit); ok {
		limit, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 {
			return nil, errors.New("invalid limit: must be a positive number")
		}
		req.Limit = uint32(limit)
	}

	if s, ok := extractQueryParam(r, urlParamSpansPerSpanSet); ok {
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

	groupBy := r.URL.Query().Get(urlParamGroupBy)
	req.GroupBy = groupBy

	query := r.URL.Query().Get(urlParamQuery)
	req.Query = query

	l := r.URL.Query().Get(urlParamLimit)
	if l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		req.Limit = uint64(limit)
	}

	if s, ok := extractQueryParam(r, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(r, urlParamEnd); ok {
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

	groupBy := r.URL.Query().Get(urlParamGroupBy)
	req.GroupBy = groupBy

	query := r.URL.Query().Get(urlParamQuery)
	req.Query = query

	l := r.URL.Query().Get(urlParamLimit)
	if l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
		req.Limit = uint64(limit)
	}

	if s, ok := extractQueryParam(r, urlParamStart); ok {
		start, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		req.Start = uint32(start)
	}

	if s, ok := extractQueryParam(r, urlParamEnd); ok {
		end, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		req.End = uint32(end)
	}

	return req, nil
}

func ParseQueryRangeRequest(r *http.Request) (*tempopb.QueryRangeRequest, error) {
	req := &tempopb.QueryRangeRequest{}

	if err := r.ParseForm(); err != nil {
		return nil, httpgrpc.Errorf(http.StatusBadRequest, err.Error())
	}

	req.Query = r.Form.Get("query")
	req.QueryMode = r.Form.Get(QueryModeKey)

	start, end, _ := bounds(r)
	req.Start = uint64(start.UnixNano())
	req.End = uint64(end.UnixNano())

	step, err := step(r, start, end)
	if err != nil {
		return nil, httpgrpc.Errorf(http.StatusBadRequest, err.Error())
	}
	req.Step = uint64(step.Nanoseconds())

	if of, err := strconv.Atoi(r.Form.Get(urlParamShardCount)); err == nil {
		req.ShardCount = uint32(of)
	}
	if shard, err := strconv.Atoi(r.Form.Get(urlParamShard)); err == nil {
		req.ShardID = uint32(shard)
	}

	return req, nil
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

	q := req.URL.Query()
	q.Set(urlParamStart, strconv.FormatUint(searchReq.Start, 10))
	q.Set(urlParamEnd, strconv.FormatUint(searchReq.End, 10))
	q.Set(urlParamStep, time.Duration(searchReq.Step).String())
	q.Set(urlParamShard, strconv.FormatUint(uint64(searchReq.ShardID), 10))
	q.Set(urlParamShardCount, strconv.FormatUint(uint64(searchReq.ShardCount), 10))
	q.Set(QueryModeKey, searchReq.QueryMode)

	if len(searchReq.Query) > 0 {
		q.Set(urlParamQuery, searchReq.Query)
	}
	req.URL.RawQuery = q.Encode()

	return req
}

func bounds(r *http.Request) (time.Time, time.Time, error) {
	var (
		now   = time.Now()
		start = r.Form.Get(urlParamStart)
		end   = r.Form.Get(urlParamEnd)
		since = r.Form.Get(urlParamSince)
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

func step(r *http.Request, start, end time.Time) (time.Duration, error) {
	value := r.Form.Get(urlParamStep)
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
	if d, err := model.ParseDuration(value); err == nil {
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

	q := req.URL.Query()
	q.Set(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	q.Set(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	if searchReq.Limit != 0 {
		q.Set(urlParamLimit, strconv.FormatUint(uint64(searchReq.Limit), 10))
	}
	if searchReq.MaxDurationMs != 0 {
		q.Set(urlParamMaxDuration, strconv.FormatUint(uint64(searchReq.MaxDurationMs), 10)+"ms")
	}
	if searchReq.MinDurationMs != 0 {
		q.Set(urlParamMinDuration, strconv.FormatUint(uint64(searchReq.MinDurationMs), 10)+"ms")
	}
	if searchReq.SpansPerSpanSet != 0 {
		q.Set(urlParamSpansPerSpanSet, strconv.FormatUint(uint64(searchReq.SpansPerSpanSet), 10))
	}

	if len(searchReq.Query) > 0 {
		q.Set(urlParamQuery, searchReq.Query)
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

		q.Set(urlParamTags, builder.String())
	}

	req.URL.RawQuery = q.Encode()

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

	q := req.URL.Query()
	q.Set(urlParamSize, strconv.FormatUint(searchReq.Size_, 10))
	q.Set(urlParamBlockID, searchReq.BlockID)
	q.Set(urlParamStartPage, strconv.FormatUint(uint64(searchReq.StartPage), 10))
	q.Set(urlParamPagesToSearch, strconv.FormatUint(uint64(searchReq.PagesToSearch), 10))
	q.Set(urlParamEncoding, searchReq.Encoding)
	q.Set(urlParamIndexPageSize, strconv.FormatUint(uint64(searchReq.IndexPageSize), 10))
	q.Set(urlParamTotalRecords, strconv.FormatUint(uint64(searchReq.TotalRecords), 10))
	q.Set(urlParamDataEncoding, searchReq.DataEncoding)
	q.Set(urlParamVersion, searchReq.Version)
	q.Set(urlParamFooterSize, strconv.FormatUint(uint64(searchReq.FooterSize), 10))
	if len(searchReq.DedicatedColumns) > 0 {
		columnsJSON, err := json.Marshal(searchReq.DedicatedColumns)
		if err != nil {
			return nil, err
		}
		q.Set(urlParamDedicatedColumns, string(columnsJSON))
	}

	req.URL.RawQuery = q.Encode()

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

	q := req.URL.Query()
	q.Set(urlParamMaxBytes, strconv.FormatInt(int64(maxBytes), 10))
	req.URL.RawQuery = q.Encode()

	return req
}

// ExtractServerlessParams extracts params for the serverless functions from
// an http.Request
func ExtractServerlessParams(req *http.Request) (int, error) {
	s, exists := extractQueryParam(req, urlParamMaxBytes)
	if !exists {
		return 0, nil
	}
	maxBytes, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid maxBytes: %w", err)
	}

	return int(maxBytes), nil
}

func extractQueryParam(r *http.Request, param string) (string, bool) {
	value := r.URL.Query().Get(param)
	return value, value != ""
}

// ValidateAndSanitizeRequest validates params for trace by id api
// return values are (blockStart, blockEnd, queryMode, start, end, error)
func ValidateAndSanitizeRequest(r *http.Request) (string, string, string, int64, int64, error) {
	q, _ := extractQueryParam(r, QueryModeKey)

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

	if start, ok := extractQueryParam(r, BlockStartKey); ok {
		_, err := uuid.Parse(start)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid value for blockstart: %w", err)
		}
		blockStart = start
	} else {
		blockStart = tempodb.BlockIDMin
	}

	if end, ok := extractQueryParam(r, BlockEndKey); ok {
		_, err := uuid.Parse(end)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid value for blockEnd: %w", err)
		}
		blockEnd = end
	} else {
		blockEnd = tempodb.BlockIDMax
	}

	if s, ok := extractQueryParam(r, urlParamStart); ok {
		var err error
		startTime, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("invalid start: %w", err)
		}
	} else {
		startTime = 0
	}

	if s, ok := extractQueryParam(r, urlParamEnd); ok {
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
