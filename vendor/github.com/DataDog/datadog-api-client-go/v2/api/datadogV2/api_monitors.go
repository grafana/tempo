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

// MonitorsApi service type
type MonitorsApi datadog.Service

type apiCreateMonitorConfigPolicyRequest struct {
	ctx  _context.Context
	body *MonitorConfigPolicyCreateRequest
}

func (a *MonitorsApi) buildCreateMonitorConfigPolicyRequest(ctx _context.Context, body MonitorConfigPolicyCreateRequest) (apiCreateMonitorConfigPolicyRequest, error) {
	req := apiCreateMonitorConfigPolicyRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateMonitorConfigPolicy Create a monitor configuration policy.
// Create a monitor configuration policy.
func (a *MonitorsApi) CreateMonitorConfigPolicy(ctx _context.Context, body MonitorConfigPolicyCreateRequest) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateMonitorConfigPolicyRequest(ctx, body)
	if err != nil {
		var localVarReturnValue MonitorConfigPolicyResponse
		return localVarReturnValue, nil, err
	}

	return a.createMonitorConfigPolicyExecute(req)
}

// createMonitorConfigPolicyExecute executes the request.
func (a *MonitorsApi) createMonitorConfigPolicyExecute(r apiCreateMonitorConfigPolicyRequest) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue MonitorConfigPolicyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MonitorsApi.CreateMonitorConfigPolicy")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/monitor/policy"

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

type apiDeleteMonitorConfigPolicyRequest struct {
	ctx      _context.Context
	policyId string
}

func (a *MonitorsApi) buildDeleteMonitorConfigPolicyRequest(ctx _context.Context, policyId string) (apiDeleteMonitorConfigPolicyRequest, error) {
	req := apiDeleteMonitorConfigPolicyRequest{
		ctx:      ctx,
		policyId: policyId,
	}
	return req, nil
}

// DeleteMonitorConfigPolicy Delete a monitor configuration policy.
// Delete a monitor configuration policy.
func (a *MonitorsApi) DeleteMonitorConfigPolicy(ctx _context.Context, policyId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteMonitorConfigPolicyRequest(ctx, policyId)
	if err != nil {
		return nil, err
	}

	return a.deleteMonitorConfigPolicyExecute(req)
}

// deleteMonitorConfigPolicyExecute executes the request.
func (a *MonitorsApi) deleteMonitorConfigPolicyExecute(r apiDeleteMonitorConfigPolicyRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MonitorsApi.DeleteMonitorConfigPolicy")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/monitor/policy/{policy_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"policy_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.policyId, "")), -1)

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

type apiGetMonitorConfigPolicyRequest struct {
	ctx      _context.Context
	policyId string
}

func (a *MonitorsApi) buildGetMonitorConfigPolicyRequest(ctx _context.Context, policyId string) (apiGetMonitorConfigPolicyRequest, error) {
	req := apiGetMonitorConfigPolicyRequest{
		ctx:      ctx,
		policyId: policyId,
	}
	return req, nil
}

// GetMonitorConfigPolicy Get a monitor configuration policy.
// Get a monitor configuration policy by `policy_id`.
func (a *MonitorsApi) GetMonitorConfigPolicy(ctx _context.Context, policyId string) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	req, err := a.buildGetMonitorConfigPolicyRequest(ctx, policyId)
	if err != nil {
		var localVarReturnValue MonitorConfigPolicyResponse
		return localVarReturnValue, nil, err
	}

	return a.getMonitorConfigPolicyExecute(req)
}

// getMonitorConfigPolicyExecute executes the request.
func (a *MonitorsApi) getMonitorConfigPolicyExecute(r apiGetMonitorConfigPolicyRequest) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MonitorConfigPolicyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MonitorsApi.GetMonitorConfigPolicy")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/monitor/policy/{policy_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"policy_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.policyId, "")), -1)

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

type apiListMonitorConfigPoliciesRequest struct {
	ctx _context.Context
}

func (a *MonitorsApi) buildListMonitorConfigPoliciesRequest(ctx _context.Context) (apiListMonitorConfigPoliciesRequest, error) {
	req := apiListMonitorConfigPoliciesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListMonitorConfigPolicies Get all monitor configuration policies.
// Get all monitor configuration policies.
func (a *MonitorsApi) ListMonitorConfigPolicies(ctx _context.Context) (MonitorConfigPolicyListResponse, *_nethttp.Response, error) {
	req, err := a.buildListMonitorConfigPoliciesRequest(ctx)
	if err != nil {
		var localVarReturnValue MonitorConfigPolicyListResponse
		return localVarReturnValue, nil, err
	}

	return a.listMonitorConfigPoliciesExecute(req)
}

// listMonitorConfigPoliciesExecute executes the request.
func (a *MonitorsApi) listMonitorConfigPoliciesExecute(r apiListMonitorConfigPoliciesRequest) (MonitorConfigPolicyListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MonitorConfigPolicyListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MonitorsApi.ListMonitorConfigPolicies")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/monitor/policy"

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

type apiUpdateMonitorConfigPolicyRequest struct {
	ctx      _context.Context
	policyId string
	body     *MonitorConfigPolicyEditRequest
}

func (a *MonitorsApi) buildUpdateMonitorConfigPolicyRequest(ctx _context.Context, policyId string, body MonitorConfigPolicyEditRequest) (apiUpdateMonitorConfigPolicyRequest, error) {
	req := apiUpdateMonitorConfigPolicyRequest{
		ctx:      ctx,
		policyId: policyId,
		body:     &body,
	}
	return req, nil
}

// UpdateMonitorConfigPolicy Edit a monitor configuration policy.
// Edit a monitor configuration policy.
func (a *MonitorsApi) UpdateMonitorConfigPolicy(ctx _context.Context, policyId string, body MonitorConfigPolicyEditRequest) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateMonitorConfigPolicyRequest(ctx, policyId, body)
	if err != nil {
		var localVarReturnValue MonitorConfigPolicyResponse
		return localVarReturnValue, nil, err
	}

	return a.updateMonitorConfigPolicyExecute(req)
}

// updateMonitorConfigPolicyExecute executes the request.
func (a *MonitorsApi) updateMonitorConfigPolicyExecute(r apiUpdateMonitorConfigPolicyRequest) (MonitorConfigPolicyResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue MonitorConfigPolicyResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MonitorsApi.UpdateMonitorConfigPolicy")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/monitor/policy/{policy_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"policy_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.policyId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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

// NewMonitorsApi Returns NewMonitorsApi.
func NewMonitorsApi(client *datadog.APIClient) *MonitorsApi {
	return &MonitorsApi{
		Client: client,
	}
}
