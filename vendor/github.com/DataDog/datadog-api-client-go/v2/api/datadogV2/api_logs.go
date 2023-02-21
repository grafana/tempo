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

// LogsApi service type
type LogsApi datadog.Service

type apiAggregateLogsRequest struct {
	ctx  _context.Context
	body *LogsAggregateRequest
}

func (a *LogsApi) buildAggregateLogsRequest(ctx _context.Context, body LogsAggregateRequest) (apiAggregateLogsRequest, error) {
	req := apiAggregateLogsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// AggregateLogs Aggregate events.
// The API endpoint to aggregate events into buckets and compute metrics and timeseries.
func (a *LogsApi) AggregateLogs(ctx _context.Context, body LogsAggregateRequest) (LogsAggregateResponse, *_nethttp.Response, error) {
	req, err := a.buildAggregateLogsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue LogsAggregateResponse
		return localVarReturnValue, nil, err
	}

	return a.aggregateLogsExecute(req)
}

// aggregateLogsExecute executes the request.
func (a *LogsApi) aggregateLogsExecute(r apiAggregateLogsRequest) (LogsAggregateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsAggregateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsApi.AggregateLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/analytics/aggregate"

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

type apiListLogsRequest struct {
	ctx  _context.Context
	body *LogsListRequest
}

// ListLogsOptionalParameters holds optional parameters for ListLogs.
type ListLogsOptionalParameters struct {
	Body *LogsListRequest
}

// NewListLogsOptionalParameters creates an empty struct for parameters.
func NewListLogsOptionalParameters() *ListLogsOptionalParameters {
	this := ListLogsOptionalParameters{}
	return &this
}

// WithBody sets the corresponding parameter name and returns the struct.
func (r *ListLogsOptionalParameters) WithBody(body LogsListRequest) *ListLogsOptionalParameters {
	r.Body = &body
	return r
}

func (a *LogsApi) buildListLogsRequest(ctx _context.Context, o ...ListLogsOptionalParameters) (apiListLogsRequest, error) {
	req := apiListLogsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListLogsOptionalParameters is allowed")
	}

	if o != nil {
		req.body = o[0].Body
	}
	return req, nil
}

// ListLogs Search logs.
// List endpoint returns logs that match a log search query.
// [Results are paginated][1].
//
// Use this endpoint to build complex logs filtering and search.
//
// **If you are considering archiving logs for your organization,
// consider use of the Datadog archive capabilities instead of the log list API.
// See [Datadog Logs Archive documentation][2].**
//
// [1]: /logs/guide/collect-multiple-logs-with-pagination
// [2]: https://docs.datadoghq.com/logs/archives
func (a *LogsApi) ListLogs(ctx _context.Context, o ...ListLogsOptionalParameters) (LogsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListLogsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue LogsListResponse
		return localVarReturnValue, nil, err
	}

	return a.listLogsExecute(req)
}

// ListLogsWithPagination provides a paginated version of ListLogs returning a channel with all items.
func (a *LogsApi) ListLogsWithPagination(ctx _context.Context, o ...ListLogsOptionalParameters) (<-chan datadog.PaginationResult[Log], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListLogsOptionalParameters{})
	}
	if o[0].Body == nil {
		o[0].Body = NewLogsListRequest()
	}
	if o[0].Body.Page == nil {
		o[0].Body.Page = NewLogsListRequestPage()
	}
	if o[0].Body.Page.Limit != nil {
		pageSize_ = *o[0].Body.Page.Limit
	}
	o[0].Body.Page.Limit = &pageSize_

	items := make(chan datadog.PaginationResult[Log], pageSize_)
	go func() {
		for {
			req, err := a.buildListLogsRequest(ctx, o...)
			if err != nil {
				var returnItem Log
				items <- datadog.PaginationResult[Log]{returnItem, err}
				break
			}

			resp, _, err := a.listLogsExecute(req)
			if err != nil {
				var returnItem Log
				items <- datadog.PaginationResult[Log]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[Log]{item, nil}:
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

// listLogsExecute executes the request.
func (a *LogsApi) listLogsExecute(r apiListLogsRequest) (LogsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsApi.ListLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/events/search"

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

type apiListLogsGetRequest struct {
	ctx               _context.Context
	filterQuery       *string
	filterIndex       *string
	filterFrom        *time.Time
	filterTo          *time.Time
	filterStorageTier *LogsStorageTier
	sort              *LogsSort
	pageCursor        *string
	pageLimit         *int32
}

// ListLogsGetOptionalParameters holds optional parameters for ListLogsGet.
type ListLogsGetOptionalParameters struct {
	FilterQuery       *string
	FilterIndex       *string
	FilterFrom        *time.Time
	FilterTo          *time.Time
	FilterStorageTier *LogsStorageTier
	Sort              *LogsSort
	PageCursor        *string
	PageLimit         *int32
}

// NewListLogsGetOptionalParameters creates an empty struct for parameters.
func NewListLogsGetOptionalParameters() *ListLogsGetOptionalParameters {
	this := ListLogsGetOptionalParameters{}
	return &this
}

// WithFilterQuery sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithFilterQuery(filterQuery string) *ListLogsGetOptionalParameters {
	r.FilterQuery = &filterQuery
	return r
}

// WithFilterIndex sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithFilterIndex(filterIndex string) *ListLogsGetOptionalParameters {
	r.FilterIndex = &filterIndex
	return r
}

// WithFilterFrom sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithFilterFrom(filterFrom time.Time) *ListLogsGetOptionalParameters {
	r.FilterFrom = &filterFrom
	return r
}

// WithFilterTo sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithFilterTo(filterTo time.Time) *ListLogsGetOptionalParameters {
	r.FilterTo = &filterTo
	return r
}

