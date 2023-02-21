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

// PagerDutyIntegrationApi service type
type PagerDutyIntegrationApi datadog.Service

type apiCreatePagerDutyIntegrationServiceRequest struct {
	ctx  _context.Context
	body *PagerDutyService
}

func (a *PagerDutyIntegrationApi) buildCreatePagerDutyIntegrationServiceRequest(ctx _context.Context, body PagerDutyService) (apiCreatePagerDutyIntegrationServiceRequest, error) {
	req := apiCreatePagerDutyIntegrationServiceRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreatePagerDutyIntegrationService Create a new service object.
// Create a new service object in the PagerDuty integration.
func (a *PagerDutyIntegrationApi) CreatePagerDutyIntegrationService(ctx _context.Context, body PagerDutyService) (PagerDutyServiceName, *_nethttp.Response, error) {
	req, err := a.buildCreatePagerDutyIntegrationServiceRequest(ctx, body)
	if err != nil {
		var localVarReturnValue PagerDutyServiceName
		return localVarReturnValue, nil, err
	}

	return a.createPagerDutyIntegrationServiceExecute(req)
}

// createPagerDutyIntegrationServiceExecute executes the request.
func (a *PagerDutyIntegrationApi) createPagerDutyIntegrationServiceExecute(r apiCreatePagerDutyIntegrationServiceRequest) (PagerDutyServiceName, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue PagerDutyServiceName
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.PagerDutyIntegrationApi.CreatePagerDutyIntegrationService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/pagerduty/configuration/services"

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

type apiDeletePagerDutyIntegrationServiceRequest struct {
	ctx         _context.Context
	serviceName string
}

func (a *PagerDutyIntegrationApi) buildDeletePagerDutyIntegrationServiceRequest(ctx _context.Context, serviceName string) (apiDeletePagerDutyIntegrationServiceRequest, error) {
	req := apiDeletePagerDutyIntegrationServiceRequest{
		ctx:         ctx,
		serviceName: serviceName,
	}
	return req, nil
}

// DeletePagerDutyIntegrationService Delete a single service object.
// Delete a single service object in the Datadog-PagerDuty integration.
func (a *PagerDutyIntegrationApi) DeletePagerDutyIntegrationService(ctx _context.Context, serviceName string) (*_nethttp.Response, error) {
	req, err := a.buildDeletePagerDutyIntegrationServiceRequest(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return a.deletePagerDutyIntegrationServiceExecute(req)
}

// deletePagerDutyIntegrationServiceExecute executes the request.
func (a *PagerDutyIntegrationApi) deletePagerDutyIntegrationServiceExecute(r apiDeletePagerDutyIntegrationServiceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.PagerDutyIntegrationApi.DeletePagerDutyIntegrationService")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/pagerduty/configuration/services/{service_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceName, "")), -1)

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

type apiGetPagerDutyIntegrationServiceRequest struct {
	ctx         _context.Context
	serviceName string
}

func (a *PagerDutyIntegrationApi) buildGetPagerDutyIntegrationServiceRequest(ctx _context.Context, serviceName string) (apiGetPagerDutyIntegrationServiceRequest, error) {
	req := apiGetPagerDutyIntegrationServiceRequest{
		ctx:         ctx,
		serviceName: serviceName,
	}
	return req, nil
}

// GetPagerDutyIntegrationService Get a single service object.
// Get service name in the Datadog-PagerDuty integration.
func (a *PagerDutyIntegrationApi) GetPagerDutyIntegrationService(ctx _context.Context, serviceName string) (PagerDutyServiceName, *_nethttp.Response, error) {
	req, err := a.buildGetPagerDutyIntegrationServiceRequest(ctx, serviceName)
	if err != nil {
		var localVarReturnValue PagerDutyServiceName
		return localVarReturnValue, nil, err
	}

	return a.getPagerDutyIntegrationServiceExecute(req)
}

// getPagerDutyIntegrationServiceExecute executes the request.
func (a *PagerDutyIntegrationApi) getPagerDutyIntegrationServiceExecute(r apiGetPagerDutyIntegrationServiceRequest) (PagerDutyServiceName, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue PagerDutyServiceName
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.PagerDutyIntegrationApi.GetPagerDutyIntegrationService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/pagerduty/configuration/services/{service_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceName, "")), -1)

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

type apiUpdatePagerDutyIntegrationServiceRequest struct {
	ctx         _context.Context
	serviceName string
	body        *PagerDutyServiceKey
}

func (a *PagerDutyIntegrationApi) buildUpdatePagerDutyIntegrationServiceRequest(ctx _context.Context, serviceName string, body PagerDutyServiceKey) (apiUpdatePagerDutyIntegrationServiceRequest, error) {
	req := apiUpdatePagerDutyIntegrationServiceRequest{
		ctx:         ctx,
		serviceName: serviceName,
		body:        &body,
	}
	return req, nil
}

// UpdatePagerDutyIntegrationService Update a single service object.
// Update a single service object in the Datadog-PagerDuty integration.
func (a *PagerDutyIntegrationApi) UpdatePagerDutyIntegrationService(ctx _context.Context, serviceName string, body PagerDutyServiceKey) (*_nethttp.Response, error) {
	req, err := a.buildUpdatePagerDutyIntegrationServiceRequest(ctx, serviceName, body)
	if err != nil {
		return nil, err
	}

	return a.updatePagerDutyIntegrationServiceExecute(req)
}

// updatePagerDutyIntegrationServiceExecute executes the request.
func (a *PagerDutyIntegrationApi) updatePagerDutyIntegrationServiceExecute(r apiUpdatePagerDutyIntegrationServiceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodPut
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.PagerDutyIntegrationApi.UpdatePagerDutyIntegrationService")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/pagerduty/configuration/services/{service_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "*/*"

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

// NewPagerDutyIntegrationApi Returns NewPagerDutyIntegrationApi.
func NewPagerDutyIntegrationApi(client *datadog.APIClient) *PagerDutyIntegrationApi {
	return &PagerDutyIntegrationApi{
		Client: client,
	}
}
