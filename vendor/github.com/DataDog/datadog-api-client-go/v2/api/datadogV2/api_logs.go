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

// AggregateLogs Aggregate events.
// The API endpoint to aggregate events into buckets and compute metrics and timeseries.
func (a *LogsApi) AggregateLogs(ctx _context.Context, body LogsAggregateRequest) (LogsAggregateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsAggregateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.LogsApi.AggregateLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/analytics/aggregate"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	// body params
	localVarPostBody = &body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
		optionalParams      ListLogsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListLogsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.LogsApi.ListLogs")
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
	localVarPostBody = &optionalParams.Body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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
			resp, _, err := a.ListLogs(ctx, o...)
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
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
		optionalParams      ListLogsGetOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListLogsGetOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.LogsApi.ListLogsGet")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/events"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.FilterQuery != nil {
		localVarQueryParams.Add("filter[query]", datadog.ParameterToString(*optionalParams.FilterQuery, ""))
	}
	if optionalParams.FilterIndex != nil {
		localVarQueryParams.Add("filter[index]", datadog.ParameterToString(*optionalParams.FilterIndex, ""))
	}
	if optionalParams.FilterFrom != nil {
		localVarQueryParams.Add("filter[from]", datadog.ParameterToString(*optionalParams.FilterFrom, ""))
	}
	if optionalParams.FilterTo != nil {
		localVarQueryParams.Add("filter[to]", datadog.ParameterToString(*optionalParams.FilterTo, ""))
	}
	if optionalParams.FilterStorageTier != nil {
		localVarQueryParams.Add("filter[storage_tier]", datadog.ParameterToString(*optionalParams.FilterStorageTier, ""))
	}
	if optionalParams.Sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*optionalParams.Sort, ""))
	}
	if optionalParams.PageCursor != nil {
		localVarQueryParams.Add("page[cursor]", datadog.ParameterToString(*optionalParams.PageCursor, ""))
	}
	if optionalParams.PageLimit != nil {
		localVarQueryParams.Add("page[limit]", datadog.ParameterToString(*optionalParams.PageLimit, ""))
	}
	localVarHeaderParams["Accept"] = "application/json"

	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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
			resp, _, err := a.ListLogsGet(ctx, o...)
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
// Log events can be submitted up to 18 hours in the past and 2 hours in the future.
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
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
		optionalParams      SubmitLogOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type SubmitLogOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.LogsApi.SubmitLog")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Ddtags != nil {
		localVarQueryParams.Add("ddtags", datadog.ParameterToString(*optionalParams.Ddtags, ""))
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	if optionalParams.ContentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = datadog.ParameterToString(*optionalParams.ContentEncoding, "")
	}

	// body params
	localVarPostBody = &body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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