// WithFilterStorageTier sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithFilterStorageTier(filterStorageTier LogsStorageTier) *ListLogsGetOptionalParameters {
	r.FilterStorageTier = &filterStorageTier
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithSort(sort LogsSort) *ListLogsGetOptionalParameters {
	r.Sort = &sort
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithPageCursor(pageCursor string) *ListLogsGetOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListLogsGetOptionalParameters) WithPageLimit(pageLimit int32) *ListLogsGetOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

func (a *LogsApi) buildListLogsGetRequest(ctx _context.Context, o ...ListLogsGetOptionalParameters) (apiListLogsGetRequest, error) {
	req := apiListLogsGetRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListLogsGetOptionalParameters is allowed")
	}

	if o != nil {
		req.filterQuery = o[0].FilterQuery
		req.filterIndex = o[0].FilterIndex
		req.filterFrom = o[0].FilterFrom
		req.filterTo = o[0].FilterTo
		req.filterStorageTier = o[0].FilterStorageTier
		req.sort = o[0].Sort
		req.pageCursor = o[0].PageCursor
		req.pageLimit = o[0].PageLimit
	}
	return req, nil
}

// ListLogsGet Get a list of logs.
// List endpoint returns logs that match a log search query.
// [Results are paginated][1].
//
// Use this endpoint to see your latest logs.
//
// **If you are considering archiving logs for your organization,
// consider use of the Datadog archive capabilities instead of the log list API.
// See [Datadog Logs Archive documentation][2].**
//
// [1]: /logs/guide/collect-multiple-logs-with-pagination
// [2]: https://docs.datadoghq.com/logs/archives
func (a *LogsApi) ListLogsGet(ctx _context.Context, o ...ListLogsGetOptionalParameters) (LogsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListLogsGetRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue LogsListResponse
		return localVarReturnValue, nil, err
	}

	return a.listLogsGetExecute(req)
}

