// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"os"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// CloudWorkloadSecurityApi service type
type CloudWorkloadSecurityApi datadog.Service

type apiCreateCloudWorkloadSecurityAgentRuleRequest struct {
	ctx  _context.Context
	body *CloudWorkloadSecurityAgentRuleCreateRequest
}

func (a *CloudWorkloadSecurityApi) buildCreateCloudWorkloadSecurityAgentRuleRequest(ctx _context.Context, body CloudWorkloadSecurityAgentRuleCreateRequest) (apiCreateCloudWorkloadSecurityAgentRuleRequest, error) {
	req := apiCreateCloudWorkloadSecurityAgentRuleRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateCloudWorkloadSecurityAgentRule Create a Cloud Workload Security Agent rule.
// Create a new Agent rule with the given parameters.
func (a *CloudWorkloadSecurityApi) CreateCloudWorkloadSecurityAgentRule(ctx _context.Context, body CloudWorkloadSecurityAgentRuleCreateRequest) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateCloudWorkloadSecurityAgentRuleRequest(ctx, body)
	if err != nil {
		var localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
		return localVarReturnValue, nil, err
	}

	return a.createCloudWorkloadSecurityAgentRuleExecute(req)
}

// createCloudWorkloadSecurityAgentRuleExecute executes the request.
func (a *CloudWorkloadSecurityApi) createCloudWorkloadSecurityAgentRuleExecute(r apiCreateCloudWorkloadSecurityAgentRuleRequest) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.CreateCloudWorkloadSecurityAgentRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security_monitoring/cloud_workload_security/agent_rules"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 409 || localVarHTTPResponse.StatusCode == 429 {
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

type apiDeleteCloudWorkloadSecurityAgentRuleRequest struct {
	ctx         _context.Context
	agentRuleId string
}

func (a *CloudWorkloadSecurityApi) buildDeleteCloudWorkloadSecurityAgentRuleRequest(ctx _context.Context, agentRuleId string) (apiDeleteCloudWorkloadSecurityAgentRuleRequest, error) {
	req := apiDeleteCloudWorkloadSecurityAgentRuleRequest{
		ctx:         ctx,
		agentRuleId: agentRuleId,
	}
	return req, nil
}

// DeleteCloudWorkloadSecurityAgentRule Delete a Cloud Workload Security Agent rule.
// Delete a specific Agent rule.
func (a *CloudWorkloadSecurityApi) DeleteCloudWorkloadSecurityAgentRule(ctx _context.Context, agentRuleId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteCloudWorkloadSecurityAgentRuleRequest(ctx, agentRuleId)
	if err != nil {
		return nil, err
	}

	return a.deleteCloudWorkloadSecurityAgentRuleExecute(req)
}

// deleteCloudWorkloadSecurityAgentRuleExecute executes the request.
func (a *CloudWorkloadSecurityApi) deleteCloudWorkloadSecurityAgentRuleExecute(r apiDeleteCloudWorkloadSecurityAgentRuleRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.DeleteCloudWorkloadSecurityAgentRule")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security_monitoring/cloud_workload_security/agent_rules/{agent_rule_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"agent_rule_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.agentRuleId, "")), -1)

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

type apiDownloadCloudWorkloadPolicyFileRequest struct {
	ctx _context.Context
}

func (a *CloudWorkloadSecurityApi) buildDownloadCloudWorkloadPolicyFileRequest(ctx _context.Context) (apiDownloadCloudWorkloadPolicyFileRequest, error) {
	req := apiDownloadCloudWorkloadPolicyFileRequest{
		ctx: ctx,
	}
	return req, nil
}

// DownloadCloudWorkloadPolicyFile Get the latest Cloud Workload Security policy.
// The download endpoint generates a Cloud Workload Security policy file from your currently active
// Cloud Workload Security rules, and downloads them as a .policy file. This file can then be deployed to
// your Agents to update the policy running in your environment.
func (a *CloudWorkloadSecurityApi) DownloadCloudWorkloadPolicyFile(ctx _context.Context) (*os.File, *_nethttp.Response, error) {
	req, err := a.buildDownloadCloudWorkloadPolicyFileRequest(ctx)
	if err != nil {
		var localVarReturnValue *os.File
		return localVarReturnValue, nil, err
	}

	return a.downloadCloudWorkloadPolicyFileExecute(req)
}

