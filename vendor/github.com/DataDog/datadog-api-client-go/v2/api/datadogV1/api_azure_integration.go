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

// AzureIntegrationApi service type
type AzureIntegrationApi datadog.Service

type apiCreateAzureIntegrationRequest struct {
	ctx  _context.Context
	body *AzureAccount
}

func (a *AzureIntegrationApi) buildCreateAzureIntegrationRequest(ctx _context.Context, body AzureAccount) (apiCreateAzureIntegrationRequest, error) {
	req := apiCreateAzureIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateAzureIntegration Create an Azure integration.
// Create a Datadog-Azure integration.
//
// Using the `POST` method updates your integration configuration by adding your new
// configuration to the existing one in your Datadog organization.
//
// Using the `PUT` method updates your integration configuration by replacing your
// current configuration with the new one sent to your Datadog organization.
func (a *AzureIntegrationApi) CreateAzureIntegration(ctx _context.Context, body AzureAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildCreateAzureIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.createAzureIntegrationExecute(req)
}

// createAzureIntegrationExecute executes the request.
func (a *AzureIntegrationApi) createAzureIntegrationExecute(r apiCreateAzureIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AzureIntegrationApi.CreateAzureIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/azure"

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

type apiDeleteAzureIntegrationRequest struct {
	ctx  _context.Context
	body *AzureAccount
}

func (a *AzureIntegrationApi) buildDeleteAzureIntegrationRequest(ctx _context.Context, body AzureAccount) (apiDeleteAzureIntegrationRequest, error) {
	req := apiDeleteAzureIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteAzureIntegration Delete an Azure integration.
// Delete a given Datadog-Azure integration from your Datadog account.
func (a *AzureIntegrationApi) DeleteAzureIntegration(ctx _context.Context, body AzureAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildDeleteAzureIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.deleteAzureIntegrationExecute(req)
}

// deleteAzureIntegrationExecute executes the request.
func (a *AzureIntegrationApi) deleteAzureIntegrationExecute(r apiDeleteAzureIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AzureIntegrationApi.DeleteAzureIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/azure"

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

type apiListAzureIntegrationRequest struct {
	ctx _context.Context
}

func (a *AzureIntegrationApi) buildListAzureIntegrationRequest(ctx _context.Context) (apiListAzureIntegrationRequest, error) {
	req := apiListAzureIntegrationRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListAzureIntegration List all Azure integrations.
// List all Datadog-Azure integrations configured in your Datadog account.
func (a *AzureIntegrationApi) ListAzureIntegration(ctx _context.Context) ([]AzureAccount, *_nethttp.Response, error) {
	req, err := a.buildListAzureIntegrationRequest(ctx)
	if err != nil {
		var localVarReturnValue []AzureAccount
		return localVarReturnValue, nil, err
	}

	return a.listAzureIntegrationExecute(req)
}

// listAzureIntegrationExecute executes the request.
func (a *AzureIntegrationApi) listAzureIntegrationExecute(r apiListAzureIntegrationRequest) ([]AzureAccount, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue []AzureAccount
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AzureIntegrationApi.ListAzureIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/azure"

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

type apiUpdateAzureHostFiltersRequest struct {
	ctx  _context.Context
	body *AzureAccount
}

func (a *AzureIntegrationApi) buildUpdateAzureHostFiltersRequest(ctx _context.Context, body AzureAccount) (apiUpdateAzureHostFiltersRequest, error) {
	req := apiUpdateAzureHostFiltersRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// UpdateAzureHostFilters Update Azure integration host filters.
// Update the defined list of host filters for a given Datadog-Azure integration.
func (a *AzureIntegrationApi) UpdateAzureHostFilters(ctx _context.Context, body AzureAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildUpdateAzureHostFiltersRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.updateAzureHostFiltersExecute(req)
}

// updateAzureHostFiltersExecute executes the request.
func (a *AzureIntegrationApi) updateAzureHostFiltersExecute(r apiUpdateAzureHostFiltersRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AzureIntegrationApi.UpdateAzureHostFilters")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/azure/host_filters"

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

type apiUpdateAzureIntegrationRequest struct {
	ctx  _context.Context
	body *AzureAccount
}

func (a *AzureIntegrationApi) buildUpdateAzureIntegrationRequest(ctx _context.Context, body AzureAccount) (apiUpdateAzureIntegrationRequest, error) {
	req := apiUpdateAzureIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// UpdateAzureIntegration Update an Azure integration.
// Update a Datadog-Azure integration. Requires an existing `tenant_name` and `client_id`.
// Any other fields supplied will overwrite existing values. To overwrite `tenant_name` or `client_id`,
// use `new_tenant_name` and `new_client_id`. To leave a field unchanged, do not supply that field in the payload.
func (a *AzureIntegrationApi) UpdateAzureIntegration(ctx _context.Context, body AzureAccount) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildUpdateAzureIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.updateAzureIntegrationExecute(req)
}

// updateAzureIntegrationExecute executes the request.
func (a *AzureIntegrationApi) updateAzureIntegrationExecute(r apiUpdateAzureIntegrationRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AzureIntegrationApi.UpdateAzureIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/azure"

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

// NewAzureIntegrationApi Returns NewAzureIntegrationApi.
func NewAzureIntegrationApi(client *datadog.APIClient) *AzureIntegrationApi {
	return &AzureIntegrationApi{
		Client: client,
	}
}
