// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// CIVisibilityPipelinesApi service type
type CIVisibilityPipelinesApi datadog.Service

type apiAggregateCIAppPipelineEventsRequest struct {
	ctx  _context.Context
	body *CIAppPipelinesAggregateRequest
}

func (a *CIVisibilityPipelinesApi) buildAggregateCIAppPipelineEventsRequest(ctx _context.Context, body CIAppPipelinesAggregateRequest) (apiAggregateCIAppPipelineEventsRequest, error) {
	req := apiAggregateCIAppPipelineEventsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// AggregateCIAppPipelineEvents Aggregate pipelines events.
// The API endpoint to aggregate CI Visibility pipeline events into buckets of computed metrics and timeseries.
func (a *CIVisibilityPipelinesApi) AggregateCIAppPipelineEvents(ctx _context.Context, body CIAppPipelinesAggregateRequest) (CIAppPipelinesAnalyticsAggregateResponse, *_nethttp.Response, error) {
	req, err := a.buildAggregateCIAppPipelineEventsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue CIAppPipelinesAnalyticsAggregateResponse
		return localVarReturnValue, nil, err
	}

	return a.aggregateCIAppPipelineEventsExecute(req)
}

// aggregateCIAppPipelineEventsExecute executes the request.
func (a *CIVisibilityPipelinesApi) aggregateCIAppPipelineEventsExecute(r apiAggregateCIAppPipelineEventsRequest) (CIAppPipelinesAnalyticsAggregateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue CIAppPipelinesAnalyticsAggregateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CIVisibilityPipelinesApi.AggregateCIAppPipelineEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/ci/pipelines/analytics/aggregate"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	// body params
	localVarPostBody = r.body
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.Client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := datadog.ReadBody(localVarHTTPResponse)
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.Client.Decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type apiListCIAppPipelineEventsRequest struct {
	ctx         _context.Context
	filterQuery *string
	filterFrom  *time.Time
	filterTo    *time.Time
	sort        *CIAppSort
	pageCursor  *string
	pageLimit   *int32
}

// ListCIAppPipelineEventsOptionalParameters holds optional parameters for ListCIAppPipelineEvents.
type ListCIAppPipelineEventsOptionalParameters struct {
	FilterQuery *string
	FilterFrom  *time.Time
	FilterTo    *time.Time
	Sort        *CIAppSort
	PageCursor  *string
	PageLimit   *int32
}

// NewListCIAppPipelineEventsOptionalParameters creates an empty struct for parameters.
func NewListCIAppPipelineEventsOptionalParameters() *ListCIAppPipelineEventsOptionalParameters {
	this := ListCIAppPipelineEventsOptionalParameters{}
	return &this
}

// WithFilterQuery sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithFilterQuery(filterQuery string) *ListCIAppPipelineEventsOptionalParameters {
	r.FilterQuery = &filterQuery
	return r
}

// WithFilterFrom sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithFilterFrom(filterFrom time.Time) *ListCIAppPipelineEventsOptionalParameters {
	r.FilterFrom = &filterFrom
	return r
}

// WithFilterTo sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithFilterTo(filterTo time.Time) *ListCIAppPipelineEventsOptionalParameters {
	r.FilterTo = &filterTo
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithSort(sort CIAppSort) *ListCIAppPipelineEventsOptionalParameters {
	r.Sort = &sort
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithPageCursor(pageCursor string) *ListCIAppPipelineEventsOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListCIAppPipelineEventsOptionalParameters) WithPageLimit(pageLimit int32) *ListCIAppPipelineEventsOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

func (a *CIVisibilityPipelinesApi) buildListCIAppPipelineEventsRequest(ctx _context.Context, o ...ListCIAppPipelineEventsOptionalParameters) (apiListCIAppPipelineEventsRequest, error) {
	req := apiListCIAppPipelineEventsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListCIAppPipelineEventsOptionalParameters is allowed")
	}

	if o != nil {
		req.filterQuery = o[0].FilterQuery
		req.filterFrom = o[0].FilterFrom
		req.filterTo = o[0].FilterTo
		req.sort = o[0].Sort
		req.pageCursor = o[0].PageCursor
		req.pageLimit = o[0].PageLimit
	}
	return req, nil
}

