package api

import (
	"errors"
	"fmt"
	"net/http"
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
	URLParamLimit       = "limit"
	URLParamStart       = "start"
	URLParamEnd         = "end"
	// backend search querier
	URLParamStartPage  = "startPage"
	URLParamTotalPages = "totalPages"
	URLParamBlockID    = "blockID"
	// backend search serverless
	URLParamEncoding      = "encoding"
	URLParamIndexPageSize = "indexPageSize"
	URLParamTotalRecords  = "totalRecords"
	URLParamTenant        = "tenant"
	URLParamDataEncoding  = "dataEncoding"
	URLParamVersion       = "version"

	HeaderAccept         = "Accept"
	HeaderAcceptProtobuf = "application/protobuf"
	HeaderAcceptJSON     = "application/json"

	PathPrefixQuerier = "/querier"

	PathTraces          = "/api/traces/{traceID}"
	PathSearch          = "/api/search"
	PathSearchTags      = "/api/search/tags"
	PathSearchTagValues = "/api/search/tag/{tagName}/values"
	PathEcho            = "/api/echo"
	PathBackendSearch   = "/api/backend_search" // todo(search): integrate with real search

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
			if k == urlParamTags || k == urlParamMinDuration || k == urlParamMaxDuration || k == URLParamLimit {
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

	if s, ok := extractQueryParam(r, URLParamLimit); ok {
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

	return req, nil
}

// ParseBackendSearch is used by both the query frontend and querier to parse backend search requests.
// /?start=0&end=0&limit=20
func ParseBackendSearch(r *http.Request) (start, end int64, limit int, err error) {
	if s := r.URL.Query().Get(URLParamStart); s != "" {
		start, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(URLParamEnd); s != "" {
		end, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
	}

	if s := r.URL.Query().Get(URLParamLimit); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}

	if start <= 0 || end <= 0 {
		err = errors.New("please provide positive values for http parameters start and end")
		return
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if end-start > maxRange {
		err = fmt.Errorf("range specified by start and end exceeds %d seconds. received start=%d end=%d", maxRange, start, end)
		return
	}
	if end <= start {
		err = fmt.Errorf("http parameter start must be before end. received start=%d end=%d", start, end)
		return
	}

	return
}

// ParseBackendSearchQuerier is used by the querier to parse backend search requests.
// /?startPage=1&totalPages=1&blockID=0f9f8f8f-8f8f-8f8f-8f8f-8f8f8f8f8f8f
func ParseBackendSearchQuerier(r *http.Request) (startPage, totalPages uint32, blockID uuid.UUID, err error) {
	var startPage64, totalPages64 int64

	if s := r.URL.Query().Get(URLParamStartPage); s != "" {
		startPage64, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			return
		}
		if startPage64 < 0 {
			err = fmt.Errorf("startPage must be non-negative. received: %s", s)
			return
		}
		startPage = uint32(startPage64)
	}

	if s := r.URL.Query().Get(URLParamTotalPages); s != "" {
		totalPages64, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			err = fmt.Errorf("failed to parse totalPages %s: %w", s, err)
			return
		}
		if totalPages64 <= 0 {
			err = fmt.Errorf("totalPages must be greater than 0. received: %s", s)
			return
		}
		totalPages = uint32(totalPages64)
	}

	if s := r.URL.Query().Get(URLParamBlockID); s != "" {
		blockID, err = uuid.Parse(s)
		if err != nil {
			err = fmt.Errorf("blockID: %w", err)
			return
		}
	}

	if blockID == uuid.Nil {
		err = errors.New("blockID required")
		return
	}

	return
}

// ParseBackendSearchServerless is used by the serverless functionality to parse backend search requests.
// /?encoding=zstd&indexPageSize=10totalRecords=10&tenant=1
// jpe test
func ParseBackendSearchServerless(r *http.Request) (encoding backend.Encoding, dataEncoding string, indexPageSize, totalRecords uint32, tenant, version string, err error) {
	var indexPageSize64, totalRecords64 int64

	if s := r.URL.Query().Get(URLParamEncoding); s != "" {
		encoding, err = backend.ParseEncoding(s)
		if err != nil {
			err = fmt.Errorf("failed to parse encoding %s: %w", s, err)
			return
		}
	}
	if s := r.URL.Query().Get(URLParamIndexPageSize); s != "" {
		indexPageSize64, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			err = fmt.Errorf("failed to parse indexPageSize %s: %w", s, err)
			return
		}
		if indexPageSize64 < 0 {
			err = fmt.Errorf("indexPageSize must be non-negative. received %d", indexPageSize64)
			return
		}
		indexPageSize = uint32(indexPageSize64)
	}

	if s := r.URL.Query().Get(URLParamTotalRecords); s != "" {
		totalRecords64, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			err = fmt.Errorf("failed to parse totalRecords %s: %w", s, err)
			return
		}
		if totalRecords64 < 0 {
			err = fmt.Errorf("totalRecords must be non-negative. received %d", indexPageSize64)
			return
		}
		totalRecords = uint32(totalRecords64)
	}

	tenant = r.URL.Query().Get(URLParamTenant)
	if tenant == "" {
		err = errors.New("tenant required")
		return
	}

	dataEncoding = r.URL.Query().Get(URLParamDataEncoding)
	if dataEncoding == "" {
		err = errors.New("dataEncoding required")
		return
	}

	version = r.URL.Query().Get(URLParamVersion)
	if dataEncoding == "" {
		err = errors.New("version required")
		return
	}

	return
}

func extractQueryParam(r *http.Request, param string) (string, bool) {
	value := r.URL.Query().Get(param)
	return value, value != ""
}
