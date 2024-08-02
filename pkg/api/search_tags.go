package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	MuxVarTagName = "tagName"

	ParamScopeIntrinsic = "intrinsic"
)

// ParseSearchBlockRequest parses all http parameters necessary to perform a block search.
func ParseSearchBlockRequest(r *http.Request) (*tempopb.SearchBlockRequest, error) {
	searchReq, err := ParseSearchRequest(r)
	if err != nil {
		return nil, err
	}
	if searchReq.Limit == 0 {
		searchReq.Limit = defaultLimit
	}

	// start and end = 0 is NOT fine for a block search request
	if searchReq.End == 0 {
		return nil, errors.New("start and end required")
	}

	req := &tempopb.SearchBlockRequest{
		SearchReq: searchReq,
	}

	vals := r.URL.Query()

	s := vals.Get(urlParamStartPage)
	startPage, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid startPage: %w", err)
	}
	if startPage < 0 {
		return nil, fmt.Errorf("startPage must be non-negative. received: %s", s)
	}
	req.StartPage = uint32(startPage)

	s = vals.Get(urlParamPagesToSearch)
	pagesToSearch64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid pagesToSearch %s: %w", s, err)
	}
	if pagesToSearch64 <= 0 {
		return nil, fmt.Errorf("pagesToSearch must be greater than 0. received: %s", s)
	}
	req.PagesToSearch = uint32(pagesToSearch64)

	s = vals.Get(urlParamBlockID)
	blockID, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid blockID: %w", err)
	}
	req.BlockID = blockID.String()

	s = vals.Get(urlParamEncoding)
	encoding, err := backend.ParseEncoding(s)
	if err != nil {
		return nil, err
	}
	req.Encoding = encoding.String()

	s = vals.Get(urlParamIndexPageSize)
	indexPageSize, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid indexPageSize %s: %w", s, err)
	}
	req.IndexPageSize = uint32(indexPageSize)

	s = vals.Get(urlParamTotalRecords)
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
	dataEncoding := vals.Get(urlParamDataEncoding)
	req.DataEncoding = dataEncoding

	version := vals.Get(urlParamVersion)
	if version == "" {
		return nil, errors.New("version required")
	}
	req.Version = version

	s = vals.Get(urlParamSize)
	size, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size %s: %w", s, err)
	}
	req.Size_ = size

	// Footer size can be 0 for some blocks, just ensure we
	// get a valid integer.
	f := vals.Get(urlParamFooterSize)
	footerSize, err := strconv.ParseUint(f, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid footerSize %s: %w", f, err)
	}
	req.FooterSize = uint32(footerSize)

	s = vals.Get(urlParamDedicatedColumns)
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

func ParseSearchTagValuesBlockRequest(r *http.Request) (*tempopb.SearchTagValuesBlockRequest, error) {
	return parseSearchTagValuesBlockRequest(r, false)
}

func ParseSearchTagValuesBlockRequestV2(r *http.Request) (*tempopb.SearchTagValuesBlockRequest, error) {
	return parseSearchTagValuesBlockRequest(r, true)
}

func parseSearchTagValuesBlockRequest(r *http.Request, enforceTraceQL bool) (*tempopb.SearchTagValuesBlockRequest, error) {
	var tagSearchReq *tempopb.SearchTagValuesRequest
	var err error
	if !enforceTraceQL {
		tagSearchReq, err = ParseSearchTagValuesRequest(r)
	} else {
		tagSearchReq, err = ParseSearchTagValuesRequestV2(r)
	}

	if err != nil {
		return nil, err
	}

	// start and end = 0 is NOT fine for a block search request
	if tagSearchReq.End == 0 {
		return nil, errors.New("start and end required")
	}

	req := &tempopb.SearchTagValuesBlockRequest{
		SearchReq: tagSearchReq,
	}

	vals := r.URL.Query()

	s := vals.Get(urlParamStartPage)
	startPage, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid startPage: %w", err)
	}
	if startPage < 0 {
		return nil, fmt.Errorf("startPage must be non-negative. received: %s", s)
	}
	req.StartPage = uint32(startPage)

	s = vals.Get(urlParamPagesToSearch)
	pagesToSearch64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid pagesToSearch %s: %w", s, err)
	}
	if pagesToSearch64 <= 0 {
		return nil, fmt.Errorf("pagesToSearch must be greater than 0. received: %s", s)
	}
	req.PagesToSearch = uint32(pagesToSearch64)

	s = vals.Get(urlParamBlockID)
	blockID, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid blockID: %w", err)
	}
	req.BlockID = blockID.String()

	s = vals.Get(urlParamEncoding)
	encoding, err := backend.ParseEncoding(s)
	if err != nil {
		return nil, err
	}
	req.Encoding = encoding.String()

	s = vals.Get(urlParamIndexPageSize)
	indexPageSize, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid indexPageSize %s: %w", s, err)
	}
	req.IndexPageSize = uint32(indexPageSize)

	s = vals.Get(urlParamTotalRecords)
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
	dataEncoding := vals.Get(urlParamDataEncoding)
	req.DataEncoding = dataEncoding

	version := vals.Get(urlParamVersion)
	if version == "" {
		return nil, errors.New("version required")
	}
	req.Version = version

	s = vals.Get(urlParamSize)
	size, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size %s: %w", s, err)
	}
	req.Size_ = size

	// Footer size can be 0 for some blocks, just ensure we
	// get a valid integer.
	f := vals.Get(urlParamFooterSize)
	footerSize, err := strconv.ParseUint(f, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid footerSize %s: %w", f, err)
	}
	req.FooterSize = uint32(footerSize)

	s = vals.Get(urlParamDedicatedColumns)
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

