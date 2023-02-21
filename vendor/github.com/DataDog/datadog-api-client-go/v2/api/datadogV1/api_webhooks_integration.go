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

// WebhooksIntegrationApi service type
type WebhooksIntegrationApi datadog.Service

type apiCreateWebhooksIntegrationRequest struct {
	ctx  _context.Context
	body *WebhooksIntegration
}

func (a *WebhooksIntegrationApi) buildCreateWebhooksIntegrationRequest(ctx _context.Context, body WebhooksIntegration) (apiCreateWebhooksIntegrationRequest, error) {
	req := apiCreateWebhooksIntegrationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateWebhooksIntegration Create a webhooks integration.
// Creates an endpoint with the name `<WEBHOOK_NAME>`.
func (a *WebhooksIntegrationApi) CreateWebhooksIntegration(ctx _context.Context, body WebhooksIntegration) (WebhooksIntegration, *_nethttp.Response, error) {
	req, err := a.buildCreateWebhooksIntegrationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue WebhooksIntegration
		return localVarReturnValue, nil, err
	}

	return a.createWebhooksIntegrationExecute(req)
}

// createWebhooksIntegrationExecute executes the request.
func (a *WebhooksIntegrationApi) createWebhooksIntegrationExecute(r apiCreateWebhooksIntegrationRequest) (WebhooksIntegration, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegration
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.CreateWebhooksIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/webhooks"

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

type apiCreateWebhooksIntegrationCustomVariableRequest struct {
	ctx  _context.Context
	body *WebhooksIntegrationCustomVariable
}

func (a *WebhooksIntegrationApi) buildCreateWebhooksIntegrationCustomVariableRequest(ctx _context.Context, body WebhooksIntegrationCustomVariable) (apiCreateWebhooksIntegrationCustomVariableRequest, error) {
	req := apiCreateWebhooksIntegrationCustomVariableRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateWebhooksIntegrationCustomVariable Create a custom variable.
// Creates an endpoint with the name `<CUSTOM_VARIABLE_NAME>`.
func (a *WebhooksIntegrationApi) CreateWebhooksIntegrationCustomVariable(ctx _context.Context, body WebhooksIntegrationCustomVariable) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateWebhooksIntegrationCustomVariableRequest(ctx, body)
	if err != nil {
		var localVarReturnValue WebhooksIntegrationCustomVariableResponse
		return localVarReturnValue, nil, err
	}

	return a.createWebhooksIntegrationCustomVariableExecute(req)
}

// createWebhooksIntegrationCustomVariableExecute executes the request.
func (a *WebhooksIntegrationApi) createWebhooksIntegrationCustomVariableExecute(r apiCreateWebhooksIntegrationCustomVariableRequest) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegrationCustomVariableResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.CreateWebhooksIntegrationCustomVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/custom-variables"

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

type apiDeleteWebhooksIntegrationRequest struct {
	ctx         _context.Context
	webhookName string
}

func (a *WebhooksIntegrationApi) buildDeleteWebhooksIntegrationRequest(ctx _context.Context, webhookName string) (apiDeleteWebhooksIntegrationRequest, error) {
	req := apiDeleteWebhooksIntegrationRequest{
		ctx:         ctx,
		webhookName: webhookName,
	}
	return req, nil
}

// DeleteWebhooksIntegration Delete a webhook.
// Deletes the endpoint with the name `<WEBHOOK NAME>`.
func (a *WebhooksIntegrationApi) DeleteWebhooksIntegration(ctx _context.Context, webhookName string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteWebhooksIntegrationRequest(ctx, webhookName)
	if err != nil {
		return nil, err
	}

	return a.deleteWebhooksIntegrationExecute(req)
}

// deleteWebhooksIntegrationExecute executes the request.
func (a *WebhooksIntegrationApi) deleteWebhooksIntegrationExecute(r apiDeleteWebhooksIntegrationRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.DeleteWebhooksIntegration")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/webhooks/{webhook_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"webhook_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.webhookName, "")), -1)

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

