// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// EventsApi service type
type EventsApi datadog.Service

type apiCreateEventRequest struct {
	ctx  _context.Context
	body *EventCreateRequest
}

func (a *EventsApi) buildCreateEventRequest(ctx _context.Context, body EventCreateRequest) (apiCreateEventRequest, error) {
	req := apiCreateEventRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateEvent Post an event.
// This endpoint allows you to post events to the stream.
// Tag them, set priority and event aggregate them with other events.
func (a *EventsApi) CreateEvent(ctx _context.Context, body EventCreateRequest) (EventCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateEventRequest(ctx, body)
	if err != nil {
		var localVarReturnValue EventCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createEventExecute(req)
}

// createEventExecute executes the request.
func (a *EventsApi) createEventExecute(r apiCreateEventRequest) (EventCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue EventCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.EventsApi.CreateEvent")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

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

type apiGetEventRequest struct {
	ctx     _context.Context
	eventId int64
}

func (a *EventsApi) buildGetEventRequest(ctx _context.Context, eventId int64) (apiGetEventRequest, error) {
	req := apiGetEventRequest{
		ctx:     ctx,
		eventId: eventId,
	}
	return req, nil
}

// GetEvent Get an event.
// This endpoint allows you to query for event details.
//
// **Note**: If the event you’re querying contains markdown formatting of any kind,
// you may see characters such as `%`,`\`,`n` in your output.
func (a *EventsApi) GetEvent(ctx _context.Context, eventId int64) (EventResponse, *_nethttp.Response, error) {
	req, err := a.buildGetEventRequest(ctx, eventId)
	if err != nil {
		var localVarReturnValue EventResponse
		return localVarReturnValue, nil, err
	}

	return a.getEventExecute(req)
}

// getEventExecute executes the request.
func (a *EventsApi) getEventExecute(r apiGetEventRequest) (EventResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.EventsApi.GetEvent")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events/{event_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"event_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.eventId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiListEventsRequest struct {
	ctx              _context.Context
	start            *int64
	end              *int64
	priority         *EventPriority
	sources          *string
	tags             *string
	unaggregated     *bool
	excludeAggregate *bool
	page             *int32
}

// ListEventsOptionalParameters holds optional parameters for ListEvents.
type ListEventsOptionalParameters struct {
	Priority         *EventPriority
	Sources          *string
	Tags             *string
	Unaggregated     *bool
	ExcludeAggregate *bool
	Page             *int32
}

// NewListEventsOptionalParameters creates an empty struct for parameters.
func NewListEventsOptionalParameters() *ListEventsOptionalParameters {
	this := ListEventsOptionalParameters{}
	return &this
}

// WithPriority sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPriority(priority EventPriority) *ListEventsOptionalParameters {
	r.Priority = &priority
	return r
}

// WithSources sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithSources(sources string) *ListEventsOptionalParameters {
	r.Sources = &sources
	return r
}

// WithTags sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithTags(tags string) *ListEventsOptionalParameters {
	r.Tags = &tags
	return r
}

// WithUnaggregated sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithUnaggregated(unaggregated bool) *ListEventsOptionalParameters {
	r.Unaggregated = &unaggregated
	return r
}

// WithExcludeAggregate sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithExcludeAggregate(excludeAggregate bool) *ListEventsOptionalParameters {
	r.ExcludeAggregate = &excludeAggregate
	return r
}

// WithPage sets the corresponding parameter name and returns the struct.
func (r *ListEventsOptionalParameters) WithPage(page int32) *ListEventsOptionalParameters {
	r.Page = &page
	return r
}

func (a *EventsApi) buildListEventsRequest(ctx _context.Context, start int64, end int64, o ...ListEventsOptionalParameters) (apiListEventsRequest, error) {
	req := apiListEventsRequest{
		ctx:   ctx,
		start: &start,
		end:   &end,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListEventsOptionalParameters is allowed")
	}

	if o != nil {
		req.priority = o[0].Priority
		req.sources = o[0].Sources
		req.tags = o[0].Tags
		req.unaggregated = o[0].Unaggregated
		req.excludeAggregate = o[0].ExcludeAggregate
		req.page = o[0].Page
	}
	return req, nil
}

// ListEvents Get a list of events.
// The event stream can be queried and filtered by time, priority, sources and tags.
//
// **Notes**:
// - If the event you’re querying contains markdown formatting of any kind,
// you may see characters such as `%`,`\`,`n` in your output.
//
// - This endpoint returns a maximum of `1000` most recent results. To return additional results,
// identify the last timestamp of the last result and set that as the `end` query time to
// paginate the results. You can also use the page parameter to specify which set of `1000` results to return.
func (a *EventsApi) ListEvents(ctx _context.Context, start int64, end int64, o ...ListEventsOptionalParameters) (EventListResponse, *_nethttp.Response, error) {
	req, err := a.buildListEventsRequest(ctx, start, end, o...)
	if err != nil {
		var localVarReturnValue EventListResponse
		return localVarReturnValue, nil, err
	}

	return a.listEventsExecute(req)
}

// listEventsExecute executes the request.
func (a *EventsApi) listEventsExecute(r apiListEventsRequest) (EventListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.EventsApi.ListEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.start == nil {
		return localVarReturnValue, nil, datadog.ReportError("start is required and must be specified")
	}
	if r.end == nil {
		return localVarReturnValue, nil, datadog.ReportError("end is required and must be specified")
	}
	localVarQueryParams.Add("start", datadog.ParameterToString(*r.start, ""))
	localVarQueryParams.Add("end", datadog.ParameterToString(*r.end, ""))
	if r.priority != nil {
		localVarQueryParams.Add("priority", datadog.ParameterToString(*r.priority, ""))
	}
	if r.sources != nil {
		localVarQueryParams.Add("sources", datadog.ParameterToString(*r.sources, ""))
	}
	if r.tags != nil {
		localVarQueryParams.Add("tags", datadog.ParameterToString(*r.tags, ""))
	}
	if r.unaggregated != nil {
		localVarQueryParams.Add("unaggregated", datadog.ParameterToString(*r.unaggregated, ""))
	}
	if r.excludeAggregate != nil {
		localVarQueryParams.Add("exclude_aggregate", datadog.ParameterToString(*r.excludeAggregate, ""))
	}
	if r.page != nil {
		localVarQueryParams.Add("page", datadog.ParameterToString(*r.page, ""))
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

// NewEventsApi Returns NewEventsApi.
func NewEventsApi(client *datadog.APIClient) *EventsApi {
	return &EventsApi{
		Client: client,
	}
}