// ListLogsGetWithPagination provides a paginated version of ListLogsGet returning a channel with all items.
func (a *LogsApi) ListLogsGetWithPagination(ctx _context.Context, o ...ListLogsGetOptionalParameters) (<-chan datadog.PaginationResult[Log], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListLogsGetOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[Log], pageSize_)
	go func() {
		for {
			req, err := a.buildListLogsGetRequest(ctx, o...)
			if err != nil {
				var returnItem Log
				items <- datadog.PaginationResult[Log]{returnItem, err}
				break
			}

			resp, _, err := a.listLogsGetExecute(req)
			if err != nil {
				var returnItem Log
				items <- datadog.PaginationResult[Log]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[Log]{item, nil}:
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

// listLogsGetExecute executes the request.
func (a *LogsApi) listLogsGetExecute(r apiListLogsGetRequest) (LogsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsApi.ListLogsGet")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/events"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filterQuery != nil {
		localVarQueryParams.Add("filter[query]", datadog.ParameterToString(*r.filterQuery, ""))
	}
	if r.filterIndex != nil {
		localVarQueryParams.Add("filter[index]", datadog.ParameterToString(*r.filterIndex, ""))
	}
	if r.filterFrom != nil {
		localVarQueryParams.Add("filter[from]", datadog.ParameterToString(*r.filterFrom, ""))
	}
	if r.filterTo != nil {
		localVarQueryParams.Add("filter[to]", datadog.ParameterToString(*r.filterTo, ""))
	}
	if r.filterStorageTier != nil {
		localVarQueryParams.Add("filter[storage_tier]", datadog.ParameterToString(*r.filterStorageTier, ""))
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

type apiSubmitLogRequest struct {
	ctx             _context.Context
	body            *[]HTTPLogItem
	contentEncoding *ContentEncoding
	ddtags          *string
}

// SubmitLogOptionalParameters holds optional parameters for SubmitLog.
type SubmitLogOptionalParameters struct {
	ContentEncoding *ContentEncoding
	Ddtags          *string
}

// NewSubmitLogOptionalParameters creates an empty struct for parameters.
func NewSubmitLogOptionalParameters() *SubmitLogOptionalParameters {
	this := SubmitLogOptionalParameters{}
	return &this
}

// WithContentEncoding sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithContentEncoding(contentEncoding ContentEncoding) *SubmitLogOptionalParameters {
	r.ContentEncoding = &contentEncoding
	return r
}

// WithDdtags sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithDdtags(ddtags string) *SubmitLogOptionalParameters {
	r.Ddtags = &ddtags
	return r
}

func (a *LogsApi) buildSubmitLogRequest(ctx _context.Context, body []HTTPLogItem, o ...SubmitLogOptionalParameters) (apiSubmitLogRequest, error) {
	req := apiSubmitLogRequest{
		ctx:  ctx,
		body: &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SubmitLogOptionalParameters is allowed")
	}

	if o != nil {
		req.contentEncoding = o[0].ContentEncoding
		req.ddtags = o[0].Ddtags
	}
	return req, nil
}

// SubmitLog Send logs.
// Send your logs to your Datadog platform over HTTP. Limits per HTTP request are:
//
// - Maximum content size per payload (uncompressed): 5MB
// - Maximum size for a single log: 1MB
// - Maximum array size if sending multiple logs in an array: 1000 entries
//
// Any log exceeding 1MB is accepted and truncated by Datadog:
// - For a single log request, the API truncates the log at 1MB and returns a 2xx.
// - For a multi-logs request, the API processes all logs, truncates only logs larger than 1MB, and returns a 2xx.
//
// Datadog recommends sending your logs compressed.
// Add the `Content-Encoding: gzip` header to the request when sending compressed logs.
//
// The status codes answered by the HTTP API are:
// - 202: Accepted: the request has been accepted for processing
// - 400: Bad request (likely an issue in the payload formatting)
// - 401: Unauthorized (likely a missing API Key)
// - 403: Permission issue (likely using an invalid API Key)
// - 408: Request Timeout, request should be retried after some time
// - 413: Payload too large (batch is above 5MB uncompressed)
// - 429: Too Many Requests, request should be retried after some time
// - 500: Internal Server Error, the server encountered an unexpected condition that prevented it from fulfilling the request, request should be retried after some time
// - 503: Service Unavailable, the server is not ready to handle the request probably because it is overloaded, request should be retried after some time
func (a *LogsApi) SubmitLog(ctx _context.Context, body []HTTPLogItem, o ...SubmitLogOptionalParameters) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildSubmitLogRequest(ctx, body, o...)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.submitLogExecute(req)
}

// submitLogExecute executes the request.
func (a *LogsApi) submitLogExecute(r apiSubmitLogRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsApi.SubmitLog")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.ddtags != nil {
		localVarQueryParams.Add("ddtags", datadog.ParameterToString(*r.ddtags, ""))
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	if r.contentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = datadog.ParameterToString(*r.contentEncoding, "")
	}

	// body params
	localVarPostBody = r.body
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 408 || localVarHTTPResponse.StatusCode == 413 || localVarHTTPResponse.StatusCode == 429 || localVarHTTPResponse.StatusCode == 500 || localVarHTTPResponse.StatusCode == 503 {
			var v HTTPLogErrors
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

// NewLogsApi Returns NewLogsApi.
func NewLogsApi(client *datadog.APIClient) *LogsApi {
	return &LogsApi{
		Client: client,
	}
}
