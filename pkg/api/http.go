package api

import (
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
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	urlParamTraceID = "traceID"
	// search
	urlParamTags        = "tags"
	urlParamMinDuration = "minDuration"
	urlParamMaxDuration = "maxDuration"
	urlParamLimit       = "limit"
	urlParamStart       = "start"
	urlParamEnd         = "end"
	// backend search querier
	urlParamStartPage  = "startPage"
	urlParamTotalPages = "totalPages"
	urlParamBlockID    = "blockID"
	// backend search serverless
	urlParamEncoding      = "encoding"
	urlParamIndexPageSize = "indexPageSize"
	urlParamTotalRecords  = "totalRecords"
	urlParamTenant        = "tenant" // jpe remove this?
	urlParamDataEncoding  = "dataEncoding"
	urlParamVersion       = "version"

	HeaderAccept         = "Accept"
	HeaderAcceptProtobuf = "application/protobuf"
	HeaderAcceptJSON     = "application/json"

	PathPrefixQuerier = "/querier"

	PathTraces          = "/api/traces/{traceID}"
	PathSearch          = "/api/search"
	PathSearchTags      = "/api/search/tags"
	PathSearchTagValues = "/api/search/tag/{tagName}/values"
	PathEcho            = "/api/echo"

	// todo(search): make configurable
	maxRange     = 1800 // 30 minutes
	defaultLimit = 20
)

func ParseTraceID(r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)
	traceID, ok := vars[urlParamTraceID]
	if !ok {
		return nil, fmt.Errorf("please provide a traceID")
	}

	byteID, err := util.HexStringToTraceID(traceID)
	if err != nil {
		return nil, err
	}

	return byteID, nil
}

// jpe provide the reverse methods? create http.request?
func ParseSearchRequest(r *http.Request, defaultLimit uint32, maxLimit uint32) (*tempopb.SearchRequest, error) {
	req := &tempopb.SearchRequest{
		Tags:  map[string]string{},
		Limit: defaultLimit,
	}

	encodedTags, tagsFound := extractQueryParam(r, urlParamTags)

	if !tagsFound {
		// Passing tags as individual query parameters is not supported anymore, clients should use the tags
		// query parameter instead. We still parse these tags since the initial Grafana implementation uses this.
		// As Grafana gets updated and/or versions using this get old we can remove this section.
		for k, v := range r.URL.Query() {
			// Skip reserved keywords
			if k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == urlParamLimit {
				continue
			}

			if len(v) > 0 && v[0] != "" {
				req.Tags[k] = v[0]
			}
		}
	} else {
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
			if syntaxErr, ok := err.(*logfmt.SyntaxError); ok {
				return nil, fmt.Errorf("invalid tags: %s at pos %d", syntaxErr.Msg, syntaxErr.Pos)
			}
			return nil, fmt.Errorf("invalid tags: %w", err)
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

	if maxLimit != 0 && req.Limit > maxLimit {
		req.Limit = maxLimit
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

	// start and end == 0 is fine
	if req.End == 0 && req.Start == 0 {
		return req, nil
	}

	// if start or end are non-zero do some checks
	if req.End <= req.Start {
		return nil, fmt.Errorf("http parameter start must be before end. received start=%d end=%d", req.Start, req.End)
	}
	if req.End-req.Start > maxRange {
		return nil, fmt.Errorf("range specified by start and end exceeds %d seconds. received start=%d end=%d", maxRange, req.Start, req.End)
	}
	return req, nil
}

// ParseBlockSearchRequest parses all http parameters necessary to perform a block search.
func ParseSearchBlockRequest(r *http.Request, defaultLimit uint32, maxLimit uint32) (*tempopb.SearchBlockRequest, error) {
	searchReq, err := ParseSearchRequest(r, defaultLimit, maxLimit)
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

	s = r.URL.Query().Get(urlParamTotalPages)
	totalPages64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid totalPages %s: %w", s, err)
	}
	if totalPages64 <= 0 {
		return nil, fmt.Errorf("totalPages must be greater than 0. received: %s", s)
	}
	req.TotalPages = uint32(totalPages64)

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
	if indexPageSize <= 0 {
		return nil, fmt.Errorf("indexPageSize must be greater than 0. received %d", indexPageSize)
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

	dataEncoding := r.URL.Query().Get(urlParamDataEncoding)
	if dataEncoding == "" {
		return nil, errors.New("dataEncoding required")
	}
	req.DataEncoding = dataEncoding

	version := r.URL.Query().Get(urlParamVersion)
	if version == "" {
		return nil, errors.New("version required")
	}
	req.Version = version

	// jpe - restore this or pull tenant id from context and return?
	// tenant = r.URL.Query().Get(URLParamTenant)
	// if tenant == "" {
	// 	err = errors.New("tenant required")
	// 	return
	// }

	return req, nil
}

// BuildSearchRequest takes a tempopb.SearchRequest and populates the passed http.Request
// with the appropriate params. If no http.Request is provided a new one is created.
// jpe test
func BuildSearchRequest(req *http.Request, searchReq *tempopb.SearchRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req, nil
	}

	// jpe add all searchReq.SearchReq params to req
	q := req.URL.Query()
	q.Set(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	q.Set(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	q.Set(urlParamLimit, strconv.FormatUint(uint64(searchReq.Limit), 10))
	q.Set(urlParamMaxDuration, strconv.FormatUint(uint64(searchReq.MaxDurationMs), 10)+"ms")
	q.Set(urlParamMinDuration, strconv.FormatUint(uint64(searchReq.MinDurationMs), 10)+"ms")

	builder := &strings.Builder{}
	encoder := logfmt.NewEncoder(builder)

	for k, v := range searchReq.Tags {
		err := encoder.EncodeKeyval(k, v)
		if err != nil {
			return nil, err
		}
	}

	q.Set(urlParamTags, builder.String())

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
	q.Set(urlParamBlockID, searchReq.BlockID)
	q.Set(urlParamStartPage, strconv.FormatUint(uint64(searchReq.StartPage), 10))
	q.Set(urlParamTotalPages, strconv.FormatUint(uint64(searchReq.TotalPages), 10))
	q.Set(urlParamEncoding, searchReq.Encoding)
	q.Set(urlParamIndexPageSize, strconv.FormatUint(uint64(searchReq.IndexPageSize), 10))
	q.Set(urlParamTotalRecords, strconv.FormatUint(uint64(searchReq.TotalRecords), 10))
	q.Set(urlParamDataEncoding, searchReq.DataEncoding)
	q.Set(urlParamVersion, searchReq.Version)

	req.URL.RawQuery = q.Encode()

	return req, nil
}

func extractQueryParam(r *http.Request, param string) (string, bool) {
	value := r.URL.Query().Get(param)
	return value, value != ""
}