// downloadCloudWorkloadPolicyFileExecute executes the request.
func (a *CloudWorkloadSecurityApi) downloadCloudWorkloadPolicyFileExecute(r apiDownloadCloudWorkloadPolicyFileRequest) (*os.File, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue *os.File
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.DownloadCloudWorkloadPolicyFile")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security/cloud_workload/policy/download"

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

type apiGetCloudWorkloadSecurityAgentRuleRequest struct {
	ctx         _context.Context
	agentRuleId string
}

func (a *CloudWorkloadSecurityApi) buildGetCloudWorkloadSecurityAgentRuleRequest(ctx _context.Context, agentRuleId string) (apiGetCloudWorkloadSecurityAgentRuleRequest, error) {
	req := apiGetCloudWorkloadSecurityAgentRuleRequest{
		ctx:         ctx,
		agentRuleId: agentRuleId,
	}
	return req, nil
}

// GetCloudWorkloadSecurityAgentRule Get a Cloud Workload Security Agent rule.
// Get the details of a specific Agent rule.
func (a *CloudWorkloadSecurityApi) GetCloudWorkloadSecurityAgentRule(ctx _context.Context, agentRuleId string) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	req, err := a.buildGetCloudWorkloadSecurityAgentRuleRequest(ctx, agentRuleId)
	if err != nil {
		var localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
		return localVarReturnValue, nil, err
	}

	return a.getCloudWorkloadSecurityAgentRuleExecute(req)
}

// getCloudWorkloadSecurityAgentRuleExecute executes the request.
func (a *CloudWorkloadSecurityApi) getCloudWorkloadSecurityAgentRuleExecute(r apiGetCloudWorkloadSecurityAgentRuleRequest) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.GetCloudWorkloadSecurityAgentRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security_monitoring/cloud_workload_security/agent_rules/{agent_rule_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"agent_rule_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.agentRuleId, "")), -1)

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

type apiListCloudWorkloadSecurityAgentRulesRequest struct {
	ctx _context.Context
}

func (a *CloudWorkloadSecurityApi) buildListCloudWorkloadSecurityAgentRulesRequest(ctx _context.Context) (apiListCloudWorkloadSecurityAgentRulesRequest, error) {
	req := apiListCloudWorkloadSecurityAgentRulesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListCloudWorkloadSecurityAgentRules Get all Cloud Workload Security Agent rules.
// Get the list of Agent rules.
func (a *CloudWorkloadSecurityApi) ListCloudWorkloadSecurityAgentRules(ctx _context.Context) (CloudWorkloadSecurityAgentRulesListResponse, *_nethttp.Response, error) {
	req, err := a.buildListCloudWorkloadSecurityAgentRulesRequest(ctx)
	if err != nil {
		var localVarReturnValue CloudWorkloadSecurityAgentRulesListResponse
		return localVarReturnValue, nil, err
	}

	return a.listCloudWorkloadSecurityAgentRulesExecute(req)
}

// listCloudWorkloadSecurityAgentRulesExecute executes the request.
func (a *CloudWorkloadSecurityApi) listCloudWorkloadSecurityAgentRulesExecute(r apiListCloudWorkloadSecurityAgentRulesRequest) (CloudWorkloadSecurityAgentRulesListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CloudWorkloadSecurityAgentRulesListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.ListCloudWorkloadSecurityAgentRules")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security_monitoring/cloud_workload_security/agent_rules"

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

type apiUpdateCloudWorkloadSecurityAgentRuleRequest struct {
	ctx         _context.Context
	agentRuleId string
	body        *CloudWorkloadSecurityAgentRuleUpdateRequest
}

func (a *CloudWorkloadSecurityApi) buildUpdateCloudWorkloadSecurityAgentRuleRequest(ctx _context.Context, agentRuleId string, body CloudWorkloadSecurityAgentRuleUpdateRequest) (apiUpdateCloudWorkloadSecurityAgentRuleRequest, error) {
	req := apiUpdateCloudWorkloadSecurityAgentRuleRequest{
		ctx:         ctx,
		agentRuleId: agentRuleId,
		body:        &body,
	}
	return req, nil
}

// UpdateCloudWorkloadSecurityAgentRule Update a Cloud Workload Security Agent rule.
// Update a specific Agent rule.
// Returns the Agent rule object when the request is successful.
func (a *CloudWorkloadSecurityApi) UpdateCloudWorkloadSecurityAgentRule(ctx _context.Context, agentRuleId string, body CloudWorkloadSecurityAgentRuleUpdateRequest) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateCloudWorkloadSecurityAgentRuleRequest(ctx, agentRuleId, body)
	if err != nil {
		var localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
		return localVarReturnValue, nil, err
	}

	return a.updateCloudWorkloadSecurityAgentRuleExecute(req)
}

// updateCloudWorkloadSecurityAgentRuleExecute executes the request.
func (a *CloudWorkloadSecurityApi) updateCloudWorkloadSecurityAgentRuleExecute(r apiUpdateCloudWorkloadSecurityAgentRuleRequest) (CloudWorkloadSecurityAgentRuleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue CloudWorkloadSecurityAgentRuleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.CloudWorkloadSecurityApi.UpdateCloudWorkloadSecurityAgentRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/security_monitoring/cloud_workload_security/agent_rules/{agent_rule_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"agent_rule_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.agentRuleId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 409 || localVarHTTPResponse.StatusCode == 429 {
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

// NewCloudWorkloadSecurityApi Returns NewCloudWorkloadSecurityApi.
func NewCloudWorkloadSecurityApi(client *datadog.APIClient) *CloudWorkloadSecurityApi {
	return &CloudWorkloadSecurityApi{
		Client: client,
	}
}
