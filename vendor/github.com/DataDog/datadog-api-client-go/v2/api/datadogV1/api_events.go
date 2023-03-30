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

// CreateEvent Post an event.
// This endpoint allows you to post events to the stream.
// Tag them, set priority and event aggregate them with other events.
func (a *EventsApi) CreateEvent(ctx _context.Context, body EventCreateRequest) (EventCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue EventCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.EventsApi.CreateEvent")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

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

// GetEvent Get an event.
// This endpoint allows you to query for event details.
//
// **Note**: If the event you’re querying contains markdown formatting of any kind,
// you may see characters such as `%`,`\`,`n` in your output.
func (a *EventsApi) GetEvent(ctx _context.Context, eventId int64) (EventResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.EventsApi.GetEvent")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events/{event_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"event_id"+"}", _neturl.PathEscape(datadog.ParameterToString(eventId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
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
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue EventListResponse
		optionalParams      ListEventsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListEventsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.EventsApi.ListEvents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/events"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start", datadog.ParameterToString(start, ""))
	localVarQueryParams.Add("end", datadog.ParameterToString(end, ""))
	if optionalParams.Priority != nil {
		localVarQueryParams.Add("priority", datadog.ParameterToString(*optionalParams.Priority, ""))
	}
	if optionalParams.Sources != nil {
		localVarQueryParams.Add("sources", datadog.ParameterToString(*optionalParams.Sources, ""))
	}
	if optionalParams.Tags != nil {
		localVarQueryParams.Add("tags", datadog.ParameterToString(*optionalParams.Tags, ""))
	}
	if optionalParams.Unaggregated != nil {
		localVarQueryParams.Add("unaggregated", datadog.ParameterToString(*optionalParams.Unaggregated, ""))
	}
	if optionalParams.ExcludeAggregate != nil {
		localVarQueryParams.Add("exclude_aggregate", datadog.ParameterToString(*optionalParams.ExcludeAggregate, ""))
	}
	if optionalParams.Page != nil {
		localVarQueryParams.Add("page", datadog.ParameterToString(*optionalParams.Page, ""))
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

// NewEventsApi Returns NewEventsApi.
func NewEventsApi(client *datadog.APIClient) *EventsApi {
	return &EventsApi{
		Client: client,
	}
}
