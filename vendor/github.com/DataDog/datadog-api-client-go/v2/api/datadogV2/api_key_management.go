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

// KeyManagementApi service type
type KeyManagementApi datadog.Service

type apiCreateAPIKeyRequest struct {
	ctx  _context.Context
	body *APIKeyCreateRequest
}

func (a *KeyManagementApi) buildCreateAPIKeyRequest(ctx _context.Context, body APIKeyCreateRequest) (apiCreateAPIKeyRequest, error) {
	req := apiCreateAPIKeyRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateAPIKey Create an API key.
// Create an API key.
func (a *KeyManagementApi) CreateAPIKey(ctx _context.Context, body APIKeyCreateRequest) (APIKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateAPIKeyRequest(ctx, body)
	if err != nil {
		var localVarReturnValue APIKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.createAPIKeyExecute(req)
}

// createAPIKeyExecute executes the request.
func (a *KeyManagementApi) createAPIKeyExecute(r apiCreateAPIKeyRequest) (APIKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue APIKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.CreateAPIKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/api_keys"

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

type apiCreateCurrentUserApplicationKeyRequest struct {
	ctx  _context.Context
	body *ApplicationKeyCreateRequest
}

func (a *KeyManagementApi) buildCreateCurrentUserApplicationKeyRequest(ctx _context.Context, body ApplicationKeyCreateRequest) (apiCreateCurrentUserApplicationKeyRequest, error) {
	req := apiCreateCurrentUserApplicationKeyRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateCurrentUserApplicationKey Create an application key for current user.
// Create an application key for current user
func (a *KeyManagementApi) CreateCurrentUserApplicationKey(ctx _context.Context, body ApplicationKeyCreateRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateCurrentUserApplicationKeyRequest(ctx, body)
	if err != nil {
		var localVarReturnValue ApplicationKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.createCurrentUserApplicationKeyExecute(req)
}

// createCurrentUserApplicationKeyExecute executes the request.
func (a *KeyManagementApi) createCurrentUserApplicationKeyExecute(r apiCreateCurrentUserApplicationKeyRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue ApplicationKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.CreateCurrentUserApplicationKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/current_user/application_keys"

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

type apiDeleteAPIKeyRequest struct {
	ctx      _context.Context
	apiKeyId string
}

func (a *KeyManagementApi) buildDeleteAPIKeyRequest(ctx _context.Context, apiKeyId string) (apiDeleteAPIKeyRequest, error) {
	req := apiDeleteAPIKeyRequest{
		ctx:      ctx,
		apiKeyId: apiKeyId,
	}
	return req, nil
}

// DeleteAPIKey Delete an API key.
// Delete an API key.
func (a *KeyManagementApi) DeleteAPIKey(ctx _context.Context, apiKeyId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteAPIKeyRequest(ctx, apiKeyId)
	if err != nil {
		return nil, err
	}

	return a.deleteAPIKeyExecute(req)
}

// deleteAPIKeyExecute executes the request.
func (a *KeyManagementApi) deleteAPIKeyExecute(r apiDeleteAPIKeyRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.DeleteAPIKey")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/api_keys/{api_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"api_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.apiKeyId, "")), -1)

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

type apiDeleteApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
}

func (a *KeyManagementApi) buildDeleteApplicationKeyRequest(ctx _context.Context, appKeyId string) (apiDeleteApplicationKeyRequest, error) {
	req := apiDeleteApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
	}
	return req, nil
}

// DeleteApplicationKey Delete an application key.
// Delete an application key
func (a *KeyManagementApi) DeleteApplicationKey(ctx _context.Context, appKeyId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteApplicationKeyRequest(ctx, appKeyId)
	if err != nil {
		return nil, err
	}

	return a.deleteApplicationKeyExecute(req)
}

// deleteApplicationKeyExecute executes the request.
func (a *KeyManagementApi) deleteApplicationKeyExecute(r apiDeleteApplicationKeyRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.DeleteApplicationKey")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

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

type apiDeleteCurrentUserApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
}

func (a *KeyManagementApi) buildDeleteCurrentUserApplicationKeyRequest(ctx _context.Context, appKeyId string) (apiDeleteCurrentUserApplicationKeyRequest, error) {
	req := apiDeleteCurrentUserApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
	}
	return req, nil
}

// DeleteCurrentUserApplicationKey Delete an application key owned by current user.
// Delete an application key owned by current user
func (a *KeyManagementApi) DeleteCurrentUserApplicationKey(ctx _context.Context, appKeyId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteCurrentUserApplicationKeyRequest(ctx, appKeyId)
	if err != nil {
		return nil, err
	}

	return a.deleteCurrentUserApplicationKeyExecute(req)
}

// deleteCurrentUserApplicationKeyExecute executes the request.
func (a *KeyManagementApi) deleteCurrentUserApplicationKeyExecute(r apiDeleteCurrentUserApplicationKeyRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.DeleteCurrentUserApplicationKey")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/current_user/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

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

type apiGetAPIKeyRequest struct {
	ctx      _context.Context
	apiKeyId string
	include  *string
}

// GetAPIKeyOptionalParameters holds optional parameters for GetAPIKey.
type GetAPIKeyOptionalParameters struct {
	Include *string
}

// NewGetAPIKeyOptionalParameters creates an empty struct for parameters.
func NewGetAPIKeyOptionalParameters() *GetAPIKeyOptionalParameters {
	this := GetAPIKeyOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *GetAPIKeyOptionalParameters) WithInclude(include string) *GetAPIKeyOptionalParameters {
	r.Include = &include
	return r
}

func (a *KeyManagementApi) buildGetAPIKeyRequest(ctx _context.Context, apiKeyId string, o ...GetAPIKeyOptionalParameters) (apiGetAPIKeyRequest, error) {
	req := apiGetAPIKeyRequest{
		ctx:      ctx,
		apiKeyId: apiKeyId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetAPIKeyOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// GetAPIKey Get API key.
// Get an API key.
func (a *KeyManagementApi) GetAPIKey(ctx _context.Context, apiKeyId string, o ...GetAPIKeyOptionalParameters) (APIKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildGetAPIKeyRequest(ctx, apiKeyId, o...)
	if err != nil {
		var localVarReturnValue APIKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.getAPIKeyExecute(req)
}

// getAPIKeyExecute executes the request.
func (a *KeyManagementApi) getAPIKeyExecute(r apiGetAPIKeyRequest) (APIKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue APIKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.GetAPIKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/api_keys/{api_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"api_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.apiKeyId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, ""))
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

type apiGetApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
	include  *string
}

// GetApplicationKeyOptionalParameters holds optional parameters for GetApplicationKey.
type GetApplicationKeyOptionalParameters struct {
	Include *string
}

// NewGetApplicationKeyOptionalParameters creates an empty struct for parameters.
func NewGetApplicationKeyOptionalParameters() *GetApplicationKeyOptionalParameters {
	this := GetApplicationKeyOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *GetApplicationKeyOptionalParameters) WithInclude(include string) *GetApplicationKeyOptionalParameters {
	r.Include = &include
	return r
}

func (a *KeyManagementApi) buildGetApplicationKeyRequest(ctx _context.Context, appKeyId string, o ...GetApplicationKeyOptionalParameters) (apiGetApplicationKeyRequest, error) {
	req := apiGetApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetApplicationKeyOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// GetApplicationKey Get an application key.
// Get an application key for your org.
func (a *KeyManagementApi) GetApplicationKey(ctx _context.Context, appKeyId string, o ...GetApplicationKeyOptionalParameters) (ApplicationKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildGetApplicationKeyRequest(ctx, appKeyId, o...)
	if err != nil {
		var localVarReturnValue ApplicationKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.getApplicationKeyExecute(req)
}

// getApplicationKeyExecute executes the request.
func (a *KeyManagementApi) getApplicationKeyExecute(r apiGetApplicationKeyRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ApplicationKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.GetApplicationKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, ""))
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

type apiGetCurrentUserApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
}

func (a *KeyManagementApi) buildGetCurrentUserApplicationKeyRequest(ctx _context.Context, appKeyId string) (apiGetCurrentUserApplicationKeyRequest, error) {
	req := apiGetCurrentUserApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
	}
	return req, nil
}

// GetCurrentUserApplicationKey Get one application key owned by current user.
// Get an application key owned by current user
func (a *KeyManagementApi) GetCurrentUserApplicationKey(ctx _context.Context, appKeyId string) (ApplicationKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildGetCurrentUserApplicationKeyRequest(ctx, appKeyId)
	if err != nil {
		var localVarReturnValue ApplicationKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.getCurrentUserApplicationKeyExecute(req)
}

// getCurrentUserApplicationKeyExecute executes the request.
func (a *KeyManagementApi) getCurrentUserApplicationKeyExecute(r apiGetCurrentUserApplicationKeyRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ApplicationKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.GetCurrentUserApplicationKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/current_user/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

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

type apiListAPIKeysRequest struct {
	ctx                   _context.Context
	pageSize              *int64
	pageNumber            *int64
	sort                  *APIKeysSort
	filter                *string
	filterCreatedAtStart  *string
	filterCreatedAtEnd    *string
	filterModifiedAtStart *string
	filterModifiedAtEnd   *string
	include               *string
}

// ListAPIKeysOptionalParameters holds optional parameters for ListAPIKeys.
type ListAPIKeysOptionalParameters struct {
	PageSize              *int64
	PageNumber            *int64
	Sort                  *APIKeysSort
	Filter                *string
	FilterCreatedAtStart  *string
	FilterCreatedAtEnd    *string
	FilterModifiedAtStart *string
	FilterModifiedAtEnd   *string
	Include               *string
}

// NewListAPIKeysOptionalParameters creates an empty struct for parameters.
func NewListAPIKeysOptionalParameters() *ListAPIKeysOptionalParameters {
	this := ListAPIKeysOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithPageSize(pageSize int64) *ListAPIKeysOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithPageNumber(pageNumber int64) *ListAPIKeysOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithSort(sort APIKeysSort) *ListAPIKeysOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithFilter(filter string) *ListAPIKeysOptionalParameters {
	r.Filter = &filter
	return r
}

// WithFilterCreatedAtStart sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithFilterCreatedAtStart(filterCreatedAtStart string) *ListAPIKeysOptionalParameters {
	r.FilterCreatedAtStart = &filterCreatedAtStart
	return r
}

// WithFilterCreatedAtEnd sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithFilterCreatedAtEnd(filterCreatedAtEnd string) *ListAPIKeysOptionalParameters {
	r.FilterCreatedAtEnd = &filterCreatedAtEnd
	return r
}

// WithFilterModifiedAtStart sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithFilterModifiedAtStart(filterModifiedAtStart string) *ListAPIKeysOptionalParameters {
	r.FilterModifiedAtStart = &filterModifiedAtStart
	return r
}

// WithFilterModifiedAtEnd sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithFilterModifiedAtEnd(filterModifiedAtEnd string) *ListAPIKeysOptionalParameters {
	r.FilterModifiedAtEnd = &filterModifiedAtEnd
	return r
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *ListAPIKeysOptionalParameters) WithInclude(include string) *ListAPIKeysOptionalParameters {
	r.Include = &include
	return r
}

func (a *KeyManagementApi) buildListAPIKeysRequest(ctx _context.Context, o ...ListAPIKeysOptionalParameters) (apiListAPIKeysRequest, error) {
	req := apiListAPIKeysRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListAPIKeysOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
		req.filterCreatedAtStart = o[0].FilterCreatedAtStart
		req.filterCreatedAtEnd = o[0].FilterCreatedAtEnd
		req.filterModifiedAtStart = o[0].FilterModifiedAtStart
		req.filterModifiedAtEnd = o[0].FilterModifiedAtEnd
		req.include = o[0].Include
	}
	return req, nil
}

// ListAPIKeys Get all API keys.
// List all API keys available for your account.
func (a *KeyManagementApi) ListAPIKeys(ctx _context.Context, o ...ListAPIKeysOptionalParameters) (APIKeysResponse, *_nethttp.Response, error) {
	req, err := a.buildListAPIKeysRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue APIKeysResponse
		return localVarReturnValue, nil, err
	}

	return a.listAPIKeysExecute(req)
}

// listAPIKeysExecute executes the request.
func (a *KeyManagementApi) listAPIKeysExecute(r apiListAPIKeysRequest) (APIKeysResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue APIKeysResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.ListAPIKeys")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/api_keys"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*r.pageNumber, ""))
	}
	if r.sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*r.sort, ""))
	}
	if r.filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*r.filter, ""))
	}
	if r.filterCreatedAtStart != nil {
		localVarQueryParams.Add("filter[created_at][start]", datadog.ParameterToString(*r.filterCreatedAtStart, ""))
	}
	if r.filterCreatedAtEnd != nil {
		localVarQueryParams.Add("filter[created_at][end]", datadog.ParameterToString(*r.filterCreatedAtEnd, ""))
	}
	if r.filterModifiedAtStart != nil {
		localVarQueryParams.Add("filter[modified_at][start]", datadog.ParameterToString(*r.filterModifiedAtStart, ""))
	}
	if r.filterModifiedAtEnd != nil {
		localVarQueryParams.Add("filter[modified_at][end]", datadog.ParameterToString(*r.filterModifiedAtEnd, ""))
	}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, ""))
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