// ListCIAppPipelineEvents Get a list of pipelines events.
// List endpoint returns CI Visibility pipeline events that match a log search query.
// [Results are paginated similarly to logs](https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination).
//
// Use this endpoint to see your latest pipeline events.
func (a *CIVisibilityPipelinesApi) ListCIAppPipelineEvents(ctx _context.Context, o ...ListCIAppPipelineEventsOptionalParameters) (CIAppPipelineEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildListCIAppPipelineEventsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue CIAppPipelineEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.listCIAppPipelineEventsExecute(req)
}

// ListCIAppPipelineEventsWithPagination provides a paginated version of ListCIAppPipelineEvents returning a channel with all items.
func (a *CIVisibilityPipelinesApi) ListCIAppPipelineEventsWithPagination(ctx _context.Context, o ...ListCIAppPipelineEventsOptionalParameters) (<-chan datadog.PaginationResult[CIAppPipelineEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListCIAppPipelineEventsOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[CIAppPipelineEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildListCIAppPipelineEventsRequest(ctx, o...)
			if err != nil {
				var returnItem CIAppPipelineEvent
				items <- datadog.PaginationResult[CIAppPipelineEvent]{returnItem, err}
				break
			}

			resp, _, err := a.listCIAppPipelineEventsExecute(req)
			if err != nil {
				var returnItem CIAppPipelineEvent
				items <- datadog.PaginationResult[CIAppPipelineEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[CIAppPipelineEvent]{item, nil}:
				case <-ctx.Done():
					close(items)
					return
				}
			}
			if len(results) < int(pageSize_) {
				break
			}
			cursorMeta, ok := resp.GetMetaOk()
			if !ok {
				break
			}
			cursorMetaPage, ok := cursorMeta.GetPageOk()
			if !ok {
				break
			}
			cursorMetaPageAfter, ok := cursorMetaPage.GetAfterOk()
			if !ok {
				break
			}

			o[0].PageCursor = cursorMetaPageAfter
		}
		close(items)
	}()
	return items, cancel
}

// listCIAppPipelineEventsExecute executes the request.
func (a *CIVisibilityPipelinesApi) listCIAppPipelineEventsExecute(r apiListCIAppPipelineEventsRequest) (CIAppPipelineEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CIAppPipelineEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CIVisibilityPipelinesApi.ListCIAppPipelineEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/ci/pipelines/events"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filterQuery != nil {
		localVarQueryParams.Add("filter[query]", datadog.ParameterToString(*r.filterQuery, ""))
	}
	if r.filterFrom != nil {
		localVarQueryParams.Add("filter[from]", datadog.ParameterToString(*r.filterFrom, ""))
	}
	if r.filterTo != nil {
		localVarQueryParams.Add("filter[to]", datadog.ParameterToString(*r.filterTo, ""))
	}
	if r.sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*r.sort, ""))
	}
	if r.pageCursor != nil {
		localVarQueryParams.Add("page[cursor]", datadog.ParameterToString(*r.pageCursor, ""))
	}
	if r.pageLimit != nil {
		localVarQueryParams.Add("page[limit]", datadog.ParameterToString(*r.pageLimit, ""))
	}
	localVarHeaderParams["Accept"] = "application/json"

	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.Client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := datadog.ReadBody(localVarHTTPResponse)
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.Client.Decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type apiSearchCIAppPipelineEventsRequest struct {
	ctx  _context.Context
	body *CIAppPipelineEventsRequest
}

// SearchCIAppPipelineEventsOptionalParameters holds optional parameters for SearchCIAppPipelineEvents.
type SearchCIAppPipelineEventsOptionalParameters struct {
	Body *CIAppPipelineEventsRequest
}

// NewSearchCIAppPipelineEventsOptionalParameters creates an empty struct for parameters.
func NewSearchCIAppPipelineEventsOptionalParameters() *SearchCIAppPipelineEventsOptionalParameters {
	this := SearchCIAppPipelineEventsOptionalParameters{}
	return &this
}

