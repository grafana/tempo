// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_fmt "fmt"
	_log "log"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// IncidentServicesApi service type
type IncidentServicesApi datadog.Service

type apiCreateIncidentServiceRequest struct {
	ctx  _context.Context
	body *IncidentServiceCreateRequest
}

func (a *IncidentServicesApi) buildCreateIncidentServiceRequest(ctx _context.Context, body IncidentServiceCreateRequest) (apiCreateIncidentServiceRequest, error) {
	req := apiCreateIncidentServiceRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateIncidentService Create a new incident service.
// Creates a new incident service.
func (a *IncidentServicesApi) CreateIncidentService(ctx _context.Context, body IncidentServiceCreateRequest) (IncidentServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateIncidentServiceRequest(ctx, body)
	if err != nil {
		var localVarReturnValue IncidentServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.createIncidentServiceExecute(req)
}

// createIncidentServiceExecute executes the request.
func (a *IncidentServicesApi) createIncidentServiceExecute(r apiCreateIncidentServiceRequest) (IncidentServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IncidentServiceResponse
	)

	operationId := "v2.CreateIncidentService"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentServicesApi.CreateIncidentService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiDeleteIncidentServiceRequest struct {
	ctx       _context.Context
	serviceId string
}

func (a *IncidentServicesApi) buildDeleteIncidentServiceRequest(ctx _context.Context, serviceId string) (apiDeleteIncidentServiceRequest, error) {
	req := apiDeleteIncidentServiceRequest{
		ctx:       ctx,
		serviceId: serviceId,
	}
	return req, nil
}

// DeleteIncidentService Delete an existing incident service.
// Deletes an existing incident service.
func (a *IncidentServicesApi) DeleteIncidentService(ctx _context.Context, serviceId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteIncidentServiceRequest(ctx, serviceId)
	if err != nil {
		return nil, err
	}

	return a.deleteIncidentServiceExecute(req)
}

// deleteIncidentServiceExecute executes the request.
func (a *IncidentServicesApi) deleteIncidentServiceExecute(r apiDeleteIncidentServiceRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	operationId := "v2.DeleteIncidentService"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentServicesApi.DeleteIncidentService")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiGetIncidentServiceRequest struct {
	ctx       _context.Context
	serviceId string
	include   *IncidentRelatedObject
}

// GetIncidentServiceOptionalParameters holds optional parameters for GetIncidentService.
type GetIncidentServiceOptionalParameters struct {
	Include *IncidentRelatedObject
}

// NewGetIncidentServiceOptionalParameters creates an empty struct for parameters.
func NewGetIncidentServiceOptionalParameters() *GetIncidentServiceOptionalParameters {
	this := GetIncidentServiceOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *GetIncidentServiceOptionalParameters) WithInclude(include IncidentRelatedObject) *GetIncidentServiceOptionalParameters {
	r.Include = &include
	return r
}

func (a *IncidentServicesApi) buildGetIncidentServiceRequest(ctx _context.Context, serviceId string, o ...GetIncidentServiceOptionalParameters) (apiGetIncidentServiceRequest, error) {
	req := apiGetIncidentServiceRequest{
		ctx:       ctx,
		serviceId: serviceId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetIncidentServiceOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// GetIncidentService Get details of an incident service.
// Get details of an incident service. If the `include[users]` query parameter is provided,
// the included attribute will contain the users related to these incident services.
func (a *IncidentServicesApi) GetIncidentService(ctx _context.Context, serviceId string, o ...GetIncidentServiceOptionalParameters) (IncidentServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildGetIncidentServiceRequest(ctx, serviceId, o...)
	if err != nil {
		var localVarReturnValue IncidentServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.getIncidentServiceExecute(req)
}

// getIncidentServiceExecute executes the request.
func (a *IncidentServicesApi) getIncidentServiceExecute(r apiGetIncidentServiceRequest) (IncidentServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentServiceResponse
	)

	operationId := "v2.GetIncidentService"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentServicesApi.GetIncidentService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiListIncidentServicesRequest struct {
	ctx        _context.Context
	include    *IncidentRelatedObject
	pageSize   *int64
	pageOffset *int64
	filter     *string
}

// ListIncidentServicesOptionalParameters holds optional parameters for ListIncidentServices.
type ListIncidentServicesOptionalParameters struct {
	Include    *IncidentRelatedObject
	PageSize   *int64
	PageOffset *int64
	Filter     *string
}

// NewListIncidentServicesOptionalParameters creates an empty struct for parameters.
func NewListIncidentServicesOptionalParameters() *ListIncidentServicesOptionalParameters {
	this := ListIncidentServicesOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *ListIncidentServicesOptionalParameters) WithInclude(include IncidentRelatedObject) *ListIncidentServicesOptionalParameters {
	r.Include = &include
	return r
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListIncidentServicesOptionalParameters) WithPageSize(pageSize int64) *ListIncidentServicesOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageOffset sets the corresponding parameter name and returns the struct.
func (r *ListIncidentServicesOptionalParameters) WithPageOffset(pageOffset int64) *ListIncidentServicesOptionalParameters {
	r.PageOffset = &pageOffset
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListIncidentServicesOptionalParameters) WithFilter(filter string) *ListIncidentServicesOptionalParameters {
	r.Filter = &filter
	return r
}

func (a *IncidentServicesApi) buildListIncidentServicesRequest(ctx _context.Context, o ...ListIncidentServicesOptionalParameters) (apiListIncidentServicesRequest, error) {
	req := apiListIncidentServicesRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListIncidentServicesOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
		req.pageSize = o[0].PageSize
		req.pageOffset = o[0].PageOffset
		req.filter = o[0].Filter
	}
	return req, nil
}

// ListIncidentServices Get a list of all incident services.
// Get all incident services uploaded for the requesting user's organization. If the `include[users]` query parameter is provided, the included attribute will contain the users related to these incident services.
func (a *IncidentServicesApi) ListIncidentServices(ctx _context.Context, o ...ListIncidentServicesOptionalParameters) (IncidentServicesResponse, *_nethttp.Response, error) {
	req, err := a.buildListIncidentServicesRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue IncidentServicesResponse
		return localVarReturnValue, nil, err
	}

	return a.listIncidentServicesExecute(req)
}

// listIncidentServicesExecute executes the request.
func (a *IncidentServicesApi) listIncidentServicesExecute(r apiListIncidentServicesRequest) (IncidentServicesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentServicesResponse
	)

	operationId := "v2.ListIncidentServices"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentServicesApi.ListIncidentServices")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, ""))
	}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageOffset != nil {
		localVarQueryParams.Add("page[offset]", datadog.ParameterToString(*r.pageOffset, ""))
	}
	if r.filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*r.filter, ""))
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateIncidentServiceRequest struct {
	ctx       _context.Context
	serviceId string
	body      *IncidentServiceUpdateRequest
}