type apiListApplicationKeysRequest struct {
	ctx                  _context.Context
	pageSize             *int64
	pageNumber           *int64
	sort                 *ApplicationKeysSort
	filter               *string
	filterCreatedAtStart *string
	filterCreatedAtEnd   *string
}

// ListApplicationKeysOptionalParameters holds optional parameters for ListApplicationKeys.
type ListApplicationKeysOptionalParameters struct {
	PageSize             *int64
	PageNumber           *int64
	Sort                 *ApplicationKeysSort
	Filter               *string
	FilterCreatedAtStart *string
	FilterCreatedAtEnd   *string
}

// NewListApplicationKeysOptionalParameters creates an empty struct for parameters.
func NewListApplicationKeysOptionalParameters() *ListApplicationKeysOptionalParameters {
	this := ListApplicationKeysOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithPageSize(pageSize int64) *ListApplicationKeysOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithPageNumber(pageNumber int64) *ListApplicationKeysOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithSort(sort ApplicationKeysSort) *ListApplicationKeysOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithFilter(filter string) *ListApplicationKeysOptionalParameters {
	r.Filter = &filter
	return r
}

// WithFilterCreatedAtStart sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithFilterCreatedAtStart(filterCreatedAtStart string) *ListApplicationKeysOptionalParameters {
	r.FilterCreatedAtStart = &filterCreatedAtStart
	return r
}