type apiDeleteWebhooksIntegrationCustomVariableRequest struct {
	ctx                _context.Context
	customVariableName string
}

func (a *WebhooksIntegrationApi) buildDeleteWebhooksIntegrationCustomVariableRequest(ctx _context.Context, customVariableName string) (apiDeleteWebhooksIntegrationCustomVariableRequest, error) {
	req := apiDeleteWebhooksIntegrationCustomVariableRequest{
		ctx:                ctx,
		customVariableName: customVariableName,
	}
	return req, nil
}

// DeleteWebhooksIntegrationCustomVariable Delete a custom variable.
// Deletes the endpoint with the name `<CUSTOM_VARIABLE_NAME>`.
func (a *WebhooksIntegrationApi) DeleteWebhooksIntegrationCustomVariable(ctx _context.Context, customVariableName string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteWebhooksIntegrationCustomVariableRequest(ctx, customVariableName)
	if err != nil {
		return nil, err
	}

	return a.deleteWebhooksIntegrationCustomVariableExecute(req)
}

// deleteWebhooksIntegrationCustomVariableExecute executes the request.
func (a *WebhooksIntegrationApi) deleteWebhooksIntegrationCustomVariableExecute(r apiDeleteWebhooksIntegrationCustomVariableRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.DeleteWebhooksIntegrationCustomVariable")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/custom-variables/{custom_variable_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"custom_variable_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.customVariableName, "")), -1)

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

type apiGetWebhooksIntegrationRequest struct {
	ctx         _context.Context
	webhookName string
}

func (a *WebhooksIntegrationApi) buildGetWebhooksIntegrationRequest(ctx _context.Context, webhookName string) (apiGetWebhooksIntegrationRequest, error) {
	req := apiGetWebhooksIntegrationRequest{
		ctx:         ctx,
		webhookName: webhookName,
	}
	return req, nil
}

// GetWebhooksIntegration Get a webhook integration.
// Gets the content of the webhook with the name `<WEBHOOK_NAME>`.
func (a *WebhooksIntegrationApi) GetWebhooksIntegration(ctx _context.Context, webhookName string) (WebhooksIntegration, *_nethttp.Response, error) {
	req, err := a.buildGetWebhooksIntegrationRequest(ctx, webhookName)
	if err != nil {
		var localVarReturnValue WebhooksIntegration
		return localVarReturnValue, nil, err
	}

	return a.getWebhooksIntegrationExecute(req)
}

