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

// ServiceDefinitionApi service type
type ServiceDefinitionApi datadog.Service

type apiCreateOrUpdateServiceDefinitionsRequest struct {
	ctx  _context.Context
	body *ServiceDefinitionsCreateRequest
}

func (a *ServiceDefinitionApi) buildCreateOrUpdateServiceDefinitionsRequest(ctx _context.Context, body ServiceDefinitionsCreateRequest) (apiCreateOrUpdateServiceDefinitionsRequest, error) {
	req := apiCreateOrUpdateServiceDefinitionsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateOrUpdateServiceDefinitions Create or update service definition.
// Create or update service definition in the Datadog Service Catalog.
func (a *ServiceDefinitionApi) CreateOrUpdateServiceDefinitions(ctx _context.Context, body ServiceDefinitionsCreateRequest) (ServiceDefinitionCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateOrUpdateServiceDefinitionsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue ServiceDefinitionCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createOrUpdateServiceDefinitionsExecute(req)
}

// createOrUpdateServiceDefinitionsExecute executes the request.
func (a *ServiceDefinitionApi) createOrUpdateServiceDefinitionsExecute(r apiCreateOrUpdateServiceDefinitionsRequest) (ServiceDefinitionCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue ServiceDefinitionCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ServiceDefinitionApi.CreateOrUpdateServiceDefinitions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/definitions"

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

type apiDeleteServiceDefinitionRequest struct {
	ctx         _context.Context
	serviceName string
}

func (a *ServiceDefinitionApi) buildDeleteServiceDefinitionRequest(ctx _context.Context, serviceName string) (apiDeleteServiceDefinitionRequest, error) {
	req := apiDeleteServiceDefinitionRequest{
		ctx:         ctx,
		serviceName: serviceName,
	}
	return req, nil
}

// DeleteServiceDefinition Delete a single service definition.
// Delete a single service definition in the Datadog Service Catalog.
func (a *ServiceDefinitionApi) DeleteServiceDefinition(ctx _context.Context, serviceName string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteServiceDefinitionRequest(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return a.deleteServiceDefinitionExecute(req)
}

// deleteServiceDefinitionExecute executes the request.
func (a *ServiceDefinitionApi) deleteServiceDefinitionExecute(r apiDeleteServiceDefinitionRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ServiceDefinitionApi.DeleteServiceDefinition")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/definitions/{service_name}"
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

type apiGetServiceDefinitionRequest struct {
	ctx         _context.Context
	serviceName string
}

func (a *ServiceDefinitionApi) buildGetServiceDefinitionRequest(ctx _context.Context, serviceName string) (apiGetServiceDefinitionRequest, error) {
	req := apiGetServiceDefinitionRequest{
		ctx:         ctx,
		serviceName: serviceName,
	}
	return req, nil
}

// GetServiceDefinition Get a single service definition.
// Get a single service definition from the Datadog Service Catalog.
func (a *ServiceDefinitionApi) GetServiceDefinition(ctx _context.Context, serviceName string) (ServiceDefinitionGetResponse, *_nethttp.Response, error) {
	req, err := a.buildGetServiceDefinitionRequest(ctx, serviceName)
	if err != nil {
		var localVarReturnValue ServiceDefinitionGetResponse
		return localVarReturnValue, nil, err
	}

	return a.getServiceDefinitionExecute(req)
}

// getServiceDefinitionExecute executes the request.
func (a *ServiceDefinitionApi) getServiceDefinitionExecute(r apiGetServiceDefinitionRequest) (ServiceDefinitionGetResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ServiceDefinitionGetResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ServiceDefinitionApi.GetServiceDefinition")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/definitions/{service_name}"
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

type apiListServiceDefinitionsRequest struct {
	ctx _context.Context
}

func (a *ServiceDefinitionApi) buildListServiceDefinitionsRequest(ctx _context.Context) (apiListServiceDefinitionsRequest, error) {
	req := apiListServiceDefinitionsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListServiceDefinitions Get all service definitions.
// Get a list of all service definitions from the Datadog Service Catalog.
func (a *ServiceDefinitionApi) ListServiceDefinitions(ctx _context.Context) (ServiceDefinitionsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListServiceDefinitionsRequest(ctx)
	if err != nil {
		var localVarReturnValue ServiceDefinitionsListResponse
		return localVarReturnValue, nil, err
	}

	return a.listServiceDefinitionsExecute(req)
}

// listServiceDefinitionsExecute executes the request.
func (a *ServiceDefinitionApi) listServiceDefinitionsExecute(r apiListServiceDefinitionsRequest) (ServiceDefinitionsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ServiceDefinitionsListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.ServiceDefinitionApi.ListServiceDefinitions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/definitions"

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

// NewServiceDefinitionApi Returns NewServiceDefinitionApi.
func NewServiceDefinitionApi(client *datadog.APIClient) *ServiceDefinitionApi {
	return &ServiceDefinitionApi{
		Client: client,
	}
}
