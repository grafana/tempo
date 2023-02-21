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

// ConfluentCloudApi service type
type ConfluentCloudApi datadog.Service

type apiCreateConfluentAccountRequest struct {
	ctx  _context.Context
	body *ConfluentAccountCreateRequest
}

func (a *ConfluentCloudApi) buildCreateConfluentAccountRequest(ctx _context.Context, body ConfluentAccountCreateRequest) (apiCreateConfluentAccountRequest, error) {
	req := apiCreateConfluentAccountRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateConfluentAccount Add Confluent account.
// Create a Confluent account.
func (a *ConfluentCloudApi) CreateConfluentAccount(ctx _context.Context, body ConfluentAccountCreateRequest) (ConfluentAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateConfluentAccountRequest(ctx, body)
	if err != nil {
		var localVarReturnValue ConfluentAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.createConfluentAccountExecute(req)
}

// createConfluentAccountExecute executes the request.
func (a *ConfluentCloudApi) createConfluentAccountExecute(r apiCreateConfluentAccountRequest) (ConfluentAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue ConfluentAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.CreateConfluentAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts"

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

type apiCreateConfluentResourceRequest struct {
	ctx       _context.Context
	accountId string
	body      *ConfluentResourceRequest
}

func (a *ConfluentCloudApi) buildCreateConfluentResourceRequest(ctx _context.Context, accountId string, body ConfluentResourceRequest) (apiCreateConfluentResourceRequest, error) {
	req := apiCreateConfluentResourceRequest{
		ctx:       ctx,
		accountId: accountId,
		body:      &body,
	}
	return req, nil
}

// CreateConfluentResource Add resource to Confluent account.
// Create a Confluent resource for the account associated with the provided ID.
func (a *ConfluentCloudApi) CreateConfluentResource(ctx _context.Context, accountId string, body ConfluentResourceRequest) (ConfluentResourceResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateConfluentResourceRequest(ctx, accountId, body)
	if err != nil {
		var localVarReturnValue ConfluentResourceResponse
		return localVarReturnValue, nil, err
	}

	return a.createConfluentResourceExecute(req)
}

// createConfluentResourceExecute executes the request.
func (a *ConfluentCloudApi) createConfluentResourceExecute(r apiCreateConfluentResourceRequest) (ConfluentResourceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue ConfluentResourceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.CreateConfluentResource")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}/resources"
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

type apiDeleteConfluentAccountRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *ConfluentCloudApi) buildDeleteConfluentAccountRequest(ctx _context.Context, accountId string) (apiDeleteConfluentAccountRequest, error) {
	req := apiDeleteConfluentAccountRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// DeleteConfluentAccount Delete Confluent account.
// Delete a Confluent account with the provided account ID.
func (a *ConfluentCloudApi) DeleteConfluentAccount(ctx _context.Context, accountId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteConfluentAccountRequest(ctx, accountId)
	if err != nil {
		return nil, err
	}

	return a.deleteConfluentAccountExecute(req)
}

// deleteConfluentAccountExecute executes the request.
func (a *ConfluentCloudApi) deleteConfluentAccountExecute(r apiDeleteConfluentAccountRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.DeleteConfluentAccount")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}"
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

type apiDeleteConfluentResourceRequest struct {
	ctx        _context.Context
	accountId  string
	resourceId string
}

func (a *ConfluentCloudApi) buildDeleteConfluentResourceRequest(ctx _context.Context, accountId string, resourceId string) (apiDeleteConfluentResourceRequest, error) {
	req := apiDeleteConfluentResourceRequest{
		ctx:        ctx,
		accountId:  accountId,
		resourceId: resourceId,
	}
	return req, nil
}

// DeleteConfluentResource Delete resource from Confluent account.
// Delete a Confluent resource with the provided resource id for the account associated with the provided account ID.
func (a *ConfluentCloudApi) DeleteConfluentResource(ctx _context.Context, accountId string, resourceId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteConfluentResourceRequest(ctx, accountId, resourceId)
	if err != nil {
		return nil, err
	}

	return a.deleteConfluentResourceExecute(req)
}

// deleteConfluentResourceExecute executes the request.
func (a *ConfluentCloudApi) deleteConfluentResourceExecute(r apiDeleteConfluentResourceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.DeleteConfluentResource")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}/resources/{resource_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"resource_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.resourceId, "")), -1)

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

type apiGetConfluentAccountRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *ConfluentCloudApi) buildGetConfluentAccountRequest(ctx _context.Context, accountId string) (apiGetConfluentAccountRequest, error) {
	req := apiGetConfluentAccountRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// GetConfluentAccount Get Confluent account.
// Get the Confluent account with the provided account ID.
func (a *ConfluentCloudApi) GetConfluentAccount(ctx _context.Context, accountId string) (ConfluentAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildGetConfluentAccountRequest(ctx, accountId)
	if err != nil {
		var localVarReturnValue ConfluentAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.getConfluentAccountExecute(req)
}

// getConfluentAccountExecute executes the request.
func (a *ConfluentCloudApi) getConfluentAccountExecute(r apiGetConfluentAccountRequest) (ConfluentAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ConfluentAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.GetConfluentAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}"
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

type apiGetConfluentResourceRequest struct {
	ctx        _context.Context
	accountId  string
	resourceId string
}

func (a *ConfluentCloudApi) buildGetConfluentResourceRequest(ctx _context.Context, accountId string, resourceId string) (apiGetConfluentResourceRequest, error) {
	req := apiGetConfluentResourceRequest{
		ctx:        ctx,
		accountId:  accountId,
		resourceId: resourceId,
	}
	return req, nil
}

// GetConfluentResource Get resource from Confluent account.
// Get a Confluent resource with the provided resource id for the account associated with the provided account ID.
func (a *ConfluentCloudApi) GetConfluentResource(ctx _context.Context, accountId string, resourceId string) (ConfluentResourceResponse, *_nethttp.Response, error) {
	req, err := a.buildGetConfluentResourceRequest(ctx, accountId, resourceId)
	if err != nil {
		var localVarReturnValue ConfluentResourceResponse
		return localVarReturnValue, nil, err
	}

	return a.getConfluentResourceExecute(req)
}

// getConfluentResourceExecute executes the request.
func (a *ConfluentCloudApi) getConfluentResourceExecute(r apiGetConfluentResourceRequest) (ConfluentResourceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ConfluentResourceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.GetConfluentResource")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}/resources/{resource_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"resource_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.resourceId, "")), -1)

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

type apiListConfluentAccountRequest struct {
	ctx _context.Context
}

func (a *ConfluentCloudApi) buildListConfluentAccountRequest(ctx _context.Context) (apiListConfluentAccountRequest, error) {
	req := apiListConfluentAccountRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListConfluentAccount List Confluent accounts.
// List Confluent accounts.
func (a *ConfluentCloudApi) ListConfluentAccount(ctx _context.Context) (ConfluentAccountsResponse, *_nethttp.Response, error) {
	req, err := a.buildListConfluentAccountRequest(ctx)
	if err != nil {
		var localVarReturnValue ConfluentAccountsResponse
		return localVarReturnValue, nil, err
	}

	return a.listConfluentAccountExecute(req)
}

// listConfluentAccountExecute executes the request.
func (a *ConfluentCloudApi) listConfluentAccountExecute(r apiListConfluentAccountRequest) (ConfluentAccountsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ConfluentAccountsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.ListConfluentAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts"

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

type apiListConfluentResourceRequest struct {
	ctx       _context.Context
	accountId string
}

func (a *ConfluentCloudApi) buildListConfluentResourceRequest(ctx _context.Context, accountId string) (apiListConfluentResourceRequest, error) {
	req := apiListConfluentResourceRequest{
		ctx:       ctx,
		accountId: accountId,
	}
	return req, nil
}

// ListConfluentResource List Confluent Account resources.
// Get a Confluent resource for the account associated with the provided ID.
func (a *ConfluentCloudApi) ListConfluentResource(ctx _context.Context, accountId string) (ConfluentResourcesResponse, *_nethttp.Response, error) {
	req, err := a.buildListConfluentResourceRequest(ctx, accountId)
	if err != nil {
		var localVarReturnValue ConfluentResourcesResponse
		return localVarReturnValue, nil, err
	}

	return a.listConfluentResourceExecute(req)
}

// listConfluentResourceExecute executes the request.
func (a *ConfluentCloudApi) listConfluentResourceExecute(r apiListConfluentResourceRequest) (ConfluentResourcesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ConfluentResourcesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.ListConfluentResource")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}/resources"
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

type apiUpdateConfluentAccountRequest struct {
	ctx       _context.Context
	accountId string
	body      *ConfluentAccountUpdateRequest
}

func (a *ConfluentCloudApi) buildUpdateConfluentAccountRequest(ctx _context.Context, accountId string, body ConfluentAccountUpdateRequest) (apiUpdateConfluentAccountRequest, error) {
	req := apiUpdateConfluentAccountRequest{
		ctx:       ctx,
		accountId: accountId,
		body:      &body,
	}
	return req, nil
}

// UpdateConfluentAccount Update Confluent account.
// Update the Confluent account with the provided account ID.
func (a *ConfluentCloudApi) UpdateConfluentAccount(ctx _context.Context, accountId string, body ConfluentAccountUpdateRequest) (ConfluentAccountResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateConfluentAccountRequest(ctx, accountId, body)
	if err != nil {
		var localVarReturnValue ConfluentAccountResponse
		return localVarReturnValue, nil, err
	}

	return a.updateConfluentAccountExecute(req)
}

// updateConfluentAccountExecute executes the request.
func (a *ConfluentCloudApi) updateConfluentAccountExecute(r apiUpdateConfluentAccountRequest) (ConfluentAccountResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue ConfluentAccountResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.UpdateConfluentAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}"
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

type apiUpdateConfluentResourceRequest struct {
	ctx        _context.Context
	accountId  string
	resourceId string
	body       *ConfluentResourceRequest
}

func (a *ConfluentCloudApi) buildUpdateConfluentResourceRequest(ctx _context.Context, accountId string, resourceId string, body ConfluentResourceRequest) (apiUpdateConfluentResourceRequest, error) {
	req := apiUpdateConfluentResourceRequest{
		ctx:        ctx,
		accountId:  accountId,
		resourceId: resourceId,
		body:       &body,
	}
	return req, nil
}

// UpdateConfluentResource Update resource in Confluent account.
// Update a Confluent resource with the provided resource id for the account associated with the provided account ID.
func (a *ConfluentCloudApi) UpdateConfluentResource(ctx _context.Context, accountId string, resourceId string, body ConfluentResourceRequest) (ConfluentResourceResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateConfluentResourceRequest(ctx, accountId, resourceId, body)
	if err != nil {
		var localVarReturnValue ConfluentResourceResponse
		return localVarReturnValue, nil, err
	}

	return a.updateConfluentResourceExecute(req)
}

// updateConfluentResourceExecute executes the request.
func (a *ConfluentCloudApi) updateConfluentResourceExecute(r apiUpdateConfluentResourceRequest) (ConfluentResourceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue ConfluentResourceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ConfluentCloudApi.UpdateConfluentResource")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integrations/confluent-cloud/accounts/{account_id}/resources/{resource_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"account_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.accountId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"resource_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.resourceId, "")), -1)

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

// NewConfluentCloudApi Returns NewConfluentCloudApi.
func NewConfluentCloudApi(client *datadog.APIClient) *ConfluentCloudApi {
	return &ConfluentCloudApi{
		Client: client,
	}
}
