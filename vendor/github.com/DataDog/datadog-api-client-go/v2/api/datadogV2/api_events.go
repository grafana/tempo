// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_fmt "fmt"
	_log "log"
	_nethttp "net/http"
	_neturl "net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// EventsApi service type
type EventsApi datadog.Service

type apiListEventsRequest struct {
	ctx         _context.Context
	filterQuery *string
	filterFrom  *string
	filterTo    *string
	sort        *EventsSort
	pageCursor  *string
	pageLimit   *int32
}

// ListEventsOptionalParameters holds optional parameters for ListEvents.
type ListEventsOptionalParameters struct {
	FilterQuery *string
	FilterFrom  *string
	FilterTo    *string
	Sort        *EventsSort
	PageCursor  *string
	PageLimit   *int32
}

// NewListEventsOptionalParameters creates an empty struct for parameters.
func NewListEventsOptionalParameters() *ListEventsOptionalParameters {
	this := ListEventsOptionalParameters{}
	return &this
}

// WithFilterQuery sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithFilterQuery(filterQuery string) *ListEventsOptionalParameters {
	r.FilterQuery = &filterQuery
	return r
}

// WithFilterFrom sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithFilterFrom(filterFrom string) *ListEventsOptionalParameters {
	r.FilterFrom = &filterFrom
	return r
}

// WithFilterTo sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithFilterTo(filterTo string) *ListEventsOptionalParameters {
	r.FilterTo = &filterTo
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithSort(sort EventsSort) *ListEventsOptionalParameters {
	r.Sort = &sort
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPageCursor(pageCursor string) *ListEventsOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPageLimit(pageLimit int32) *ListEventsOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

func (a *EventsApi) buildListEventsRequest(ctx _context.Context, o ...ListEventsOptionalParameters) (apiListEventsRequest, error) {
	req := apiListEventsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListEventsOptionalParameters is allowed")
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

// ListEvents Get a list of events.
// List endpoint returns events that match an events search query.
// [Results are paginated similarly to logs](https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination).
//
// Use this endpoint to see your latest events.
func (a *EventsApi) ListEvents(ctx _context.Context, o ...ListEventsOptionalParameters) (EventsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListEventsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue EventsListResponse
		return localVarReturnValue, nil, err
	}

	return a.listEventsExecute(req)
}

// ListEventsWithPagination provides a paginated version of ListEvents returning a channel with all items.
func (a *EventsApi) ListEventsWithPagination(ctx _context.Context, o ...ListEventsOptionalParameters) (<-chan datadog.PaginationResult[EventResponse], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, ListEventsOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[EventResponse], pageSize_)
	go func() {
		for {
			req, err := a.buildListEventsRequest(ctx, o...)
			if err != nil {
				var returnItem EventResponse
				items <- datadog.PaginationResult[EventResponse]{returnItem, err}
				break
			}

			resp, _, err := a.listEventsExecute(req)
			if err != nil {
				var returnItem EventResponse
				items <- datadog.PaginationResult[EventResponse]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[EventResponse]{item, nil}:
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

// listEventsExecute executes the request.
func (a *EventsApi) listEventsExecute(r apiListEventsRequest) (EventsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventsListResponse
	)

	operationId := "v2.ListEvents"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.EventsApi.ListEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/events"

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

type apiSearchEventsRequest struct {
	ctx  _context.Context
	body *EventsListRequest
}

// SearchEventsOptionalParameters holds optional parameters for SearchEvents.
type SearchEventsOptionalParameters struct {
	Body *EventsListRequest
}

// NewSearchEventsOptionalParameters creates an empty struct for parameters.
func NewSearchEventsOptionalParameters() *SearchEventsOptionalParameters {
	this := SearchEventsOptionalParameters{}
	return &this
}

// WithBody sets the corresponding parameter name and returns the struct.
func (r *SearchEventsOptionalParameters) WithBody(body EventsListRequest) *SearchEventsOptionalParameters {
	r.Body = &body
	return r
}

func (a *EventsApi) buildSearchEventsRequest(ctx _context.Context, o ...SearchEventsOptionalParameters) (apiSearchEventsRequest, error) {
	req := apiSearchEventsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SearchEventsOptionalParameters is allowed")
	}

	if o != nil {
		req.body = o[0].Body
	}
	return req, nil
}

// SearchEvents Search events.
// List endpoint returns events that match an events search query.
// [Results are paginated similarly to logs](https://docs.datadoghq.com/logs/guide/collect-multiple-logs-with-pagination).
//
// Use this endpoint to build complex events filtering and search.
func (a *EventsApi) SearchEvents(ctx _context.Context, o ...SearchEventsOptionalParameters) (EventsListResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchEventsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue EventsListResponse
		return localVarReturnValue, nil, err
	}

	return a.searchEventsExecute(req)
}

// SearchEventsWithPagination provides a paginated version of SearchEvents returning a channel with all items.
func (a *EventsApi) SearchEventsWithPagination(ctx _context.Context, o ...SearchEventsOptionalParameters) (<-chan datadog.PaginationResult[EventResponse], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(10)
	if len(o) == 0 {
		o = append(o, SearchEventsOptionalParameters{})
	}
	if o[0].Body == nil {
		o[0].Body = NewEventsListRequest()
	}
	if o[0].Body.Page == nil {
		o[0].Body.Page = NewEventsRequestPage()
	}
	if o[0].Body.Page.Limit != nil {
		pageSize_ = *o[0].Body.Page.Limit
	}
	o[0].Body.Page.Limit = &pageSize_

	items := make(chan datadog.PaginationResult[EventResponse], pageSize_)
	go func() {
		for {
			req, err := a.buildSearchEventsRequest(ctx, o...)
			if err != nil {
				var returnItem EventResponse
				items <- datadog.PaginationResult[EventResponse]{returnItem, err}
				break
			}

			resp, _, err := a.searchEventsExecute(req)
			if err != nil {
				var returnItem EventResponse
				items <- datadog.PaginationResult[EventResponse]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[EventResponse]{item, nil}:
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

// searchEventsExecute executes the request.
func (a *EventsApi) searchEventsExecute(r apiSearchEventsRequest) (EventsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue EventsListResponse
	)

	operationId := "v2.SearchEvents"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.EventsApi.SearchEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/events/search"

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

// NewEventsApi Returns NewEventsApi.
func NewEventsApi(client *datadog.APIClient) *EventsApi {
	return &EventsApi{
		Client: client,
	}
}
