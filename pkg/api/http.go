package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/util"
)

const (
	urlParamTraceID = "traceID"
	URLParamLimit   = "limit"
	URLParamStart   = "start"
	URLParamEnd     = "end"

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
