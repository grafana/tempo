// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// DashboardListsApi service type
type DashboardListsApi datadog.Service

type apiCreateDashboardListItemsRequest struct {
	ctx             _context.Context
	dashboardListId int64
	body            *DashboardListAddItemsRequest
}

func (a *DashboardListsApi) buildCreateDashboardListItemsRequest(ctx _context.Context, dashboardListId int64, body DashboardListAddItemsRequest) (apiCreateDashboardListItemsRequest, error) {
	req := apiCreateDashboardListItemsRequest{
		ctx:             ctx,
		dashboardListId: dashboardListId,
		body:            &body,
	}
	return req, nil
}

// CreateDashboardListItems Add Items to a Dashboard List.
// Add dashboards to an existing dashboard list.
func (a *DashboardListsApi) CreateDashboardListItems(ctx _context.Context, dashboardListId int64, body DashboardListAddItemsRequest) (DashboardListAddItemsResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateDashboardListItemsRequest(ctx, dashboardListId, body)
	if err != nil {
		var localVarReturnValue DashboardListAddItemsResponse
		return localVarReturnValue, nil, err
	}

	return a.createDashboardListItemsExecute(req)
}

// createDashboardListItemsExecute executes the request.
func (a *DashboardListsApi) createDashboardListItemsExecute(r apiCreateDashboardListItemsRequest) (DashboardListAddItemsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue DashboardListAddItemsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.DashboardListsApi.CreateDashboardListItems")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/dashboard/lists/manual/{dashboard_list_id}/dashboards"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardListId, "")), -1)

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

type apiDeleteDashboardListItemsRequest struct {
	ctx             _context.Context
	dashboardListId int64
	body            *DashboardListDeleteItemsRequest
}

func (a *DashboardListsApi) buildDeleteDashboardListItemsRequest(ctx _context.Context, dashboardListId int64, body DashboardListDeleteItemsRequest) (apiDeleteDashboardListItemsRequest, error) {
	req := apiDeleteDashboardListItemsRequest{
		ctx:             ctx,
		dashboardListId: dashboardListId,
		body:            &body,
	}
	return req, nil
}

// DeleteDashboardListItems Delete items from a dashboard list.
// Delete dashboards from an existing dashboard list.
func (a *DashboardListsApi) DeleteDashboardListItems(ctx _context.Context, dashboardListId int64, body DashboardListDeleteItemsRequest) (DashboardListDeleteItemsResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteDashboardListItemsRequest(ctx, dashboardListId, body)
	if err != nil {
		var localVarReturnValue DashboardListDeleteItemsResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteDashboardListItemsExecute(req)
}

// deleteDashboardListItemsExecute executes the request.
func (a *DashboardListsApi) deleteDashboardListItemsExecute(r apiDeleteDashboardListItemsRequest) (DashboardListDeleteItemsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue DashboardListDeleteItemsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.DashboardListsApi.DeleteDashboardListItems")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/dashboard/lists/manual/{dashboard_list_id}/dashboards"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardListId, "")), -1)

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

type apiGetDashboardListItemsRequest struct {
	ctx             _context.Context
	dashboardListId int64
}

func (a *DashboardListsApi) buildGetDashboardListItemsRequest(ctx _context.Context, dashboardListId int64) (apiGetDashboardListItemsRequest, error) {
	req := apiGetDashboardListItemsRequest{
		ctx:             ctx,
		dashboardListId: dashboardListId,
	}
	return req, nil
}

// GetDashboardListItems Get items of a Dashboard List.
// Fetch the dashboard list’s dashboard definitions.
func (a *DashboardListsApi) GetDashboardListItems(ctx _context.Context, dashboardListId int64) (DashboardListItems, *_nethttp.Response, error) {
	req, err := a.buildGetDashboardListItemsRequest(ctx, dashboardListId)
	if err != nil {
		var localVarReturnValue DashboardListItems
		return localVarReturnValue, nil, err
	}

	return a.getDashboardListItemsExecute(req)
}

// getDashboardListItemsExecute executes the request.
func (a *DashboardListsApi) getDashboardListItemsExecute(r apiGetDashboardListItemsRequest) (DashboardListItems, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue DashboardListItems
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.DashboardListsApi.GetDashboardListItems")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/dashboard/lists/manual/{dashboard_list_id}/dashboards"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardListId, "")), -1)

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

type apiUpdateDashboardListItemsRequest struct {
	ctx             _context.Context
	dashboardListId int64
	body            *DashboardListUpdateItemsRequest
}

func (a *DashboardListsApi) buildUpdateDashboardListItemsRequest(ctx _context.Context, dashboardListId int64, body DashboardListUpdateItemsRequest) (apiUpdateDashboardListItemsRequest, error) {
	req := apiUpdateDashboardListItemsRequest{
		ctx:             ctx,
		dashboardListId: dashboardListId,
		body:            &body,
	}
	return req, nil
}

// UpdateDashboardListItems Update items of a dashboard list.
// Update dashboards of an existing dashboard list.
func (a *DashboardListsApi) UpdateDashboardListItems(ctx _context.Context, dashboardListId int64, body DashboardListUpdateItemsRequest) (DashboardListUpdateItemsResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateDashboardListItemsRequest(ctx, dashboardListId, body)
	if err != nil {
		var localVarReturnValue DashboardListUpdateItemsResponse
		return localVarReturnValue, nil, err
	}

	return a.updateDashboardListItemsExecute(req)
}

// updateDashboardListItemsExecute executes the request.
func (a *DashboardListsApi) updateDashboardListItemsExecute(r apiUpdateDashboardListItemsRequest) (DashboardListUpdateItemsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue DashboardListUpdateItemsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.DashboardListsApi.UpdateDashboardListItems")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/dashboard/lists/manual/{dashboard_list_id}/dashboards"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_list_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardListId, "")), -1)

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
