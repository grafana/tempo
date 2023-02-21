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

// DowntimesApi service type
type DowntimesApi datadog.Service

type apiCancelDowntimeRequest struct {
	ctx        _context.Context
	downtimeId int64
}

func (a *DowntimesApi) buildCancelDowntimeRequest(ctx _context.Context, downtimeId int64) (apiCancelDowntimeRequest, error) {
	req := apiCancelDowntimeRequest{
		ctx:        ctx,
		downtimeId: downtimeId,
	}
	return req, nil
}

// CancelDowntime Cancel a downtime.
// Cancel a downtime.
func (a *DowntimesApi) CancelDowntime(ctx _context.Context, downtimeId int64) (*_nethttp.Response, error) {
	req, err := a.buildCancelDowntimeRequest(ctx, downtimeId)
	if err != nil {
		return nil, err
	}

	return a.cancelDowntimeExecute(req)
}

// cancelDowntimeExecute executes the request.
func (a *DowntimesApi) cancelDowntimeExecute(r apiCancelDowntimeRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.CancelDowntime")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime/{downtime_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"downtime_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.downtimeId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiCancelDowntimesByScopeRequest struct {
	ctx  _context.Context
	body *CancelDowntimesByScopeRequest
}

func (a *DowntimesApi) buildCancelDowntimesByScopeRequest(ctx _context.Context, body CancelDowntimesByScopeRequest) (apiCancelDowntimesByScopeRequest, error) {
	req := apiCancelDowntimesByScopeRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CancelDowntimesByScope Cancel downtimes by scope.
// Delete all downtimes that match the scope of `X`.
func (a *DowntimesApi) CancelDowntimesByScope(ctx _context.Context, body CancelDowntimesByScopeRequest) (CanceledDowntimesIds, *_nethttp.Response, error) {
	req, err := a.buildCancelDowntimesByScopeRequest(ctx, body)
	if err != nil {
		var localVarReturnValue CanceledDowntimesIds
		return localVarReturnValue, nil, err
	}

	return a.cancelDowntimesByScopeExecute(req)
}

// cancelDowntimesByScopeExecute executes the request.
func (a *DowntimesApi) cancelDowntimesByScopeExecute(r apiCancelDowntimesByScopeRequest) (CanceledDowntimesIds, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue CanceledDowntimesIds
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.CancelDowntimesByScope")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime/cancel/by_scope"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiCreateDowntimeRequest struct {
	ctx  _context.Context
	body *Downtime
}

func (a *DowntimesApi) buildCreateDowntimeRequest(ctx _context.Context, body Downtime) (apiCreateDowntimeRequest, error) {
	req := apiCreateDowntimeRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateDowntime Schedule a downtime.
// Schedule a downtime.
func (a *DowntimesApi) CreateDowntime(ctx _context.Context, body Downtime) (Downtime, *_nethttp.Response, error) {
	req, err := a.buildCreateDowntimeRequest(ctx, body)
	if err != nil {
		var localVarReturnValue Downtime
		return localVarReturnValue, nil, err
	}

	return a.createDowntimeExecute(req)
}

// createDowntimeExecute executes the request.
func (a *DowntimesApi) createDowntimeExecute(r apiCreateDowntimeRequest) (Downtime, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue Downtime
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.CreateDowntime")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime"

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

type apiGetDowntimeRequest struct {
	ctx        _context.Context
	downtimeId int64
}

func (a *DowntimesApi) buildGetDowntimeRequest(ctx _context.Context, downtimeId int64) (apiGetDowntimeRequest, error) {
	req := apiGetDowntimeRequest{
		ctx:        ctx,
		downtimeId: downtimeId,
	}
	return req, nil
}

// GetDowntime Get a downtime.
// Get downtime detail by `downtime_id`.
func (a *DowntimesApi) GetDowntime(ctx _context.Context, downtimeId int64) (Downtime, *_nethttp.Response, error) {
	req, err := a.buildGetDowntimeRequest(ctx, downtimeId)
	if err != nil {
		var localVarReturnValue Downtime
		return localVarReturnValue, nil, err
	}

	return a.getDowntimeExecute(req)
}

// getDowntimeExecute executes the request.
func (a *DowntimesApi) getDowntimeExecute(r apiGetDowntimeRequest) (Downtime, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue Downtime
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.GetDowntime")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime/{downtime_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"downtime_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.downtimeId, "")), -1)

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

type apiListDowntimesRequest struct {
	ctx         _context.Context
	currentOnly *bool
}

// ListDowntimesOptionalParameters holds optional parameters for ListDowntimes.
type ListDowntimesOptionalParameters struct {
	CurrentOnly *bool
}

// NewListDowntimesOptionalParameters creates an empty struct for parameters.
func NewListDowntimesOptionalParameters() *ListDowntimesOptionalParameters {
	this := ListDowntimesOptionalParameters{}
	return &this
}

// WithCurrentOnly sets the corresponding parameter name and returns the struct.
func (r *ListDowntimesOptionalParameters) WithCurrentOnly(currentOnly bool) *ListDowntimesOptionalParameters {
	r.CurrentOnly = &currentOnly
	return r
}

func (a *DowntimesApi) buildListDowntimesRequest(ctx _context.Context, o ...ListDowntimesOptionalParameters) (apiListDowntimesRequest, error) {
	req := apiListDowntimesRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListDowntimesOptionalParameters is allowed")
	}

	if o != nil {
		req.currentOnly = o[0].CurrentOnly
	}
	return req, nil
}

// ListDowntimes Get all downtimes.
// Get all scheduled downtimes.
func (a *DowntimesApi) ListDowntimes(ctx _context.Context, o ...ListDowntimesOptionalParameters) ([]Downtime, *_nethttp.Response, error) {
	req, err := a.buildListDowntimesRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue []Downtime
		return localVarReturnValue, nil, err
	}

	return a.listDowntimesExecute(req)
}

// listDowntimesExecute executes the request.
func (a *DowntimesApi) listDowntimesExecute(r apiListDowntimesRequest) ([]Downtime, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue []Downtime
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.ListDowntimes")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.currentOnly != nil {
		localVarQueryParams.Add("current_only", datadog.ParameterToString(*r.currentOnly, ""))
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
		if localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
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

type apiListMonitorDowntimesRequest struct {
	ctx       _context.Context
	monitorId int64
}

func (a *DowntimesApi) buildListMonitorDowntimesRequest(ctx _context.Context, monitorId int64) (apiListMonitorDowntimesRequest, error) {
	req := apiListMonitorDowntimesRequest{
		ctx:       ctx,
		monitorId: monitorId,
	}
	return req, nil
}

// ListMonitorDowntimes Get all downtimes for a monitor.
// Get all active downtimes for the specified monitor.
func (a *DowntimesApi) ListMonitorDowntimes(ctx _context.Context, monitorId int64) ([]Downtime, *_nethttp.Response, error) {
	req, err := a.buildListMonitorDowntimesRequest(ctx, monitorId)
	if err != nil {
		var localVarReturnValue []Downtime
		return localVarReturnValue, nil, err
	}

	return a.listMonitorDowntimesExecute(req)
}

// listMonitorDowntimesExecute executes the request.
func (a *DowntimesApi) listMonitorDowntimesExecute(r apiListMonitorDowntimesRequest) ([]Downtime, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue []Downtime
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.ListMonitorDowntimes")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/monitor/{monitor_id}/downtimes"
	localVarPath = strings.Replace(localVarPath, "{"+"monitor_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.monitorId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateDowntimeRequest struct {
	ctx        _context.Context
	downtimeId int64
	body       *Downtime
}

func (a *DowntimesApi) buildUpdateDowntimeRequest(ctx _context.Context, downtimeId int64, body Downtime) (apiUpdateDowntimeRequest, error) {
	req := apiUpdateDowntimeRequest{
		ctx:        ctx,
		downtimeId: downtimeId,
		body:       &body,
	}
	return req, nil
}

// UpdateDowntime Update a downtime.
// Update a single downtime by `downtime_id`.
func (a *DowntimesApi) UpdateDowntime(ctx _context.Context, downtimeId int64, body Downtime) (Downtime, *_nethttp.Response, error) {
	req, err := a.buildUpdateDowntimeRequest(ctx, downtimeId, body)
	if err != nil {
		var localVarReturnValue Downtime
		return localVarReturnValue, nil, err
	}

	return a.updateDowntimeExecute(req)
}

// updateDowntimeExecute executes the request.
func (a *DowntimesApi) updateDowntimeExecute(r apiUpdateDowntimeRequest) (Downtime, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue Downtime
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DowntimesApi.UpdateDowntime")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/downtime/{downtime_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"downtime_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.downtimeId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

// NewDowntimesApi Returns NewDowntimesApi.
func NewDowntimesApi(client *datadog.APIClient) *DowntimesApi {
	return &DowntimesApi{
		Client: client,
	}
}
