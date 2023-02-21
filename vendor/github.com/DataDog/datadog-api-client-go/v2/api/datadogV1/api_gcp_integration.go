// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// GCPIntegrationApi service type
type GCPIntegrationApi datadog.Service

type apiCreateGCPIntegrationRequest struct {
	ctx  _context.Context
	body *GCPAccount
}

func (a *GCPIntegrationApi) buildCreateGCPIntegrationRequest(ctx _context.Context, body GCPAccount) (apiCreateGCPIntegrationRequest, error) {
	req := apiCreateGCPIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateGCPIntegration Create a GCP integration.
// Create a Datadog-GCP integration.
func (a *GCPIntegrationApi) CreateGCPIntegration(ctx _context.Context, body GCPAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildCreateGCPIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.createGCPIntegrationExecute(req)
}

// createGCPIntegrationExecute executes the request.
func (a *GCPIntegrationApi) createGCPIntegrationExecute(r apiCreateGCPIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.GCPIntegrationApi.CreateGCPIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/gcp"

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

type apiDeleteGCPIntegrationRequest struct {
	ctx  _context.Context
	body *GCPAccount
}

func (a *GCPIntegrationApi) buildDeleteGCPIntegrationRequest(ctx _context.Context, body GCPAccount) (apiDeleteGCPIntegrationRequest, error) {
	req := apiDeleteGCPIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteGCPIntegration Delete a GCP integration.
// Delete a given Datadog-GCP integration.
func (a *GCPIntegrationApi) DeleteGCPIntegration(ctx _context.Context, body GCPAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildDeleteGCPIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.deleteGCPIntegrationExecute(req)
}

// deleteGCPIntegrationExecute executes the request.
func (a *GCPIntegrationApi) deleteGCPIntegrationExecute(r apiDeleteGCPIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.GCPIntegrationApi.DeleteGCPIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/gcp"

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

type apiListGCPIntegrationRequest struct {
	ctx _context.Context
}

func (a *GCPIntegrationApi) buildListGCPIntegrationRequest(ctx _context.Context) (apiListGCPIntegrationRequest, error) {
	req := apiListGCPIntegrationRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListGCPIntegration List all GCP integrations.
// List all Datadog-GCP integrations configured in your Datadog account.
func (a *GCPIntegrationApi) ListGCPIntegration(ctx _context.Context) ([]GCPAccount, *_nethttp.Response, error) {
	req, err := a.buildListGCPIntegrationRequest(ctx)
	if err != nil {
		var localVarReturnValue []GCPAccount
		return localVarReturnValue, nil, err
	}

	return a.listGCPIntegrationExecute(req)
}

// listGCPIntegrationExecute executes the request.
func (a *GCPIntegrationApi) listGCPIntegrationExecute(r apiListGCPIntegrationRequest) ([]GCPAccount, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue []GCPAccount
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.GCPIntegrationApi.ListGCPIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/gcp"

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

type apiUpdateGCPIntegrationRequest struct {
	ctx  _context.Context
	body *GCPAccount
}

func (a *GCPIntegrationApi) buildUpdateGCPIntegrationRequest(ctx _context.Context, body GCPAccount) (apiUpdateGCPIntegrationRequest, error) {
	req := apiUpdateGCPIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// UpdateGCPIntegration Update a GCP integration.
// Update a Datadog-GCP integrations host_filters and/or auto-mute.
// Requires a `project_id` and `client_email`, however these fields cannot be updated.
// If you need to update these fields, delete and use the create (`POST`) endpoint.
// The unspecified fields will keep their original values.
func (a *GCPIntegrationApi) UpdateGCPIntegration(ctx _context.Context, body GCPAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildUpdateGCPIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.updateGCPIntegrationExecute(req)
}

// updateGCPIntegrationExecute executes the request.
func (a *GCPIntegrationApi) updateGCPIntegrationExecute(r apiUpdateGCPIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.GCPIntegrationApi.UpdateGCPIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/gcp"

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

// NewGCPIntegrationApi Returns NewGCPIntegrationApi.
func NewGCPIntegrationApi(client *datadog.APIClient) *GCPIntegrationApi {
	return &GCPIntegrationApi{
		Client: client,
	}
}
