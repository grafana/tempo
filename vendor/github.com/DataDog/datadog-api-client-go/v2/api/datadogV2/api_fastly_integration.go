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

// FastlyIntegrationApi service type
type FastlyIntegrationApi datadog.Service

type apiCreateFastlyAccountRequest struct {
	ctx  _context.Context
	body *FastlyAccountCreateRequest
}

func (a *FastlyIntegrationApi) buildCreateFastlyAccountRequest(ctx _context.Context, body FastlyAccountCreateRequest) (apiCreateFastlyAccountRequest, error) {
	req := apiCreateFastlyAccountRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateFastlyAccount Add Fastly account.
// Create a Fastly account.
func (a *FastlyIntegrationApi) CreateFastlyAccount(ctx _context.Context, body FastlyAccountCreateRequest) (FastlyAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateFastlyAccountRequest(ctx, body)
	if err != nil {
		var localVarReturnValue FastlyAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.createFastlyAccountExecute(req)
}

// createFastlyAccountExecute executes the request.
func (a *FastlyIntegrationApi) createFastlyAccountExecute(r apiCreateFastlyAccountRequest) (FastlyAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue FastlyAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.CreateFastlyAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts"

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

type apiCreateFastlyServiceRequest struct {
	ctx       _context.Context
	accountId string
	body      *FastlyServiceRequest
}

func (a *FastlyIntegrationApi) buildCreateFastlyServiceRequest(ctx _context.Context, accountId string, body FastlyServiceRequest) (apiCreateFastlyServiceRequest, error) {
	req := apiCreateFastlyServiceRequest{
		ctx:       ctx,
		accountId: accountId,
		body:      &body,
	}
	return req, nil
}

// CreateFastlyService Add Fastly service.
// Create a Fastly service for an account.
func (a *FastlyIntegrationApi) CreateFastlyService(ctx _context.Context, accountId string, body FastlyServiceRequest) (FastlyServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateFastlyServiceRequest(ctx, accountId, body)
	if err != nil {
		var localVarReturnValue FastlyServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.createFastlyServiceExecute(req)
}

// createFastlyServiceExecute executes the request.
func (a *FastlyIntegrationApi) createFastlyServiceExecute(r apiCreateFastlyServiceRequest) (FastlyServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue FastlyServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.CreateFastlyService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}/services"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)

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

type apiDeleteFastlyAccountRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *FastlyIntegrationApi) buildDeleteFastlyAccountRequest(ctx _context.Context, accountId string) (apiDeleteFastlyAccountRequest, error) {
	req := apiDeleteFastlyAccountRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// DeleteFastlyAccount Delete Fastly account.
// Delete a Fastly account.
func (a *FastlyIntegrationApi) DeleteFastlyAccount(ctx _context.Context, accountId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteFastlyAccountRequest(ctx, accountId)
	if err != nil {
		return nil, err
	}

	return a.deleteFastlyAccountExecute(req)
}

// deleteFastlyAccountExecute executes the request.
func (a *FastlyIntegrationApi) deleteFastlyAccountExecute(r apiDeleteFastlyAccountRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.DeleteFastlyAccount")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)

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

type apiDeleteFastlyServiceRequest struct {
	ctx       _context.Context
	accountId string
	serviceId string
}

func (a *FastlyIntegrationApi) buildDeleteFastlyServiceRequest(ctx _context.Context, accountId string, serviceId string) (apiDeleteFastlyServiceRequest, error) {
	req := apiDeleteFastlyServiceRequest{
		ctx:       ctx,
		accountId: accountId,
		serviceId: serviceId,
	}
	return req, nil
}

// DeleteFastlyService Delete Fastly service.
// Delete a Fastly service for an account.
func (a *FastlyIntegrationApi) DeleteFastlyService(ctx _context.Context, accountId string, serviceId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteFastlyServiceRequest(ctx, accountId, serviceId)
	if err != nil {
		return nil, err
	}

	return a.deleteFastlyServiceExecute(req)
}

// deleteFastlyServiceExecute executes the request.
func (a *FastlyIntegrationApi) deleteFastlyServiceExecute(r apiDeleteFastlyServiceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.DeleteFastlyService")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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

type apiGetFastlyAccountRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *FastlyIntegrationApi) buildGetFastlyAccountRequest(ctx _context.Context, accountId string) (apiGetFastlyAccountRequest, error) {
	req := apiGetFastlyAccountRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// GetFastlyAccount Get Fastly account.
// Get a Fastly account.
func (a *FastlyIntegrationApi) GetFastlyAccount(ctx _context.Context, accountId string) (FastlyAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildGetFastlyAccountRequest(ctx, accountId)
	if err != nil {
		var localVarReturnValue FastlyAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.getFastlyAccountExecute(req)
}

// getFastlyAccountExecute executes the request.
func (a *FastlyIntegrationApi) getFastlyAccountExecute(r apiGetFastlyAccountRequest) (FastlyAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue FastlyAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.GetFastlyAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)

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

type apiGetFastlyServiceRequest struct {
	ctx       _context.Context
	accountId string
	serviceId string
}

func (a *FastlyIntegrationApi) buildGetFastlyServiceRequest(ctx _context.Context, accountId string, serviceId string) (apiGetFastlyServiceRequest, error) {
	req := apiGetFastlyServiceRequest{
		ctx:       ctx,
		accountId: accountId,
		serviceId: serviceId,
	}
	return req, nil
}

// GetFastlyService Get Fastly service.
// Get a Fastly service for an account.
func (a *FastlyIntegrationApi) GetFastlyService(ctx _context.Context, accountId string, serviceId string) (FastlyServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildGetFastlyServiceRequest(ctx, accountId, serviceId)
	if err != nil {
		var localVarReturnValue FastlyServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.getFastlyServiceExecute(req)
}

// getFastlyServiceExecute executes the request.
func (a *FastlyIntegrationApi) getFastlyServiceExecute(r apiGetFastlyServiceRequest) (FastlyServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue FastlyServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.GetFastlyService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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

type apiListFastlyAccountsRequest struct {
	ctx _context.Context
}

func (a *FastlyIntegrationApi) buildListFastlyAccountsRequest(ctx _context.Context) (apiListFastlyAccountsRequest, error) {
	req := apiListFastlyAccountsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListFastlyAccounts List Fastly accounts.
// List Fastly accounts.
func (a *FastlyIntegrationApi) ListFastlyAccounts(ctx _context.Context) (FastlyAccountsResponse, *_nethttp.Response, error) {
	req, err := a.buildListFastlyAccountsRequest(ctx)
	if err != nil {
		var localVarReturnValue FastlyAccountsResponse
		return localVarReturnValue, nil, err
	}

	return a.listFastlyAccountsExecute(req)
}

// listFastlyAccountsExecute executes the request.
func (a *FastlyIntegrationApi) listFastlyAccountsExecute(r apiListFastlyAccountsRequest) (FastlyAccountsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue FastlyAccountsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.ListFastlyAccounts")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts"

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

type apiListFastlyServicesRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *FastlyIntegrationApi) buildListFastlyServicesRequest(ctx _context.Context, accountId string) (apiListFastlyServicesRequest, error) {
	req := apiListFastlyServicesRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// ListFastlyServices List Fastly services.
// List Fastly services for an account.
func (a *FastlyIntegrationApi) ListFastlyServices(ctx _context.Context, accountId string) (FastlyServicesResponse, *_nethttp.Response, error) {
	req, err := a.buildListFastlyServicesRequest(ctx, accountId)
	if err != nil {
		var localVarReturnValue FastlyServicesResponse
		return localVarReturnValue, nil, err
	}

	return a.listFastlyServicesExecute(req)
}

// listFastlyServicesExecute executes the request.
func (a *FastlyIntegrationApi) listFastlyServicesExecute(r apiListFastlyServicesRequest) (FastlyServicesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue FastlyServicesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.ListFastlyServices")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}/services"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)

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

type apiUpdateFastlyAccountRequest struct {
	ctx       _context.Context
	accountId string
	body      *FastlyAccountUpdateRequest
}

func (a *FastlyIntegrationApi) buildUpdateFastlyAccountRequest(ctx _context.Context, accountId string, body FastlyAccountUpdateRequest) (apiUpdateFastlyAccountRequest, error) {
	req := apiUpdateFastlyAccountRequest{
		ctx:       ctx,
		accountId: accountId,
		body:      &body,
	}
	return req, nil
}

// UpdateFastlyAccount Update Fastly account.
// Update a Fastly account.
func (a *FastlyIntegrationApi) UpdateFastlyAccount(ctx _context.Context, accountId string, body FastlyAccountUpdateRequest) (FastlyAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateFastlyAccountRequest(ctx, accountId, body)
	if err != nil {
		var localVarReturnValue FastlyAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.updateFastlyAccountExecute(req)
}

// updateFastlyAccountExecute executes the request.
func (a *FastlyIntegrationApi) updateFastlyAccountExecute(r apiUpdateFastlyAccountRequest) (FastlyAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue FastlyAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.UpdateFastlyAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)

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

type apiUpdateFastlyServiceRequest struct {
	ctx       _context.Context
	accountId string
	serviceId string
	body      *FastlyServiceRequest
}

func (a *FastlyIntegrationApi) buildUpdateFastlyServiceRequest(ctx _context.Context, accountId string, serviceId string, body FastlyServiceRequest) (apiUpdateFastlyServiceRequest, error) {
	req := apiUpdateFastlyServiceRequest{
		ctx:       ctx,
		accountId: accountId,
		serviceId: serviceId,
		body:      &body,
	}
	return req, nil
}

// UpdateFastlyService Update Fastly service.
// Update a Fastly service for an account.
func (a *FastlyIntegrationApi) UpdateFastlyService(ctx _context.Context, accountId string, serviceId string, body FastlyServiceRequest) (FastlyServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateFastlyServiceRequest(ctx, accountId, serviceId, body)
	if err != nil {
		var localVarReturnValue FastlyServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.updateFastlyServiceExecute(req)
}

// updateFastlyServiceExecute executes the request.
func (a *FastlyIntegrationApi) updateFastlyServiceExecute(r apiUpdateFastlyServiceRequest) (FastlyServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue FastlyServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.FastlyIntegrationApi.UpdateFastlyService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/fastly/accounts/{account_id}/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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

// NewFastlyIntegrationApi Returns NewFastlyIntegrationApi.
func NewFastlyIntegrationApi(client *datadog.APIClient) *FastlyIntegrationApi {
	return &FastlyIntegrationApi{
		Client: client,
	}
}
