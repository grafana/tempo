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

// OpsgenieIntegrationApi service type
type OpsgenieIntegrationApi datadog.Service

type apiCreateOpsgenieServiceRequest struct {
	ctx  _context.Context
	body *OpsgenieServiceCreateRequest
}

func (a *OpsgenieIntegrationApi) buildCreateOpsgenieServiceRequest(ctx _context.Context, body OpsgenieServiceCreateRequest) (apiCreateOpsgenieServiceRequest, error) {
	req := apiCreateOpsgenieServiceRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateOpsgenieService Create a new service object.
// Create a new service object in the Opsgenie integration.
func (a *OpsgenieIntegrationApi) CreateOpsgenieService(ctx _context.Context, body OpsgenieServiceCreateRequest) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateOpsgenieServiceRequest(ctx, body)
	if err != nil {
		var localVarReturnValue OpsgenieServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.createOpsgenieServiceExecute(req)
}

// createOpsgenieServiceExecute executes the request.
func (a *OpsgenieIntegrationApi) createOpsgenieServiceExecute(r apiCreateOpsgenieServiceRequest) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue OpsgenieServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.OpsgenieIntegrationApi.CreateOpsgenieService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integration/opsgenie/services"

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

type apiDeleteOpsgenieServiceRequest struct {
	ctx                  _context.Context
	integrationServiceId string
}

func (a *OpsgenieIntegrationApi) buildDeleteOpsgenieServiceRequest(ctx _context.Context, integrationServiceId string) (apiDeleteOpsgenieServiceRequest, error) {
	req := apiDeleteOpsgenieServiceRequest{
		ctx:                  ctx,
		integrationServiceId: integrationServiceId,
	}
	return req, nil
}

// DeleteOpsgenieService Delete a single service object.
// Delete a single service object in the Datadog Opsgenie integration.
func (a *OpsgenieIntegrationApi) DeleteOpsgenieService(ctx _context.Context, integrationServiceId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteOpsgenieServiceRequest(ctx, integrationServiceId)
	if err != nil {
		return nil, err
	}

	return a.deleteOpsgenieServiceExecute(req)
}

// deleteOpsgenieServiceExecute executes the request.
func (a *OpsgenieIntegrationApi) deleteOpsgenieServiceExecute(r apiDeleteOpsgenieServiceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.OpsgenieIntegrationApi.DeleteOpsgenieService")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integration/opsgenie/services/{integration_service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"integration_service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.integrationServiceId, "")), -1)

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

type apiGetOpsgenieServiceRequest struct {
	ctx                  _context.Context
	integrationServiceId string
}

func (a *OpsgenieIntegrationApi) buildGetOpsgenieServiceRequest(ctx _context.Context, integrationServiceId string) (apiGetOpsgenieServiceRequest, error) {
	req := apiGetOpsgenieServiceRequest{
		ctx:                  ctx,
		integrationServiceId: integrationServiceId,
	}
	return req, nil
}

// GetOpsgenieService Get a single service object.
// Get a single service from the Datadog Opsgenie integration.
func (a *OpsgenieIntegrationApi) GetOpsgenieService(ctx _context.Context, integrationServiceId string) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildGetOpsgenieServiceRequest(ctx, integrationServiceId)
	if err != nil {
		var localVarReturnValue OpsgenieServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.getOpsgenieServiceExecute(req)
}

// getOpsgenieServiceExecute executes the request.
func (a *OpsgenieIntegrationApi) getOpsgenieServiceExecute(r apiGetOpsgenieServiceRequest) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue OpsgenieServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.OpsgenieIntegrationApi.GetOpsgenieService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integration/opsgenie/services/{integration_service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"integration_service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.integrationServiceId, "")), -1)

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

type apiListOpsgenieServicesRequest struct {
	ctx _context.Context
}

func (a *OpsgenieIntegrationApi) buildListOpsgenieServicesRequest(ctx _context.Context) (apiListOpsgenieServicesRequest, error) {
	req := apiListOpsgenieServicesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListOpsgenieServices Get all service objects.
// Get a list of all services from the Datadog Opsgenie integration.
func (a *OpsgenieIntegrationApi) ListOpsgenieServices(ctx _context.Context) (OpsgenieServicesResponse, *_nethttp.Response, error) {
	req, err := a.buildListOpsgenieServicesRequest(ctx)
	if err != nil {
		var localVarReturnValue OpsgenieServicesResponse
		return localVarReturnValue, nil, err
	}

	return a.listOpsgenieServicesExecute(req)
}

// listOpsgenieServicesExecute executes the request.
func (a *OpsgenieIntegrationApi) listOpsgenieServicesExecute(r apiListOpsgenieServicesRequest) (OpsgenieServicesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue OpsgenieServicesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.OpsgenieIntegrationApi.ListOpsgenieServices")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integration/opsgenie/services"

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

type apiUpdateOpsgenieServiceRequest struct {
	ctx                  _context.Context
	integrationServiceId string
	body                 *OpsgenieServiceUpdateRequest
}

func (a *OpsgenieIntegrationApi) buildUpdateOpsgenieServiceRequest(ctx _context.Context, integrationServiceId string, body OpsgenieServiceUpdateRequest) (apiUpdateOpsgenieServiceRequest, error) {
	req := apiUpdateOpsgenieServiceRequest{
		ctx:                  ctx,
		integrationServiceId: integrationServiceId,
		body:                 &body,
	}
	return req, nil
}

// UpdateOpsgenieService Update a single service object.
// Update a single service object in the Datadog Opsgenie integration.
func (a *OpsgenieIntegrationApi) UpdateOpsgenieService(ctx _context.Context, integrationServiceId string, body OpsgenieServiceUpdateRequest) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateOpsgenieServiceRequest(ctx, integrationServiceId, body)
	if err != nil {
		var localVarReturnValue OpsgenieServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.updateOpsgenieServiceExecute(req)
}

// updateOpsgenieServiceExecute executes the request.
func (a *OpsgenieIntegrationApi) updateOpsgenieServiceExecute(r apiUpdateOpsgenieServiceRequest) (OpsgenieServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue OpsgenieServiceResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.OpsgenieIntegrationApi.UpdateOpsgenieService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/integration/opsgenie/services/{integration_service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"integration_service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.integrationServiceId, "")), -1)

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

// NewOpsgenieIntegrationApi Returns NewOpsgenieIntegrationApi.
func NewOpsgenieIntegrationApi(client *datadog.APIClient) *OpsgenieIntegrationApi {
	return &OpsgenieIntegrationApi{
		Client: client,
	}
}