func ParseSearchTagsBlockRequest(r *http.Request) (*tempopb.SearchTagsBlockRequest, error) {
	tagSearchReq, err := ParseSearchTagsRequest(r)
	if err != nil {
		return nil, err
	}

	// start and end = 0 is NOT fine for a block search request
	if tagSearchReq.End == 0 {
		return nil, errors.New("start and end required")
	}

	req := &tempopb.SearchTagsBlockRequest{
		SearchReq: tagSearchReq,
	}

	vals := r.URL.Query()

	s := vals.Get(urlParamStartPage)
	startPage, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid startPage: %w", err)
	}
	if startPage < 0 {
		return nil, fmt.Errorf("startPage must be non-negative. received: %s", s)
	}
	req.StartPage = uint32(startPage)

	s = vals.Get(urlParamPagesToSearch)
	pagesToSearch64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid pagesToSearch %s: %w", s, err)
	}
	if pagesToSearch64 <= 0 {
		return nil, fmt.Errorf("pagesToSearch must be greater than 0. received: %s", s)
	}
	req.PagesToSearch = uint32(pagesToSearch64)

	s = vals.Get(urlParamBlockID)
	blockID, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid blockID: %w", err)
	}
	req.BlockID = blockID.String()

	s = vals.Get(urlParamEncoding)
	encoding, err := backend.ParseEncoding(s)
	if err != nil {
		return nil, err
	}
	req.Encoding = encoding.String()

	s = vals.Get(urlParamIndexPageSize)
	indexPageSize, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid indexPageSize %s: %w", s, err)
	}
	req.IndexPageSize = uint32(indexPageSize)

	s = vals.Get(urlParamTotalRecords)
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
	dataEncoding := vals.Get(urlParamDataEncoding)
	req.DataEncoding = dataEncoding

	version := vals.Get(urlParamVersion)
	if version == "" {
		return nil, errors.New("version required")
	}
	req.Version = version

	s = vals.Get(urlParamSize)
	size, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size %s: %w", s, err)
	}
	req.Size_ = size

	// Footer size can be 0 for some blocks, just ensure we
	// get a valid integer.
	f := vals.Get(urlParamFooterSize)
	footerSize, err := strconv.ParseUint(f, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid footerSize %s: %w", f, err)
	}
	req.FooterSize = uint32(footerSize)

	s = vals.Get(urlParamDedicatedColumns)
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

// ParseSearchTagValuesRequest handles parsing of requests from /api/search/tags/{tagName}/values and /api/v2/search/tags/{tagName}/values
func ParseSearchTagValuesRequest(r *http.Request) (*tempopb.SearchTagValuesRequest, error) {
	return parseSearchTagValuesRequest(r, false)
}

func ParseSearchTagValuesRequestV2(r *http.Request) (*tempopb.SearchTagValuesRequest, error) {
	return parseSearchTagValuesRequest(r, true)
}

