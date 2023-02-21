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

// DashboardListsApi service type
type DashboardListsApi datadog.Service

type apiCreateDashboardListRequest struct {
	ctx  _context.Context
	body *DashboardList
}

func (a *DashboardListsApi) buildCreateDashboardListRequest(ctx _context.Context, body DashboardList) (apiCreateDashboardListRequest, error) {
	req := apiCreateDashboardListRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateDashboardList Create a dashboard list.
// Create an empty dashboard list.
func (a *DashboardListsApi) CreateDashboardList(ctx _context.Context, body DashboardList) (DashboardList, *_nethttp.Response, error) {
	req, err := a.buildCreateDashboardListRequest(ctx, body)
	if err != nil {
		var localVarReturnValue DashboardList
		return localVarReturnValue, nil, err
	}

	return a.createDashboardListExecute(req)
}

// createDashboardListExecute executes the request.
func (a *DashboardListsApi) createDashboardListExecute(r apiCreateDashboardListRequest) (DashboardList, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue DashboardList
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardListsApi.CreateDashboardList")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/lists/manual"

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

type apiDeleteDashboardListRequest struct {
	ctx    _context.Context
	listId int64
}

func (a *DashboardListsApi) buildDeleteDashboardListRequest(ctx _context.Context, listId int64) (apiDeleteDashboardListRequest, error) {
	req := apiDeleteDashboardListRequest{
		ctx:    ctx,
		listId: listId,
	}
	return req, nil
}

// DeleteDashboardList Delete a dashboard list.
// Delete a dashboard list.
func (a *DashboardListsApi) DeleteDashboardList(ctx _context.Context, listId int64) (DashboardListDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteDashboardListRequest(ctx, listId)
	if err != nil {
		var localVarReturnValue DashboardListDeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteDashboardListExecute(req)
}

// deleteDashboardListExecute executes the request.
func (a *DashboardListsApi) deleteDashboardListExecute(r apiDeleteDashboardListRequest) (DashboardListDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue DashboardListDeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardListsApi.DeleteDashboardList")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/lists/manual/{list_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.listId, "")), -1)

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

type apiGetDashboardListRequest struct {
	ctx    _context.Context
	listId int64
}

func (a *DashboardListsApi) buildGetDashboardListRequest(ctx _context.Context, listId int64) (apiGetDashboardListRequest, error) {
	req := apiGetDashboardListRequest{
		ctx:    ctx,
		listId: listId,
	}
	return req, nil
}

// GetDashboardList Get a dashboard list.
// Fetch an existing dashboard list's definition.
func (a *DashboardListsApi) GetDashboardList(ctx _context.Context, listId int64) (DashboardList, *_nethttp.Response, error) {
	req, err := a.buildGetDashboardListRequest(ctx, listId)
	if err != nil {
		var localVarReturnValue DashboardList
		return localVarReturnValue, nil, err
	}

	return a.getDashboardListExecute(req)
}

// getDashboardListExecute executes the request.
func (a *DashboardListsApi) getDashboardListExecute(r apiGetDashboardListRequest) (DashboardList, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue DashboardList
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardListsApi.GetDashboardList")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/lists/manual/{list_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.listId, "")), -1)

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

type apiListDashboardListsRequest struct {
	ctx _context.Context
}

func (a *DashboardListsApi) buildListDashboardListsRequest(ctx _context.Context) (apiListDashboardListsRequest, error) {
	req := apiListDashboardListsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListDashboardLists Get all dashboard lists.
// Fetch all of your existing dashboard list definitions.
func (a *DashboardListsApi) ListDashboardLists(ctx _context.Context) (DashboardListListResponse, *_nethttp.Response, error) {
	req, err := a.buildListDashboardListsRequest(ctx)
	if err != nil {
		var localVarReturnValue DashboardListListResponse
		return localVarReturnValue, nil, err
	}

	return a.listDashboardListsExecute(req)
}

// listDashboardListsExecute executes the request.
func (a *DashboardListsApi) listDashboardListsExecute(r apiListDashboardListsRequest) (DashboardListListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue DashboardListListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardListsApi.ListDashboardLists")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/lists/manual"

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

type apiUpdateDashboardListRequest struct {
	ctx    _context.Context
	listId int64
	body   *DashboardList
}

func (a *DashboardListsApi) buildUpdateDashboardListRequest(ctx _context.Context, listId int64, body DashboardList) (apiUpdateDashboardListRequest, error) {
	req := apiUpdateDashboardListRequest{
		ctx:    ctx,
		listId: listId,
		body:   &body,
	}
	return req, nil
}

// UpdateDashboardList Update a dashboard list.
// Update the name of a dashboard list.
func (a *DashboardListsApi) UpdateDashboardList(ctx _context.Context, listId int64, body DashboardList) (DashboardList, *_nethttp.Response, error) {
	req, err := a.buildUpdateDashboardListRequest(ctx, listId, body)
	if err != nil {
		var localVarReturnValue DashboardList
		return localVarReturnValue, nil, err
	}

	return a.updateDashboardListExecute(req)
}

// updateDashboardListExecute executes the request.
func (a *DashboardListsApi) updateDashboardListExecute(r apiUpdateDashboardListRequest) (DashboardList, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue DashboardList
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardListsApi.UpdateDashboardList")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/lists/manual/{list_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.listId, "")), -1)

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

// NewDashboardListsApi Returns NewDashboardListsApi.
func NewDashboardListsApi(client *datadog.APIClient) *DashboardListsApi {
	return &DashboardListsApi{
		Client: client,
	}
}