func (a *IncidentServicesApi) buildUpdateIncidentServiceRequest(ctx _context.Context, serviceId string, body IncidentServiceUpdateRequest) (apiUpdateIncidentServiceRequest, error) {
	req := apiUpdateIncidentServiceRequest{
		ctx:       ctx,
		serviceId: serviceId,
		body:      &body,
	}
	return req, nil
}

// UpdateIncidentService Update an existing incident service.
// Updates an existing incident service. Only provide the attributes which should be updated as this request is a partial update.
func (a *IncidentServicesApi) UpdateIncidentService(ctx _context.Context, serviceId string, body IncidentServiceUpdateRequest) (IncidentServiceResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateIncidentServiceRequest(ctx, serviceId, body)
	if err != nil {
		var localVarReturnValue IncidentServiceResponse
		return localVarReturnValue, nil, err
	}

	return a.updateIncidentServiceExecute(req)
}

// updateIncidentServiceExecute executes the request.
func (a *IncidentServicesApi) updateIncidentServiceExecute(r apiUpdateIncidentServiceRequest) (IncidentServiceResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue IncidentServiceResponse
	)

	operationId := "v2.UpdateIncidentService"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentServicesApi.UpdateIncidentService")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/services/{service_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"service_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.serviceId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

// NewIncidentServicesApi Returns NewIncidentServicesApi.
func NewIncidentServicesApi(client *datadog.APIClient) *IncidentServicesApi {
	return &IncidentServicesApi{
		Client: client,
	}
}