// WithFilterCreatedAtEnd sets the corresponding parameter name and returns the struct.
func (r *ListApplicationKeysOptionalParameters) WithFilterCreatedAtEnd(filterCreatedAtEnd string) *ListApplicationKeysOptionalParameters {
	r.FilterCreatedAtEnd = &filterCreatedAtEnd
	return r
}

func (a *KeyManagementApi) buildListApplicationKeysRequest(ctx _context.Context, o ...ListApplicationKeysOptionalParameters) (apiListApplicationKeysRequest, error) {
	req := apiListApplicationKeysRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListApplicationKeysOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
		req.filterCreatedAtStart = o[0].FilterCreatedAtStart
		req.filterCreatedAtEnd = o[0].FilterCreatedAtEnd
	}
	return req, nil
}

// ListApplicationKeys Get all application keys.
// List all application keys available for your org
func (a *KeyManagementApi) ListApplicationKeys(ctx _context.Context, o ...ListApplicationKeysOptionalParameters) (ListApplicationKeysResponse, *_nethttp.Response, error) {
	req, err := a.buildListApplicationKeysRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue ListApplicationKeysResponse
		return localVarReturnValue, nil, err
	}

	return a.listApplicationKeysExecute(req)
}

// listApplicationKeysExecute executes the request.
func (a *KeyManagementApi) listApplicationKeysExecute(r apiListApplicationKeysRequest) (ListApplicationKeysResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ListApplicationKeysResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.ListApplicationKeys")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/application_keys"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*r.pageNumber, ""))
	}
	if r.sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*r.sort, ""))
	}
	if r.filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*r.filter, ""))
	}
	if r.filterCreatedAtStart != nil {
		localVarQueryParams.Add("filter[created_at][start]", datadog.ParameterToString(*r.filterCreatedAtStart, ""))
	}
	if r.filterCreatedAtEnd != nil {
		localVarQueryParams.Add("filter[created_at][end]", datadog.ParameterToString(*r.filterCreatedAtEnd, ""))
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

type apiListCurrentUserApplicationKeysRequest struct {
	ctx                  _context.Context
	pageSize             *int64
	pageNumber           *int64
	sort                 *ApplicationKeysSort
	filter               *string
	filterCreatedAtStart *string
	filterCreatedAtEnd   *string
}

// ListCurrentUserApplicationKeysOptionalParameters holds optional parameters for ListCurrentUserApplicationKeys.
type ListCurrentUserApplicationKeysOptionalParameters struct {
	PageSize             *int64
	PageNumber           *int64
	Sort                 *ApplicationKeysSort
	Filter               *string
	FilterCreatedAtStart *string
	FilterCreatedAtEnd   *string
}

// NewListCurrentUserApplicationKeysOptionalParameters creates an empty struct for parameters.
func NewListCurrentUserApplicationKeysOptionalParameters() *ListCurrentUserApplicationKeysOptionalParameters {
	this := ListCurrentUserApplicationKeysOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithPageSize(pageSize int64) *ListCurrentUserApplicationKeysOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithPageNumber(pageNumber int64) *ListCurrentUserApplicationKeysOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithSort(sort ApplicationKeysSort) *ListCurrentUserApplicationKeysOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithFilter(filter string) *ListCurrentUserApplicationKeysOptionalParameters {
	r.Filter = &filter
	return r
}

// WithFilterCreatedAtStart sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithFilterCreatedAtStart(filterCreatedAtStart string) *ListCurrentUserApplicationKeysOptionalParameters {
	r.FilterCreatedAtStart = &filterCreatedAtStart
	return r
}

// WithFilterCreatedAtEnd sets the corresponding parameter name and returns the struct.
func (r *ListCurrentUserApplicationKeysOptionalParameters) WithFilterCreatedAtEnd(filterCreatedAtEnd string) *ListCurrentUserApplicationKeysOptionalParameters {
	r.FilterCreatedAtEnd = &filterCreatedAtEnd
	return r
}

func (a *KeyManagementApi) buildListCurrentUserApplicationKeysRequest(ctx _context.Context, o ...ListCurrentUserApplicationKeysOptionalParameters) (apiListCurrentUserApplicationKeysRequest, error) {
	req := apiListCurrentUserApplicationKeysRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListCurrentUserApplicationKeysOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
		req.filterCreatedAtStart = o[0].FilterCreatedAtStart
		req.filterCreatedAtEnd = o[0].FilterCreatedAtEnd
	}
	return req, nil
}