// getWebhooksIntegrationExecute executes the request.
func (a *WebhooksIntegrationApi) getWebhooksIntegrationExecute(r apiGetWebhooksIntegrationRequest) (WebhooksIntegration, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegration
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.GetWebhooksIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/webhooks/{webhook_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"webhook_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.webhookName, "")), -1)

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

type apiGetWebhooksIntegrationCustomVariableRequest struct {
	ctx                _context.Context
	customVariableName string
}

func (a *WebhooksIntegrationApi) buildGetWebhooksIntegrationCustomVariableRequest(ctx _context.Context, customVariableName string) (apiGetWebhooksIntegrationCustomVariableRequest, error) {
	req := apiGetWebhooksIntegrationCustomVariableRequest{
		ctx:                ctx,
		customVariableName: customVariableName,
	}
	return req, nil
}

// GetWebhooksIntegrationCustomVariable Get a custom variable.
// Shows the content of the custom variable with the name `<CUSTOM_VARIABLE_NAME>`.
//
// If the custom variable is secret, the value does not return in the
// response payload.
func (a *WebhooksIntegrationApi) GetWebhooksIntegrationCustomVariable(ctx _context.Context, customVariableName string) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	req, err := a.buildGetWebhooksIntegrationCustomVariableRequest(ctx, customVariableName)
	if err != nil {
		var localVarReturnValue WebhooksIntegrationCustomVariableResponse
		return localVarReturnValue, nil, err
	}

	return a.getWebhooksIntegrationCustomVariableExecute(req)
}

// getWebhooksIntegrationCustomVariableExecute executes the request.
func (a *WebhooksIntegrationApi) getWebhooksIntegrationCustomVariableExecute(r apiGetWebhooksIntegrationCustomVariableRequest) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegrationCustomVariableResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.GetWebhooksIntegrationCustomVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/custom-variables/{custom_variable_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"custom_variable_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.customVariableName, "")), -1)

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

type apiUpdateWebhooksIntegrationRequest struct {
	ctx         _context.Context
	webhookName string
	body        *WebhooksIntegrationUpdateRequest
}

func (a *WebhooksIntegrationApi) buildUpdateWebhooksIntegrationRequest(ctx _context.Context, webhookName string, body WebhooksIntegrationUpdateRequest) (apiUpdateWebhooksIntegrationRequest, error) {
	req := apiUpdateWebhooksIntegrationRequest{
		ctx:         ctx,
		webhookName: webhookName,
		body:        &body,
	}
	return req, nil
}

// UpdateWebhooksIntegration Update a webhook.
// Updates the endpoint with the name `<WEBHOOK_NAME>`.
func (a *WebhooksIntegrationApi) UpdateWebhooksIntegration(ctx _context.Context, webhookName string, body WebhooksIntegrationUpdateRequest) (WebhooksIntegration, *_nethttp.Response, error) {
	req, err := a.buildUpdateWebhooksIntegrationRequest(ctx, webhookName, body)
	if err != nil {
		var localVarReturnValue WebhooksIntegration
		return localVarReturnValue, nil, err
	}

	return a.updateWebhooksIntegrationExecute(req)
}

// updateWebhooksIntegrationExecute executes the request.
func (a *WebhooksIntegrationApi) updateWebhooksIntegrationExecute(r apiUpdateWebhooksIntegrationRequest) (WebhooksIntegration, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegration
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.UpdateWebhooksIntegration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/webhooks/{webhook_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"webhook_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.webhookName, "")), -1)

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

type apiUpdateWebhooksIntegrationCustomVariableRequest struct {
	ctx                _context.Context
	customVariableName string
	body               *WebhooksIntegrationCustomVariableUpdateRequest
}

func (a *WebhooksIntegrationApi) buildUpdateWebhooksIntegrationCustomVariableRequest(ctx _context.Context, customVariableName string, body WebhooksIntegrationCustomVariableUpdateRequest) (apiUpdateWebhooksIntegrationCustomVariableRequest, error) {
	req := apiUpdateWebhooksIntegrationCustomVariableRequest{
		ctx:                ctx,
		customVariableName: customVariableName,
		body:               &body,
	}
	return req, nil
}

// UpdateWebhooksIntegrationCustomVariable Update a custom variable.
// Updates the endpoint with the name `<CUSTOM_VARIABLE_NAME>`.
func (a *WebhooksIntegrationApi) UpdateWebhooksIntegrationCustomVariable(ctx _context.Context, customVariableName string, body WebhooksIntegrationCustomVariableUpdateRequest) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateWebhooksIntegrationCustomVariableRequest(ctx, customVariableName, body)
	if err != nil {
		var localVarReturnValue WebhooksIntegrationCustomVariableResponse
		return localVarReturnValue, nil, err
	}

	return a.updateWebhooksIntegrationCustomVariableExecute(req)
}

// updateWebhooksIntegrationCustomVariableExecute executes the request.
func (a *WebhooksIntegrationApi) updateWebhooksIntegrationCustomVariableExecute(r apiUpdateWebhooksIntegrationCustomVariableRequest) (WebhooksIntegrationCustomVariableResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue WebhooksIntegrationCustomVariableResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.WebhooksIntegrationApi.UpdateWebhooksIntegrationCustomVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/webhooks/configuration/custom-variables/{custom_variable_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"custom_variable_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.customVariableName, "")), -1)

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

// NewWebhooksIntegrationApi Returns NewWebhooksIntegrationApi.
func NewWebhooksIntegrationApi(client *datadog.APIClient) *WebhooksIntegrationApi {
	return &WebhooksIntegrationApi{
		Client: client,
	}
}
