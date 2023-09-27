package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

const (
	muxVarTagName = "tagName"

	ParamScopeIntrinsic = "intrinsic"
)

// ParseSearchTagValuesRequest handles parsing of requests from /api/search/tags/{tagName}/values and /api/v2/search/tags/{tagName}/values
func ParseSearchTagValuesRequest(r *http.Request) (*tempopb.SearchTagValuesRequest, error) {
	return parseSearchTagValuesRequest(r, false)
}

func ParseSearchTagValuesRequestV2(r *http.Request) (*tempopb.SearchTagValuesRequest, error) {
	return parseSearchTagValuesRequest(r, true)
}

func parseSearchTagValuesRequest(r *http.Request, enforceTraceQL bool) (*tempopb.SearchTagValuesRequest, error) {
	vars := mux.Vars(r)
	escapedTagName, ok := vars[muxVarTagName]
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

	return req, nil
}

func ParseSearchTagsRequest(r *http.Request) (*tempopb.SearchTagsRequest, error) {
	scope, _ := extractQueryParam(r, urlParamScope)

	attScope := traceql.AttributeScopeFromString(scope)
	if attScope == traceql.AttributeScopeUnknown && scope != ParamScopeIntrinsic {
		return nil, fmt.Errorf("invalid scope: %s", scope)
	}

	return &tempopb.SearchTagsRequest{
		Scope: scope,
	}, nil
}