func parseSearchTagValuesRequest(r *http.Request, enforceTraceQL bool) (*tempopb.SearchTagValuesRequest, error) {
	vars := mux.Vars(r)
	escapedTagName, ok := vars[MuxVarTagName]
	if !ok {
		return nil, errors.New("please provide a tagName")
	}

	if escapedTagName == "" {
		return nil, errors.New("please provide a non-empty tagName")
	}

	tagName, unescapingError := url.QueryUnescape(escapedTagName)
	if unescapingError != nil {
		return nil, errors.New("error in unescaping tagName")
	}

	if enforceTraceQL {
		_, err := traceql.ParseIdentifier(tagName)
		if err != nil {
			return nil, fmt.Errorf("please provide a valid tagName: %w", err)
		}
	}

	vals := r.URL.Query()
	query, _ := extractQueryParam(vals, urlParamQuery)

	req := &tempopb.SearchTagValuesRequest{
		TagName: tagName,
		Query:   query,
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

func ParseSearchTagsRequest(r *http.Request) (*tempopb.SearchTagsRequest, error) {
	vals := r.URL.Query()
	scope, _ := extractQueryParam(vals, urlParamScope)
	query, _ := extractQueryParam(vals, urlParamQuery)

	attScope := traceql.AttributeScopeFromString(scope)
	if attScope == traceql.AttributeScopeUnknown && scope != ParamScopeIntrinsic {
		return nil, fmt.Errorf("invalid scope: %s", scope)
	}

	req := &tempopb.SearchTagsRequest{
		Query: query,
		Scope: scope,
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

func BuildSearchTagsRequest(req *http.Request, searchReq *tempopb.SearchTagsRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req, nil
	}

	qb := NewQueryBuilder("")
	qb.AddParam(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	qb.AddParam(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	qb.AddParam(urlParamScope, searchReq.Scope)
	qb.AddParam(urlParamQuery, searchReq.Query)

	req.URL.RawQuery = qb.Query()

	return req, nil
}

func BuildSearchTagsBlockRequest(req *http.Request, searchReq *tempopb.SearchTagsBlockRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	req, err := BuildSearchTagsRequest(req, searchReq.SearchReq)
	if err != nil {
		return nil, err
	}

	q := NewQueryBuilder(req.URL.RawQuery)
	q.AddParam(urlParamSize, strconv.FormatUint(searchReq.Size_, 10))
	q.AddParam(urlParamBlockID, searchReq.BlockID)
	q.AddParam(urlParamStartPage, strconv.FormatUint(uint64(searchReq.StartPage), 10))
	q.AddParam(urlParamPagesToSearch, strconv.FormatUint(uint64(searchReq.PagesToSearch), 10))
	q.AddParam(urlParamEncoding, searchReq.Encoding)
	q.AddParam(urlParamIndexPageSize, strconv.FormatUint(uint64(searchReq.IndexPageSize), 10))
	q.AddParam(urlParamTotalRecords, strconv.FormatUint(uint64(searchReq.TotalRecords), 10))
	q.AddParam(urlParamDataEncoding, searchReq.DataEncoding)
	q.AddParam(urlParamVersion, searchReq.Version)
	q.AddParam(urlParamFooterSize, strconv.FormatUint(uint64(searchReq.FooterSize), 10))

	req.URL.RawQuery = q.Query()

	return req, nil
}

func BuildSearchTagValuesRequest(req *http.Request, searchReq *tempopb.SearchTagValuesRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	if searchReq == nil {
		return req, nil
	}

	qb := NewQueryBuilder("")
	qb.AddParam(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	qb.AddParam(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	qb.AddParam(urlParamQuery, searchReq.Query)

	req.URL.RawQuery = qb.Query()

	return req, nil
}

func BuildSearchTagValuesBlockRequest(req *http.Request, searchReq *tempopb.SearchTagValuesBlockRequest) (*http.Request, error) {
	if req == nil {
		req = &http.Request{
			URL: &url.URL{},
		}
	}

	req, err := BuildSearchTagValuesRequest(req, searchReq.SearchReq)
	if err != nil {
		return nil, err
	}

	qb := NewQueryBuilder(req.URL.RawQuery)
	qb.AddParam(urlParamSize, strconv.FormatUint(searchReq.Size_, 10))
	qb.AddParam(urlParamBlockID, searchReq.BlockID)
	qb.AddParam(urlParamStartPage, strconv.FormatUint(uint64(searchReq.StartPage), 10))
	qb.AddParam(urlParamPagesToSearch, strconv.FormatUint(uint64(searchReq.PagesToSearch), 10))
	qb.AddParam(urlParamEncoding, searchReq.Encoding)
	qb.AddParam(urlParamIndexPageSize, strconv.FormatUint(uint64(searchReq.IndexPageSize), 10))
	qb.AddParam(urlParamTotalRecords, strconv.FormatUint(uint64(searchReq.TotalRecords), 10))
	qb.AddParam(urlParamDataEncoding, searchReq.DataEncoding)
	qb.AddParam(urlParamVersion, searchReq.Version)
	qb.AddParam(urlParamFooterSize, strconv.FormatUint(uint64(searchReq.FooterSize), 10))

	req.URL.RawQuery = qb.Query()

	return req, nil
}
