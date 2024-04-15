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

	query, _ := extractQueryParam(r, urlParamQuery)

	req := &tempopb.SearchTagValuesRequest{
		TagName: tagName,
		Query:   query,
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

func ParseSearchTagsRequest(r *http.Request) (*tempopb.SearchTagsRequest, error) {
	scope, _ := extractQueryParam(r, urlParamScope)

	attScope := traceql.AttributeScopeFromString(scope)
	if attScope == traceql.AttributeScopeUnknown && scope != ParamScopeIntrinsic {
		return nil, fmt.Errorf("invalid scope: %s", scope)
	}

	req := &tempopb.SearchTagsRequest{}
	req.Scope = scope

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

func BuildSearchTagsRequest(req *http.Request, searchReq *tempopb.SearchTagsRequest) (*http.Request, error) {
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
	q.Set(urlParamScope, searchReq.Scope)

	req.URL.RawQuery = q.Encode()

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

	req.URL.RawQuery = q.Encode()

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

	q := req.URL.Query()
	q.Set(urlParamStart, strconv.FormatUint(uint64(searchReq.Start), 10))
	q.Set(urlParamEnd, strconv.FormatUint(uint64(searchReq.End), 10))
	q.Set(urlParamQuery, searchReq.Query)

	req.URL.RawQuery = q.Encode()

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

	req.URL.RawQuery = q.Encode()

	return req, nil
}