// ListCurrentUserApplicationKeys Get all application keys owned by current user.
// List all application keys available for current user
func (a *KeyManagementApi) ListCurrentUserApplicationKeys(ctx _context.Context, o ...ListCurrentUserApplicationKeysOptionalParameters) (ListApplicationKeysResponse, *_nethttp.Response, error) {
	req, err := a.buildListCurrentUserApplicationKeysRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue ListApplicationKeysResponse
		return localVarReturnValue, nil, err
	}

	return a.listCurrentUserApplicationKeysExecute(req)
}

// listCurrentUserApplicationKeysExecute executes the request.
func (a *KeyManagementApi) listCurrentUserApplicationKeysExecute(r apiListCurrentUserApplicationKeysRequest) (ListApplicationKeysResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ListApplicationKeysResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.ListCurrentUserApplicationKeys")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/current_user/application_keys"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*r.pageNumber, ""))
	}
	if r.sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*r.sort, ""))
	}
	if r.filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*r.filter, ""))
	}
	if r.filterCreatedAtStart != nil {
		localVarQueryParams.Add("filter[created_at][start]", datadog.ParameterToString(*r.filterCreatedAtStart, ""))
	}
	if r.filterCreatedAtEnd != nil {
		localVarQueryParams.Add("filter[created_at][end]", datadog.ParameterToString(*r.filterCreatedAtEnd, ""))
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

type apiUpdateAPIKeyRequest struct {
	ctx      _context.Context
	apiKeyId string
	body     *APIKeyUpdateRequest
}

func (a *KeyManagementApi) buildUpdateAPIKeyRequest(ctx _context.Context, apiKeyId string, body APIKeyUpdateRequest) (apiUpdateAPIKeyRequest, error) {
	req := apiUpdateAPIKeyRequest{
		ctx:      ctx,
		apiKeyId: apiKeyId,
		body:     &body,
	}
	return req, nil
}

// UpdateAPIKey Edit an API key.
// Update an API key.
func (a *KeyManagementApi) UpdateAPIKey(ctx _context.Context, apiKeyId string, body APIKeyUpdateRequest) (APIKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateAPIKeyRequest(ctx, apiKeyId, body)
	if err != nil {
		var localVarReturnValue APIKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.updateAPIKeyExecute(req)
}

// updateAPIKeyExecute executes the request.
func (a *KeyManagementApi) updateAPIKeyExecute(r apiUpdateAPIKeyRequest) (APIKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue APIKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.UpdateAPIKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/api_keys/{api_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"api_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.apiKeyId, "")), -1)

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

type apiUpdateApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
	body     *ApplicationKeyUpdateRequest
}

func (a *KeyManagementApi) buildUpdateApplicationKeyRequest(ctx _context.Context, appKeyId string, body ApplicationKeyUpdateRequest) (apiUpdateApplicationKeyRequest, error) {
	req := apiUpdateApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
		body:     &body,
	}
	return req, nil
}

// UpdateApplicationKey Edit an application key.
// Edit an application key
func (a *KeyManagementApi) UpdateApplicationKey(ctx _context.Context, appKeyId string, body ApplicationKeyUpdateRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateApplicationKeyRequest(ctx, appKeyId, body)
	if err != nil {
		var localVarReturnValue ApplicationKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.updateApplicationKeyExecute(req)
}

// updateApplicationKeyExecute executes the request.
func (a *KeyManagementApi) updateApplicationKeyExecute(r apiUpdateApplicationKeyRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue ApplicationKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.UpdateApplicationKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

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

type apiUpdateCurrentUserApplicationKeyRequest struct {
	ctx      _context.Context
	appKeyId string
	body     *ApplicationKeyUpdateRequest
}

func (a *KeyManagementApi) buildUpdateCurrentUserApplicationKeyRequest(ctx _context.Context, appKeyId string, body ApplicationKeyUpdateRequest) (apiUpdateCurrentUserApplicationKeyRequest, error) {
	req := apiUpdateCurrentUserApplicationKeyRequest{
		ctx:      ctx,
		appKeyId: appKeyId,
		body:     &body,
	}
	return req, nil
}

// UpdateCurrentUserApplicationKey Edit an application key owned by current user.
// Edit an application key owned by current user
func (a *KeyManagementApi) UpdateCurrentUserApplicationKey(ctx _context.Context, appKeyId string, body ApplicationKeyUpdateRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateCurrentUserApplicationKeyRequest(ctx, appKeyId, body)
	if err != nil {
		var localVarReturnValue ApplicationKeyResponse
		return localVarReturnValue, nil, err
	}

	return a.updateCurrentUserApplicationKeyExecute(req)
}

// updateCurrentUserApplicationKeyExecute executes the request.
func (a *KeyManagementApi) updateCurrentUserApplicationKeyExecute(r apiUpdateCurrentUserApplicationKeyRequest) (ApplicationKeyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue ApplicationKeyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.KeyManagementApi.UpdateCurrentUserApplicationKey")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/current_user/application_keys/{app_key_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"app_key_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.appKeyId, "")), -1)

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

// NewKeyManagementApi Returns NewKeyManagementApi.
func NewKeyManagementApi(client *datadog.APIClient) *KeyManagementApi {
	return &KeyManagementApi{
		Client: client,
	}
}
