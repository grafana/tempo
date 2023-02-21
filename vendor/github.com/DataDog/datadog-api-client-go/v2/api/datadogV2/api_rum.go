// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// RUMApi service type
type RUMApi datadog.Service

type apiAggregateRUMEventsRequest struct {
	ctx  _context.Context
	body *RUMAggregateRequest
}

func (a *RUMApi) buildAggregateRUMEventsRequest(ctx _context.Context, body RUMAggregateRequest) (apiAggregateRUMEventsRequest, error) {
	req := apiAggregateRUMEventsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// AggregateRUMEvents Aggregate RUM events.
// The API endpoint to aggregate RUM events into buckets of computed metrics and timeseries.
func (a *RUMApi) AggregateRUMEvents(ctx _context.Context, body RUMAggregateRequest) (RUMAnalyticsAggregateResponse, *_nethttp.Response, error) {
	req, err := a.buildAggregateRUMEventsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue RUMAnalyticsAggregateResponse
		return localVarReturnValue, nil, err
	}

	return a.aggregateRUMEventsExecute(req)
}

// aggregateRUMEventsExecute executes the request.
func (a *RUMApi) aggregateRUMEventsExecute(r apiAggregateRUMEventsRequest) (RUMAnalyticsAggregateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue RUMAnalyticsAggregateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.AggregateRUMEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/analytics/aggregate"

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

type apiCreateRUMApplicationRequest struct {
	ctx  _context.Context
	body *RUMApplicationCreateRequest
}

func (a *RUMApi) buildCreateRUMApplicationRequest(ctx _context.Context, body RUMApplicationCreateRequest) (apiCreateRUMApplicationRequest, error) {
	req := apiCreateRUMApplicationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateRUMApplication Create a new RUM application.
// Create a new RUM application in your organization.
func (a *RUMApi) CreateRUMApplication(ctx _context.Context, body RUMApplicationCreateRequest) (RUMApplicationResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateRUMApplicationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue RUMApplicationResponse
		return localVarReturnValue, nil, err
	}

	return a.createRUMApplicationExecute(req)
}

// createRUMApplicationExecute executes the request.
func (a *RUMApi) createRUMApplicationExecute(r apiCreateRUMApplicationRequest) (RUMApplicationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue RUMApplicationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.CreateRUMApplication")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/applications"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 429 {
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

type apiDeleteRUMApplicationRequest struct {
	ctx _context.Context
	id  string
}

func (a *RUMApi) buildDeleteRUMApplicationRequest(ctx _context.Context, id string) (apiDeleteRUMApplicationRequest, error) {
	req := apiDeleteRUMApplicationRequest{
		ctx: ctx,
		id:  id,
	}
	return req, nil
}

// DeleteRUMApplication Delete a RUM application.
// Delete an existing RUM application in your organization.
func (a *RUMApi) DeleteRUMApplication(ctx _context.Context, id string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteRUMApplicationRequest(ctx, id)
	if err != nil {
		return nil, err
	}

	return a.deleteRUMApplicationExecute(req)
}

// deleteRUMApplicationExecute executes the request.
func (a *RUMApi) deleteRUMApplicationExecute(r apiDeleteRUMApplicationRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.DeleteRUMApplication")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/applications/{id}"
	localVarPath = strings.Replace(localVarPath, "{"+"id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.id, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Accept"] = "*/*"

	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
	if err != nil {
		return nil, err
	}

	localVarHTTPResponse, err := a.Client.CallAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarHTTPResponse, err
	}

	localVarBody, err := datadog.ReadBody(localVarHTTPResponse)
	if err != nil {
		return localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := datadog.GenericOpenAPIError{
			ErrorBody:    localVarBody,
			ErrorMessage: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
		}
		return localVarHTTPResponse, newErr
	}

	return localVarHTTPResponse, nil
}

type apiGetRUMApplicationRequest struct {
	ctx _context.Context
	id  string
}

func (a *RUMApi) buildGetRUMApplicationRequest(ctx _context.Context, id string) (apiGetRUMApplicationRequest, error) {
	req := apiGetRUMApplicationRequest{
		ctx: ctx,
		id:  id,
	}
	return req, nil
}

// GetRUMApplication Get a RUM application.
// Get the RUM application with given ID in your organization.
func (a *RUMApi) GetRUMApplication(ctx _context.Context, id string) (RUMApplicationResponse, *_nethttp.Response, error) {
	req, err := a.buildGetRUMApplicationRequest(ctx, id)
	if err != nil {
		var localVarReturnValue RUMApplicationResponse
		return localVarReturnValue, nil, err
	}

	return a.getRUMApplicationExecute(req)
}

// getRUMApplicationExecute executes the request.
func (a *RUMApi) getRUMApplicationExecute(r apiGetRUMApplicationRequest) (RUMApplicationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RUMApplicationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.GetRUMApplication")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/applications/{id}"
	localVarPath = strings.Replace(localVarPath, "{"+"id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.id, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiGetRUMApplicationsRequest struct {
	ctx _context.Context
}

func (a *RUMApi) buildGetRUMApplicationsRequest(ctx _context.Context) (apiGetRUMApplicationsRequest, error) {
	req := apiGetRUMApplicationsRequest{
		ctx: ctx,
	}
	return req, nil
}

// GetRUMApplications List all the RUM applications.
// List all the RUM applications in your organization.
func (a *RUMApi) GetRUMApplications(ctx _context.Context) (RUMApplicationsResponse, *_nethttp.Response, error) {
	req, err := a.buildGetRUMApplicationsRequest(ctx)
	if err != nil {
		var localVarReturnValue RUMApplicationsResponse
		return localVarReturnValue, nil, err
	}

	return a.getRUMApplicationsExecute(req)
}

// getRUMApplicationsExecute executes the request.
func (a *RUMApi) getRUMApplicationsExecute(r apiGetRUMApplicationsRequest) (RUMApplicationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RUMApplicationsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.GetRUMApplications")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/applications"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiListRUMEventsRequest struct {
	ctx         _context.Context
	filterQuery *string
	filterFrom  *time.Time
	filterTo    *time.Time
	sort        *RUMSort
	pageCursor  *string
	pageLimit   *int32
}

// ListRUMEventsOptionalParameters holds optional parameters for ListRUMEvents.
type ListRUMEventsOptionalParameters struct {
	FilterQuery *string
	FilterFrom  *time.Time
	FilterTo    *time.Time
	Sort        *RUMSort
	PageCursor  *string
	PageLimit   *int32
}

// NewListRUMEventsOptionalParameters creates an empty struct for parameters.
func NewListRUMEventsOptionalParameters() *ListRUMEventsOptionalParameters {
	this := ListRUMEventsOptionalParameters{}
	return &this
}

// WithFilterQuery sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithFilterQuery(filterQuery string) *ListRUMEventsOptionalParameters {
	r.FilterQuery = &filterQuery
	return r
}

// WithFilterFrom sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithFilterFrom(filterFrom time.Time) *ListRUMEventsOptionalParameters {
	r.FilterFrom = &filterFrom
	return r
}

// WithFilterTo sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithFilterTo(filterTo time.Time) *ListRUMEventsOptionalParameters {
	r.FilterTo = &filterTo
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithSort(sort RUMSort) *ListRUMEventsOptionalParameters {
	r.Sort = &sort
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithPageCursor(pageCursor string) *ListRUMEventsOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListRUMEventsOptionalParameters) WithPageLimit(pageLimit int32) *ListRUMEventsOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

func (a *RUMApi) buildListRUMEventsRequest(ctx _context.Context, o ...ListRUMEventsOptionalParameters) (apiListRUMEventsRequest, error) {
	req := apiListRUMEventsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListRUMEventsOptionalParameters is allowed")
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

// ListRUMEvents Get a list of RUM events.
// List endpoint returns events that match a RUM search query.
// [Results are paginated][1].
//
// Use this endpoint to see your latest RUM events.
//
// [1]: https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination
func (a *RUMApi) ListRUMEvents(ctx _context.Context, o ...ListRUMEventsOptionalParameters) (RUMEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildListRUMEventsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue RUMEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.listRUMEventsExecute(req)
}

// ListRUMEventsWithPagination provides a paginated version of ListRUMEvents returning a channel with all items.
func (a *RUMApi) ListRUMEventsWithPagination(ctx _context.Context, o ...ListRUMEventsOptionalParameters) (<-chan datadog.PaginationResult[RUMEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListRUMEventsOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[RUMEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildListRUMEventsRequest(ctx, o...)
			if err != nil {
				var returnItem RUMEvent
				items <- datadog.PaginationResult[RUMEvent]{returnItem, err}
				break
			}

			resp, _, err := a.listRUMEventsExecute(req)
			if err != nil {
				var returnItem RUMEvent
				items <- datadog.PaginationResult[RUMEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[RUMEvent]{item, nil}:
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

// listRUMEventsExecute executes the request.
func (a *RUMApi) listRUMEventsExecute(r apiListRUMEventsRequest) (RUMEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RUMEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.ListRUMEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/events"

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

type apiSearchRUMEventsRequest struct {
	ctx  _context.Context
	body *RUMSearchEventsRequest
}

func (a *RUMApi) buildSearchRUMEventsRequest(ctx _context.Context, body RUMSearchEventsRequest) (apiSearchRUMEventsRequest, error) {
	req := apiSearchRUMEventsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// SearchRUMEvents Search RUM events.
// List endpoint returns RUM events that match a RUM search query.
// [Results are paginated][1].
//
// Use this endpoint to build complex RUM events filtering and search.
//
// [1]: https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination
func (a *RUMApi) SearchRUMEvents(ctx _context.Context, body RUMSearchEventsRequest) (RUMEventsResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchRUMEventsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue RUMEventsResponse
		return localVarReturnValue, nil, err
	}

	return a.searchRUMEventsExecute(req)
}

// SearchRUMEventsWithPagination provides a paginated version of SearchRUMEvents returning a channel with all items.
func (a *RUMApi) SearchRUMEventsWithPagination(ctx _context.Context, body RUMSearchEventsRequest) (<-chan datadog.PaginationResult[RUMEvent], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if body.Page == nil {
		body.Page = NewRUMQueryPageOptions()
	}
	if body.Page.Limit == nil {
		// int32
		body.Page.Limit = &pageSize_
	} else {
		pageSize_ = *body.Page.Limit
	}

	items := make(chan datadog.PaginationResult[RUMEvent], pageSize_)
	go func() {
		for {
			req, err := a.buildSearchRUMEventsRequest(ctx, body)
			if err != nil {
				var returnItem RUMEvent
				items <- datadog.PaginationResult[RUMEvent]{returnItem, err}
				break
			}

			resp, _, err := a.searchRUMEventsExecute(req)
			if err != nil {
				var returnItem RUMEvent
				items <- datadog.PaginationResult[RUMEvent]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[RUMEvent]{item, nil}:
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

			body.Page.Cursor = cursorMetaPageAfter
		}
		close(items)
	}()
	return items, cancel
}

// searchRUMEventsExecute executes the request.
func (a *RUMApi) searchRUMEventsExecute(r apiSearchRUMEventsRequest) (RUMEventsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue RUMEventsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.SearchRUMEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/events/search"

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

type apiUpdateRUMApplicationRequest struct {
	ctx  _context.Context
	id   string
	body *RUMApplicationUpdateRequest
}

func (a *RUMApi) buildUpdateRUMApplicationRequest(ctx _context.Context, id string, body RUMApplicationUpdateRequest) (apiUpdateRUMApplicationRequest, error) {
	req := apiUpdateRUMApplicationRequest{
		ctx:  ctx,
		id:   id,
		body: &body,
	}
	return req, nil
}

// UpdateRUMApplication Update a RUM application.
// Update the RUM application with given ID in your organization.
func (a *RUMApi) UpdateRUMApplication(ctx _context.Context, id string, body RUMApplicationUpdateRequest) (RUMApplicationResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateRUMApplicationRequest(ctx, id, body)
	if err != nil {
		var localVarReturnValue RUMApplicationResponse
		return localVarReturnValue, nil, err
	}

	return a.updateRUMApplicationExecute(req)
}

// updateRUMApplicationExecute executes the request.
func (a *RUMApi) updateRUMApplicationExecute(r apiUpdateRUMApplicationRequest) (RUMApplicationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue RUMApplicationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RUMApi.UpdateRUMApplication")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/rum/applications/{id}"
	localVarPath = strings.Replace(localVarPath, "{"+"id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.id, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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

// NewRUMApi Returns NewRUMApi.
func NewRUMApi(client *datadog.APIClient) *RUMApi {
	return &RUMApi{
		Client: client,
	}
}
