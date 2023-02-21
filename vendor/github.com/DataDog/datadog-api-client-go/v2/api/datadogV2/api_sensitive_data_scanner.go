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

// SensitiveDataScannerApi service type
type SensitiveDataScannerApi datadog.Service

type apiCreateScanningGroupRequest struct {
	ctx  _context.Context
	body *SensitiveDataScannerGroupCreateRequest
}

func (a *SensitiveDataScannerApi) buildCreateScanningGroupRequest(ctx _context.Context, body SensitiveDataScannerGroupCreateRequest) (apiCreateScanningGroupRequest, error) {
	req := apiCreateScanningGroupRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateScanningGroup Create Scanning Group.
// Create a scanning group.
// The request MAY include a configuration relationship.
// A rules relationship can be omitted entirely, but if it is included it MUST be
// null or an empty array (rules cannot be created at the same time).
// The new group will be ordered last within the configuration.
func (a *SensitiveDataScannerApi) CreateScanningGroup(ctx _context.Context, body SensitiveDataScannerGroupCreateRequest) (SensitiveDataScannerCreateGroupResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateScanningGroupRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerCreateGroupResponse
		return localVarReturnValue, nil, err
	}

	return a.createScanningGroupExecute(req)
}

// createScanningGroupExecute executes the request.
func (a *SensitiveDataScannerApi) createScanningGroupExecute(r apiCreateScanningGroupRequest) (SensitiveDataScannerCreateGroupResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerCreateGroupResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.CreateScanningGroup")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/groups"

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

type apiCreateScanningRuleRequest struct {
	ctx  _context.Context
	body *SensitiveDataScannerRuleCreateRequest
}

func (a *SensitiveDataScannerApi) buildCreateScanningRuleRequest(ctx _context.Context, body SensitiveDataScannerRuleCreateRequest) (apiCreateScanningRuleRequest, error) {
	req := apiCreateScanningRuleRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateScanningRule Create Scanning Rule.
// Create a scanning rule in a sensitive data scanner group, ordered last.
// The posted rule MUST include a group relationship.
// It MUST include either a standard_pattern relationship or a regex attribute, but not both.
// If included_attributes is empty or missing, we will scan all attributes except
// excluded_attributes. If both are missing, we will scan the whole event.
func (a *SensitiveDataScannerApi) CreateScanningRule(ctx _context.Context, body SensitiveDataScannerRuleCreateRequest) (SensitiveDataScannerCreateRuleResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateScanningRuleRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerCreateRuleResponse
		return localVarReturnValue, nil, err
	}

	return a.createScanningRuleExecute(req)
}

// createScanningRuleExecute executes the request.
func (a *SensitiveDataScannerApi) createScanningRuleExecute(r apiCreateScanningRuleRequest) (SensitiveDataScannerCreateRuleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerCreateRuleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.CreateScanningRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/rules"

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

type apiDeleteScanningGroupRequest struct {
	ctx     _context.Context
	groupId string
	body    *SensitiveDataScannerGroupDeleteRequest
}

func (a *SensitiveDataScannerApi) buildDeleteScanningGroupRequest(ctx _context.Context, groupId string, body SensitiveDataScannerGroupDeleteRequest) (apiDeleteScanningGroupRequest, error) {
	req := apiDeleteScanningGroupRequest{
		ctx:     ctx,
		groupId: groupId,
		body:    &body,
	}
	return req, nil
}

// DeleteScanningGroup Delete Scanning Group.
// Delete a given group.
func (a *SensitiveDataScannerApi) DeleteScanningGroup(ctx _context.Context, groupId string, body SensitiveDataScannerGroupDeleteRequest) (SensitiveDataScannerGroupDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteScanningGroupRequest(ctx, groupId, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerGroupDeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteScanningGroupExecute(req)
}

// deleteScanningGroupExecute executes the request.
func (a *SensitiveDataScannerApi) deleteScanningGroupExecute(r apiDeleteScanningGroupRequest) (SensitiveDataScannerGroupDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerGroupDeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.DeleteScanningGroup")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/groups/{group_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"group_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.groupId, "")), -1)

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

type apiDeleteScanningRuleRequest struct {
	ctx    _context.Context
	ruleId string
	body   *SensitiveDataScannerRuleDeleteRequest
}

func (a *SensitiveDataScannerApi) buildDeleteScanningRuleRequest(ctx _context.Context, ruleId string, body SensitiveDataScannerRuleDeleteRequest) (apiDeleteScanningRuleRequest, error) {
	req := apiDeleteScanningRuleRequest{
		ctx:    ctx,
		ruleId: ruleId,
		body:   &body,
	}
	return req, nil
}

// DeleteScanningRule Delete Scanning Rule.
// Delete a given rule.
func (a *SensitiveDataScannerApi) DeleteScanningRule(ctx _context.Context, ruleId string, body SensitiveDataScannerRuleDeleteRequest) (SensitiveDataScannerRuleDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteScanningRuleRequest(ctx, ruleId, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerRuleDeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteScanningRuleExecute(req)
}

// deleteScanningRuleExecute executes the request.
func (a *SensitiveDataScannerApi) deleteScanningRuleExecute(r apiDeleteScanningRuleRequest) (SensitiveDataScannerRuleDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerRuleDeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.DeleteScanningRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/rules/{rule_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"rule_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.ruleId, "")), -1)

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

type apiListScanningGroupsRequest struct {
	ctx _context.Context
}

func (a *SensitiveDataScannerApi) buildListScanningGroupsRequest(ctx _context.Context) (apiListScanningGroupsRequest, error) {
	req := apiListScanningGroupsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListScanningGroups List Scanning Groups.
// List all the Scanning groups in your organization.
func (a *SensitiveDataScannerApi) ListScanningGroups(ctx _context.Context) (SensitiveDataScannerGetConfigResponse, *_nethttp.Response, error) {
	req, err := a.buildListScanningGroupsRequest(ctx)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerGetConfigResponse
		return localVarReturnValue, nil, err
	}

	return a.listScanningGroupsExecute(req)
}

// listScanningGroupsExecute executes the request.
func (a *SensitiveDataScannerApi) listScanningGroupsExecute(r apiListScanningGroupsRequest) (SensitiveDataScannerGetConfigResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerGetConfigResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.ListScanningGroups")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config"

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

type apiListStandardPatternsRequest struct {
	ctx _context.Context
}

func (a *SensitiveDataScannerApi) buildListStandardPatternsRequest(ctx _context.Context) (apiListStandardPatternsRequest, error) {
	req := apiListStandardPatternsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListStandardPatterns List standard patterns.
// Returns all standard patterns.
func (a *SensitiveDataScannerApi) ListStandardPatterns(ctx _context.Context) (SensitiveDataScannerStandardPatternsResponseData, *_nethttp.Response, error) {
	req, err := a.buildListStandardPatternsRequest(ctx)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerStandardPatternsResponseData
		return localVarReturnValue, nil, err
	}

	return a.listStandardPatternsExecute(req)
}

// listStandardPatternsExecute executes the request.
func (a *SensitiveDataScannerApi) listStandardPatternsExecute(r apiListStandardPatternsRequest) (SensitiveDataScannerStandardPatternsResponseData, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerStandardPatternsResponseData
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.ListStandardPatterns")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/standard-patterns"

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

type apiReorderScanningGroupsRequest struct {
	ctx  _context.Context
	body *SensitiveDataScannerConfigRequest
}

func (a *SensitiveDataScannerApi) buildReorderScanningGroupsRequest(ctx _context.Context, body SensitiveDataScannerConfigRequest) (apiReorderScanningGroupsRequest, error) {
	req := apiReorderScanningGroupsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// ReorderScanningGroups Reorder Groups.
// Reorder the list of groups.
func (a *SensitiveDataScannerApi) ReorderScanningGroups(ctx _context.Context, body SensitiveDataScannerConfigRequest) (SensitiveDataScannerReorderGroupsResponse, *_nethttp.Response, error) {
	req, err := a.buildReorderScanningGroupsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerReorderGroupsResponse
		return localVarReturnValue, nil, err
	}

	return a.reorderScanningGroupsExecute(req)
}

// reorderScanningGroupsExecute executes the request.
func (a *SensitiveDataScannerApi) reorderScanningGroupsExecute(r apiReorderScanningGroupsRequest) (SensitiveDataScannerReorderGroupsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerReorderGroupsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.ReorderScanningGroups")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config"

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

type apiUpdateScanningGroupRequest struct {
	ctx     _context.Context
	groupId string
	body    *SensitiveDataScannerGroupUpdateRequest
}

func (a *SensitiveDataScannerApi) buildUpdateScanningGroupRequest(ctx _context.Context, groupId string, body SensitiveDataScannerGroupUpdateRequest) (apiUpdateScanningGroupRequest, error) {
	req := apiUpdateScanningGroupRequest{
		ctx:     ctx,
		groupId: groupId,
		body:    &body,
	}
	return req, nil
}

// UpdateScanningGroup Update Scanning Group.
// Update a group, including the order of the rules.
// Rules within the group are reordered by including a rules relationship. If the rules
// relationship is present, its data section MUST contain linkages for all of the rules
// currently in the group, and MUST NOT contain any others.
func (a *SensitiveDataScannerApi) UpdateScanningGroup(ctx _context.Context, groupId string, body SensitiveDataScannerGroupUpdateRequest) (SensitiveDataScannerGroupUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateScanningGroupRequest(ctx, groupId, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerGroupUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.updateScanningGroupExecute(req)
}

// updateScanningGroupExecute executes the request.
func (a *SensitiveDataScannerApi) updateScanningGroupExecute(r apiUpdateScanningGroupRequest) (SensitiveDataScannerGroupUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerGroupUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.UpdateScanningGroup")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/groups/{group_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"group_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.groupId, "")), -1)

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

type apiUpdateScanningRuleRequest struct {
	ctx    _context.Context
	ruleId string
	body   *SensitiveDataScannerRuleUpdateRequest
}

func (a *SensitiveDataScannerApi) buildUpdateScanningRuleRequest(ctx _context.Context, ruleId string, body SensitiveDataScannerRuleUpdateRequest) (apiUpdateScanningRuleRequest, error) {
	req := apiUpdateScanningRuleRequest{
		ctx:    ctx,
		ruleId: ruleId,
		body:   &body,
	}
	return req, nil
}

// UpdateScanningRule Update Scanning Rule.
// Update a scanning rule.
// The request body MUST NOT include a standard_pattern relationship, as that relationship
// is non-editable. Trying to edit the regex attribute of a rule with a standard_pattern
// relationship will also result in an error.
func (a *SensitiveDataScannerApi) UpdateScanningRule(ctx _context.Context, ruleId string, body SensitiveDataScannerRuleUpdateRequest) (SensitiveDataScannerRuleUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateScanningRuleRequest(ctx, ruleId, body)
	if err != nil {
		var localVarReturnValue SensitiveDataScannerRuleUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.updateScanningRuleExecute(req)
}

// updateScanningRuleExecute executes the request.
func (a *SensitiveDataScannerApi) updateScanningRuleExecute(r apiUpdateScanningRuleRequest) (SensitiveDataScannerRuleUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SensitiveDataScannerRuleUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.SensitiveDataScannerApi.UpdateScanningRule")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/sensitive-data-scanner/config/rules/{rule_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"rule_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.ruleId, "")), -1)

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

// NewSensitiveDataScannerApi Returns NewSensitiveDataScannerApi.
func NewSensitiveDataScannerApi(client *datadog.APIClient) *SensitiveDataScannerApi {
	return &SensitiveDataScannerApi{
		Client: client,
	}
}
