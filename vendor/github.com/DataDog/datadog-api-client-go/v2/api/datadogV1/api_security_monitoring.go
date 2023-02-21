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

// SecurityMonitoringApi service type
type SecurityMonitoringApi datadog.Service

type apiAddSecurityMonitoringSignalToIncidentRequest struct {
	ctx      _context.Context
	signalId string
	body     *AddSignalToIncidentRequest
}

func (a *SecurityMonitoringApi) buildAddSecurityMonitoringSignalToIncidentRequest(ctx _context.Context, signalId string, body AddSignalToIncidentRequest) (apiAddSecurityMonitoringSignalToIncidentRequest, error) {
	req := apiAddSecurityMonitoringSignalToIncidentRequest{
		ctx:      ctx,
		signalId: signalId,
		body:     &body,
	}
	return req, nil
}

// AddSecurityMonitoringSignalToIncident Add a security signal to an incident.
// Add a security signal to an incident. This makes it possible to search for signals by incident within the signal explorer and to view the signals on the incident timeline.
func (a *SecurityMonitoringApi) AddSecurityMonitoringSignalToIncident(ctx _context.Context, signalId string, body AddSignalToIncidentRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildAddSecurityMonitoringSignalToIncidentRequest(ctx, signalId, body)
	if err != nil {
		var localVarReturnValue SuccessfulSignalUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.addSecurityMonitoringSignalToIncidentExecute(req)
}

// addSecurityMonitoringSignalToIncidentExecute executes the request.
func (a *SecurityMonitoringApi) addSecurityMonitoringSignalToIncidentExecute(r apiAddSecurityMonitoringSignalToIncidentRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SuccessfulSignalUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SecurityMonitoringApi.AddSecurityMonitoringSignalToIncident")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/security_analytics/signals/{signal_id}/add_to_incident"
	localVarPath = strings.Replace(localVarPath, "{"+"signal_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.signalId, "")), -1)

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

type apiEditSecurityMonitoringSignalAssigneeRequest struct {
	ctx      _context.Context
	signalId string
	body     *SignalAssigneeUpdateRequest
}

func (a *SecurityMonitoringApi) buildEditSecurityMonitoringSignalAssigneeRequest(ctx _context.Context, signalId string, body SignalAssigneeUpdateRequest) (apiEditSecurityMonitoringSignalAssigneeRequest, error) {
	req := apiEditSecurityMonitoringSignalAssigneeRequest{
		ctx:      ctx,
		signalId: signalId,
		body:     &body,
	}
	return req, nil
}

// EditSecurityMonitoringSignalAssignee Modify the triage assignee of a security signal.
// Modify the triage assignee of a security signal.
func (a *SecurityMonitoringApi) EditSecurityMonitoringSignalAssignee(ctx _context.Context, signalId string, body SignalAssigneeUpdateRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildEditSecurityMonitoringSignalAssigneeRequest(ctx, signalId, body)
	if err != nil {
		var localVarReturnValue SuccessfulSignalUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.editSecurityMonitoringSignalAssigneeExecute(req)
}

// editSecurityMonitoringSignalAssigneeExecute executes the request.
func (a *SecurityMonitoringApi) editSecurityMonitoringSignalAssigneeExecute(r apiEditSecurityMonitoringSignalAssigneeRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SuccessfulSignalUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SecurityMonitoringApi.EditSecurityMonitoringSignalAssignee")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/security_analytics/signals/{signal_id}/assignee"
	localVarPath = strings.Replace(localVarPath, "{"+"signal_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.signalId, "")), -1)

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

type apiEditSecurityMonitoringSignalStateRequest struct {
	ctx      _context.Context
	signalId string
	body     *SignalStateUpdateRequest
}

func (a *SecurityMonitoringApi) buildEditSecurityMonitoringSignalStateRequest(ctx _context.Context, signalId string, body SignalStateUpdateRequest) (apiEditSecurityMonitoringSignalStateRequest, error) {
	req := apiEditSecurityMonitoringSignalStateRequest{
		ctx:      ctx,
		signalId: signalId,
		body:     &body,
	}
	return req, nil
}

// EditSecurityMonitoringSignalState Change the triage state of a security signal.
// Change the triage state of a security signal.
func (a *SecurityMonitoringApi) EditSecurityMonitoringSignalState(ctx _context.Context, signalId string, body SignalStateUpdateRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildEditSecurityMonitoringSignalStateRequest(ctx, signalId, body)
	if err != nil {
		var localVarReturnValue SuccessfulSignalUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.editSecurityMonitoringSignalStateExecute(req)
}

// editSecurityMonitoringSignalStateExecute executes the request.
func (a *SecurityMonitoringApi) editSecurityMonitoringSignalStateExecute(r apiEditSecurityMonitoringSignalStateRequest) (SuccessfulSignalUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SuccessfulSignalUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SecurityMonitoringApi.EditSecurityMonitoringSignalState")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/security_analytics/signals/{signal_id}/state"
	localVarPath = strings.Replace(localVarPath, "{"+"signal_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.signalId, "")), -1)

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

// NewSecurityMonitoringApi Returns NewSecurityMonitoringApi.
func NewSecurityMonitoringApi(client *datadog.APIClient) *SecurityMonitoringApi {
	return &SecurityMonitoringApi{
		Client: client,
	}
}