// WithBody sets the corresponding parameter name and returns the struct.
func (r *SearchCIAppPipelineEventsOptionalParameters) WithBody(body CIAppPipelineEventsRequest) *SearchCIAppPipelineEventsOptionalParameters {
	r.Body = &body
	return r
}

func (a *CIVisibilityPipelinesApi) buildSearchCIAppPipelineEventsRequest(ctx _context.Context, o ...SearchCIAppPipelineEventsOptionalParameters) (apiSearchCIAppPipelineEventsRequest, error) {
	req := apiSearchCIAppPipelineEventsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SearchCIAppPipelineEventsOptionalParameters is allowed")
	}

	if o != nil {
		req.body = o[0].Body
	}
	return req, nil
}

// SearchCIAppPipelineEvents Search pipelines events.
// List endpoint returns CI Visibility pipeline events that match a log search query.
// [Results are paginated similarly to logs](https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination).
//
// Use this endpoint to build complex events filtering and search.
func (a *CIVisibilityPipelinesApi) SearchCIAppPipelineEvents(ctx _context.Context, o ...SearchCIAppPipelineEventsOptionalParameters) (CIAppPipelineEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchCIAppPipelineEventsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue CIAppPipelineEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.searchCIAppPipelineEventsExecute(req)
}

// SearchCIAppPipelineEventsWithPagination provides a paginated version of SearchCIAppPipelineEvents returning a channel with all items.
func (a *CIVisibilityPipelinesApi) SearchCIAppPipelineEventsWithPagination(ctx _context.Context, o ...SearchCIAppPipelineEventsOptionalParameters) (<-chan datadog.PaginationResult[CIAppPipelineEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, SearchCIAppPipelineEventsOptionalParameters{})
	}
	if o[0].Body == nil {
		o[0].Body = NewCIAppPipelineEventsRequest()
	}
	if o[0].Body.Page == nil {
		o[0].Body.Page = NewCIAppQueryPageOptions()
	}
	if o[0].Body.Page.Limit != nil {
		pageSize_ = *o[0].Body.Page.Limit
	}
	o[0].Body.Page.Limit = &pageSize_

	items := make(chan datadog.PaginationResult[CIAppPipelineEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildSearchCIAppPipelineEventsRequest(ctx, o...)
			if err != nil {
				var returnItem CIAppPipelineEvent
				items <- datadog.PaginationResult[CIAppPipelineEvent]{returnItem, err}
				break
			}

			resp, _, err := a.searchCIAppPipelineEventsExecute(req)
			if err != nil {
				var returnItem CIAppPipelineEvent
				items <- datadog.PaginationResult[CIAppPipelineEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[CIAppPipelineEvent]{item, nil}:
				case <-ctx.Done():
					close(items)
					return
				}
			}
			if len(results) < int(pageSize_) {
				break
			}
			cursorMeta, ok := resp.GetMetaOk()
			if !ok {
				break
			}
			cursorMetaPage, ok := cursorMeta.GetPageOk()
			if !ok {
				break
			}
			cursorMetaPageAfter, ok := cursorMetaPage.GetAfterOk()
			if !ok {
				break
			}

			o[0].Body.Page.Cursor = cursorMetaPageAfter
		}
		close(items)
	}()
	return items, cancel
}

// searchCIAppPipelineEventsExecute executes the request.
func (a *CIVisibilityPipelinesApi) searchCIAppPipelineEventsExecute(r apiSearchCIAppPipelineEventsRequest) (CIAppPipelineEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue CIAppPipelineEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CIVisibilityPipelinesApi.SearchCIAppPipelineEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/ci/pipelines/events/search"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	// body params
	localVarPostBody = r.body
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.Client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := datadog.ReadBody(localVarHTTPResponse)
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.Client.Decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

// NewCIVisibilityPipelinesApi Returns NewCIVisibilityPipelinesApi.
func NewCIVisibilityPipelinesApi(client *datadog.APIClient) *CIVisibilityPipelinesApi {
	return &CIVisibilityPipelinesApi{
		Client: client,
	}
}
