package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
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

	HeaderAccept         = "Accept"
	HeaderContentType    = "Content-Type"
	HeaderAcceptProtobuf = "application/protobuf"
	HeaderAcceptJSON     = "application/json"

	PathPrefixQuerier   = "/querier"
	PathPrefixGenerator = "/generator"

	PathTraces             = "/api/traces/{traceID}"
	PathSearch             = "/api/search"
	PathSearchTags         = "/api/search/tags"
	PathSearchTagValues    = "/api/search/tag/{" + muxVarTagName + "}/values"
	PathEcho               = "/api/echo"
	PathBuildInfo          = "/api/status/buildinfo"
	PathUsageStats         = "/status/usage-stats"
	PathSpanMetrics        = "/api/metrics"
	PathSpanMetricsSummary = "/api/metrics/summary"

	// PathOverrides user configurable overrides
	PathOverrides = "/api/overrides"

	PathSearchTagValuesV2 = "/api/v2/search/tag/{" + muxVarTagName + "}/values"
	PathSearchTagsV2      = "/api/v2/search/tags"

	QueryModeKey       = "mode"
	QueryModeIngesters = "ingesters"
	QueryModeBlocks    = "blocks"
	QueryModeAll       = "all"
	BlockStartKey      = "blockStart"
	BlockEndKey        = "blockEnd"

	defaultLimit           = 20
	defaultSpansPerSpanSet = 3
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
		Limit:           defaultLimit,
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

// ParseSearchBlockRequest parses all http parameters necessary to perform a block search.
func ParseSearchBlockRequest(r *http.Request) (*tempopb.SearchBlockRequest, error) {
	searchReq, err := ParseSearchRequest(r)
	if err != nil {
		return nil, err
	}

	// start and end = 0 is NOT fine for a block search request
	if searchReq.End == 0 {
		return nil, errors.New("start and end required")
	}

	req := &tempopb.SearchBlockRequest{
		SearchReq: searchReq,
	}

	s := r.URL.Query().Get(urlParamStartPage)
	startPage, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid startPage: %w", err)
	}
	if startPage < 0 {
		return nil, fmt.Errorf("startPage must be non-negative. received: %s", s)
	}
	req.StartPage = uint32(startPage)

	s = r.URL.Query().Get(urlParamPagesToSearch)
	pagesToSearch64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid pagesToSearch %s: %w", s, err)
	}
	if pagesToSearch64 <= 0 {
		return nil, fmt.Errorf("pagesToSearch must be greater than 0. received: %s", s)
	}
	req.PagesToSearch = uint32(pagesToSearch64)

	s = r.URL.Query().Get(urlParamBlockID)
	blockID, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid blockID: %w", err)
	}
	req.BlockID = blockID.String()

	s = r.URL.Query().Get(urlParamEncoding)
	encoding, err := backend.ParseEncoding(s)
	if err != nil {
		return nil, err
	}
	req.Encoding = encoding.String()

	s = r.URL.Query().Get(urlParamIndexPageSize)
	indexPageSize, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid indexPageSize %s: %w", s, err)
	}
	req.IndexPageSize = uint32(indexPageSize)

	s = r.URL.Query().Get(urlParamTotalRecords)
	totalRecords, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid totalRecords %s: %w", s, err)
	}
	if totalRecords <= 0 {
		return nil, fmt.Errorf("totalRecords must be greater than 0. received %d", totalRecords)
	}
	req.TotalRecords = uint32(totalRecords)

	// Data encoding can be blank for some block formats, therefore
	// no validation on the param here.  Eventually we may be able
	// to remove this parameter entirely.
	dataEncoding := r.URL.Query().Get(urlParamDataEncoding)
	req.DataEncoding = dataEncoding

	version := r.URL.Query().Get(urlParamVersion)
	if version == "" {
		return nil, errors.New("version required")
	}
	req.Version = version

	s = r.URL.Query().Get(urlParamSize)
	size, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size %s: %w", s, err)
	}
	req.Size_ = size

	// Footer size can be 0 for some blocks, just ensure we
	// get a valid integer.
	f := r.URL.Query().Get(urlParamFooterSize)
	footerSize, err := strconv.ParseUint(f, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid footerSize %s: %w", f, err)
	}
	req.FooterSize = uint32(footerSize)

	s = r.URL.Query().Get(urlParamDedicatedColumns)
	if s != "" {
		var dedicatedColumns []*tempopb.DedicatedColumn
		err = json.Unmarshal([]byte(s), &dedicatedColumns)
		if err != nil {
			return nil, fmt.Errorf("invalid dedicatedColumns '%s': %w", s, err)
		}
		req.DedicatedColumns = dedicatedColumns
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
