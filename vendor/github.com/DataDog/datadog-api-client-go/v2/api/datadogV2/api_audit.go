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

// AuditApi service type
type AuditApi datadog.Service

type apiListAuditLogsRequest struct {
	ctx         _context.Context
	filterQuery *string
	filterFrom  *time.Time
	filterTo    *time.Time
	sort        *AuditLogsSort
	pageCursor  *string
	pageLimit   *int32
}

// ListAuditLogsOptionalParameters holds optional parameters for ListAuditLogs.
type ListAuditLogsOptionalParameters struct {
	FilterQuery *string
	FilterFrom  *time.Time
	FilterTo    *time.Time
	Sort        *AuditLogsSort
	PageCursor  *string
	PageLimit   *int32
}

// NewListAuditLogsOptionalParameters creates an empty struct for parameters.
func NewListAuditLogsOptionalParameters() *ListAuditLogsOptionalParameters {
	this := ListAuditLogsOptionalParameters{}
	return &this
}

// WithFilterQuery sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithFilterQuery(filterQuery string) *ListAuditLogsOptionalParameters {
	r.FilterQuery = &filterQuery
	return r
}

// WithFilterFrom sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithFilterFrom(filterFrom time.Time) *ListAuditLogsOptionalParameters {
	r.FilterFrom = &filterFrom
	return r
}

// WithFilterTo sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithFilterTo(filterTo time.Time) *ListAuditLogsOptionalParameters {
	r.FilterTo = &filterTo
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithSort(sort AuditLogsSort) *ListAuditLogsOptionalParameters {
	r.Sort = &sort
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithPageCursor(pageCursor string) *ListAuditLogsOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListAuditLogsOptionalParameters) WithPageLimit(pageLimit int32) *ListAuditLogsOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

func (a *AuditApi) buildListAuditLogsRequest(ctx _context.Context, o ...ListAuditLogsOptionalParameters) (apiListAuditLogsRequest, error) {
	req := apiListAuditLogsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListAuditLogsOptionalParameters is allowed")
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

// ListAuditLogs Get a list of Audit Logs events.
// List endpoint returns events that match a Audit Logs search query.
// [Results are paginated][1].
//
// Use this endpoint to see your latest Audit Logs events.
//
// [1]: https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination
func (a *AuditApi) ListAuditLogs(ctx _context.Context, o ...ListAuditLogsOptionalParameters) (AuditLogsEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildListAuditLogsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue AuditLogsEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.listAuditLogsExecute(req)
}

// ListAuditLogsWithPagination provides a paginated version of ListAuditLogs returning a channel with all items.
func (a *AuditApi) ListAuditLogsWithPagination(ctx _context.Context, o ...ListAuditLogsOptionalParameters) (<-chan datadog.PaginationResult[AuditLogsEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListAuditLogsOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[AuditLogsEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildListAuditLogsRequest(ctx, o...)
			if err != nil {
				var returnItem AuditLogsEvent
				items <- datadog.PaginationResult[AuditLogsEvent]{returnItem, err}
				break
			}

			resp, _, err := a.listAuditLogsExecute(req)
			if err != nil {
				var returnItem AuditLogsEvent
				items <- datadog.PaginationResult[AuditLogsEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[AuditLogsEvent]{item, nil}:
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

// listAuditLogsExecute executes the request.
func (a *AuditApi) listAuditLogsExecute(r apiListAuditLogsRequest) (AuditLogsEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue AuditLogsEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuditApi.ListAuditLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/audit/events"

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

type apiSearchAuditLogsRequest struct {
	ctx  _context.Context
	body *AuditLogsSearchEventsRequest
}

// SearchAuditLogsOptionalParameters holds optional parameters for SearchAuditLogs.
type SearchAuditLogsOptionalParameters struct {
	Body *AuditLogsSearchEventsRequest
}

// NewSearchAuditLogsOptionalParameters creates an empty struct for parameters.
func NewSearchAuditLogsOptionalParameters() *SearchAuditLogsOptionalParameters {
	this := SearchAuditLogsOptionalParameters{}
	return &this
}

// WithBody sets the corresponding parameter name and returns the struct.
func (r *SearchAuditLogsOptionalParameters) WithBody(body AuditLogsSearchEventsRequest) *SearchAuditLogsOptionalParameters {
	r.Body = &body
	return r
}

func (a *AuditApi) buildSearchAuditLogsRequest(ctx _context.Context, o ...SearchAuditLogsOptionalParameters) (apiSearchAuditLogsRequest, error) {
	req := apiSearchAuditLogsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SearchAuditLogsOptionalParameters is allowed")
	}

	if o != nil {
		req.body = o[0].Body
	}
	return req, nil
}

// SearchAuditLogs Search Audit Logs events.
// List endpoint returns Audit Logs events that match an Audit search query.
// [Results are paginated][1].
//
// Use this endpoint to build complex Audit Logs events filtering and search.
//
// [1]: https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination
func (a *AuditApi) SearchAuditLogs(ctx _context.Context, o ...SearchAuditLogsOptionalParameters) (AuditLogsEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchAuditLogsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue AuditLogsEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.searchAuditLogsExecute(req)
}

// SearchAuditLogsWithPagination provides a paginated version of SearchAuditLogs returning a channel with all items.
func (a *AuditApi) SearchAuditLogsWithPagination(ctx _context.Context, o ...SearchAuditLogsOptionalParameters) (<-chan datadog.PaginationResult[AuditLogsEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, SearchAuditLogsOptionalParameters{})
	}
	if o[0].Body == nil {
		o[0].Body = NewAuditLogsSearchEventsRequest()
	}
	if o[0].Body.Page == nil {
		o[0].Body.Page = NewAuditLogsQueryPageOptions()
	}
	if o[0].Body.Page.Limit != nil {
		pageSize_ = *o[0].Body.Page.Limit
	}
	o[0].Body.Page.Limit = &pageSize_

	items := make(chan datadog.PaginationResult[AuditLogsEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildSearchAuditLogsRequest(ctx, o...)
			if err != nil {
				var returnItem AuditLogsEvent
				items <- datadog.PaginationResult[AuditLogsEvent]{returnItem, err}
				break
			}

			resp, _, err := a.searchAuditLogsExecute(req)
			if err != nil {
				var returnItem AuditLogsEvent
				items <- datadog.PaginationResult[AuditLogsEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[AuditLogsEvent]{item, nil}:
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

// searchAuditLogsExecute executes the request.
func (a *AuditApi) searchAuditLogsExecute(r apiSearchAuditLogsRequest) (AuditLogsEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue AuditLogsEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuditApi.SearchAuditLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/audit/events/search"

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

// NewAuditApi Returns NewAuditApi.
func NewAuditApi(client *datadog.APIClient) *AuditApi {
	return &AuditApi{
		Client: client,
	}
}
