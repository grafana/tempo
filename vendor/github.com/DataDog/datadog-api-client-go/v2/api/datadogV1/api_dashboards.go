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

// DashboardsApi service type
type DashboardsApi datadog.Service

type apiCreateDashboardRequest struct {
	ctx  _context.Context
	body *Dashboard
}

func (a *DashboardsApi) buildCreateDashboardRequest(ctx _context.Context, body Dashboard) (apiCreateDashboardRequest, error) {
	req := apiCreateDashboardRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateDashboard Create a new dashboard.
// Create a dashboard using the specified options. When defining queries in your widgets, take note of which queries should have the `as_count()` or `as_rate()` modifiers appended.
// Refer to the following [documentation](https://docs.datadoghq.com/developers/metrics/type_modifiers/?tab=count#in-application-modifiers) for more information on these modifiers.
func (a *DashboardsApi) CreateDashboard(ctx _context.Context, body Dashboard) (Dashboard, *_nethttp.Response, error) {
	req, err := a.buildCreateDashboardRequest(ctx, body)
	if err != nil {
		var localVarReturnValue Dashboard
		return localVarReturnValue, nil, err
	}

	return a.createDashboardExecute(req)
}

// createDashboardExecute executes the request.
func (a *DashboardsApi) createDashboardExecute(r apiCreateDashboardRequest) (Dashboard, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue Dashboard
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.CreateDashboard")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard"

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

type apiDeleteDashboardRequest struct {
	ctx         _context.Context
	dashboardId string
}

func (a *DashboardsApi) buildDeleteDashboardRequest(ctx _context.Context, dashboardId string) (apiDeleteDashboardRequest, error) {
	req := apiDeleteDashboardRequest{
		ctx:         ctx,
		dashboardId: dashboardId,
	}
	return req, nil
}

// DeleteDashboard Delete a dashboard.
// Delete a dashboard using the specified ID.
func (a *DashboardsApi) DeleteDashboard(ctx _context.Context, dashboardId string) (DashboardDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteDashboardRequest(ctx, dashboardId)
	if err != nil {
		var localVarReturnValue DashboardDeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteDashboardExecute(req)
}

// deleteDashboardExecute executes the request.
func (a *DashboardsApi) deleteDashboardExecute(r apiDeleteDashboardRequest) (DashboardDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue DashboardDeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.DeleteDashboard")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/{dashboard_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardId, "")), -1)

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

type apiDeleteDashboardsRequest struct {
	ctx  _context.Context
	body *DashboardBulkDeleteRequest
}

func (a *DashboardsApi) buildDeleteDashboardsRequest(ctx _context.Context, body DashboardBulkDeleteRequest) (apiDeleteDashboardsRequest, error) {
	req := apiDeleteDashboardsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteDashboards Delete dashboards.
// Delete dashboards using the specified IDs. If there are any failures, no dashboards will be deleted (partial success is not allowed).
func (a *DashboardsApi) DeleteDashboards(ctx _context.Context, body DashboardBulkDeleteRequest) (*_nethttp.Response, error) {
	req, err := a.buildDeleteDashboardsRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	return a.deleteDashboardsExecute(req)
}

// deleteDashboardsExecute executes the request.
func (a *DashboardsApi) deleteDashboardsExecute(r apiDeleteDashboardsRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.DeleteDashboards")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "*/*"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiGetDashboardRequest struct {
	ctx         _context.Context
	dashboardId string
}

func (a *DashboardsApi) buildGetDashboardRequest(ctx _context.Context, dashboardId string) (apiGetDashboardRequest, error) {
	req := apiGetDashboardRequest{
		ctx:         ctx,
		dashboardId: dashboardId,
	}
	return req, nil
}

// GetDashboard Get a dashboard.
// Get a dashboard using the specified ID.
func (a *DashboardsApi) GetDashboard(ctx _context.Context, dashboardId string) (Dashboard, *_nethttp.Response, error) {
	req, err := a.buildGetDashboardRequest(ctx, dashboardId)
	if err != nil {
		var localVarReturnValue Dashboard
		return localVarReturnValue, nil, err
	}

	return a.getDashboardExecute(req)
}

// getDashboardExecute executes the request.
func (a *DashboardsApi) getDashboardExecute(r apiGetDashboardRequest) (Dashboard, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue Dashboard
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.GetDashboard")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/{dashboard_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardId, "")), -1)

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

type apiListDashboardsRequest struct {
	ctx           _context.Context
	filterShared  *bool
	filterDeleted *bool
}

// ListDashboardsOptionalParameters holds optional parameters for ListDashboards.
type ListDashboardsOptionalParameters struct {
	FilterShared  *bool
	FilterDeleted *bool
}

// NewListDashboardsOptionalParameters creates an empty struct for parameters.
func NewListDashboardsOptionalParameters() *ListDashboardsOptionalParameters {
	this := ListDashboardsOptionalParameters{}
	return &this
}

// WithFilterShared sets the corresponding parameter name and returns the struct.
func (r *ListDashboardsOptionalParameters) WithFilterShared(filterShared bool) *ListDashboardsOptionalParameters {
	r.FilterShared = &filterShared
	return r
}

// WithFilterDeleted sets the corresponding parameter name and returns the struct.
func (r *ListDashboardsOptionalParameters) WithFilterDeleted(filterDeleted bool) *ListDashboardsOptionalParameters {
	r.FilterDeleted = &filterDeleted
	return r
}

func (a *DashboardsApi) buildListDashboardsRequest(ctx _context.Context, o ...ListDashboardsOptionalParameters) (apiListDashboardsRequest, error) {
	req := apiListDashboardsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListDashboardsOptionalParameters is allowed")
	}

	if o != nil {
		req.filterShared = o[0].FilterShared
		req.filterDeleted = o[0].FilterDeleted
	}
	return req, nil
}

// ListDashboards Get all dashboards.
// Get all dashboards.
//
// **Note**: This query will only return custom created or cloned dashboards.
// This query will not return preset dashboards.
func (a *DashboardsApi) ListDashboards(ctx _context.Context, o ...ListDashboardsOptionalParameters) (DashboardSummary, *_nethttp.Response, error) {
	req, err := a.buildListDashboardsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue DashboardSummary
		return localVarReturnValue, nil, err
	}

	return a.listDashboardsExecute(req)
}

// listDashboardsExecute executes the request.
func (a *DashboardsApi) listDashboardsExecute(r apiListDashboardsRequest) (DashboardSummary, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue DashboardSummary
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.ListDashboards")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filterShared != nil {
		localVarQueryParams.Add("filter[shared]", datadog.ParameterToString(*r.filterShared, ""))
	}
	if r.filterDeleted != nil {
		localVarQueryParams.Add("filter[deleted]", datadog.ParameterToString(*r.filterDeleted, ""))
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

type apiRestoreDashboardsRequest struct {
	ctx  _context.Context
	body *DashboardRestoreRequest
}

func (a *DashboardsApi) buildRestoreDashboardsRequest(ctx _context.Context, body DashboardRestoreRequest) (apiRestoreDashboardsRequest, error) {
	req := apiRestoreDashboardsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// RestoreDashboards Restore deleted dashboards.
// Restore dashboards using the specified IDs. If there are any failures, no dashboards will be restored (partial success is not allowed).
func (a *DashboardsApi) RestoreDashboards(ctx _context.Context, body DashboardRestoreRequest) (*_nethttp.Response, error) {
	req, err := a.buildRestoreDashboardsRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	return a.restoreDashboardsExecute(req)
}

// restoreDashboardsExecute executes the request.
func (a *DashboardsApi) restoreDashboardsExecute(r apiRestoreDashboardsRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodPatch
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.RestoreDashboards")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "*/*"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateDashboardRequest struct {
	ctx         _context.Context
	dashboardId string
	body        *Dashboard
}

func (a *DashboardsApi) buildUpdateDashboardRequest(ctx _context.Context, dashboardId string, body Dashboard) (apiUpdateDashboardRequest, error) {
	req := apiUpdateDashboardRequest{
		ctx:         ctx,
		dashboardId: dashboardId,
		body:        &body,
	}
	return req, nil
}

// UpdateDashboard Update a dashboard.
// Update a dashboard using the specified ID.
func (a *DashboardsApi) UpdateDashboard(ctx _context.Context, dashboardId string, body Dashboard) (Dashboard, *_nethttp.Response, error) {
	req, err := a.buildUpdateDashboardRequest(ctx, dashboardId, body)
	if err != nil {
		var localVarReturnValue Dashboard
		return localVarReturnValue, nil, err
	}

	return a.updateDashboardExecute(req)
}

// updateDashboardExecute executes the request.
func (a *DashboardsApi) updateDashboardExecute(r apiUpdateDashboardRequest) (Dashboard, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue Dashboard
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.DashboardsApi.UpdateDashboard")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/dashboard/{dashboard_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"dashboard_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.dashboardId, "")), -1)

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

// NewDashboardsApi Returns NewDashboardsApi.
func NewDashboardsApi(client *datadog.APIClient) *DashboardsApi {
	return &DashboardsApi{
		Client: client,
	}
}
