package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/util"
)

const (
	urlParamTraceID    = "traceID"
	URLParamLimit      = "limit"
	URLParamStart      = "start"
	URLParamEnd        = "end"
	URLParamStartPage  = "startPage"
	URLParamTotalPages = "totalPages"
	URLParamBlockID    = "blockID"

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

	if start == 0 || end == 0 {
		err = errors.New("please provide non-zero values for http parameters start and end")
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
		startPage = uint32(startPage64)
	}

	if s := r.URL.Query().Get(URLParamTotalPages); s != "" {
		totalPages64, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
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
