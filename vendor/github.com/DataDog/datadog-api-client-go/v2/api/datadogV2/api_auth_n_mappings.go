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

// AuthNMappingsApi service type
type AuthNMappingsApi datadog.Service

type apiCreateAuthNMappingRequest struct {
	ctx  _context.Context
	body *AuthNMappingCreateRequest
}

func (a *AuthNMappingsApi) buildCreateAuthNMappingRequest(ctx _context.Context, body AuthNMappingCreateRequest) (apiCreateAuthNMappingRequest, error) {
	req := apiCreateAuthNMappingRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateAuthNMapping Create an AuthN Mapping.
// Create an AuthN Mapping.
func (a *AuthNMappingsApi) CreateAuthNMapping(ctx _context.Context, body AuthNMappingCreateRequest) (AuthNMappingResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateAuthNMappingRequest(ctx, body)
	if err != nil {
		var localVarReturnValue AuthNMappingResponse
		return localVarReturnValue, nil, err
	}

	return a.createAuthNMappingExecute(req)
}

// createAuthNMappingExecute executes the request.
func (a *AuthNMappingsApi) createAuthNMappingExecute(r apiCreateAuthNMappingRequest) (AuthNMappingResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue AuthNMappingResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuthNMappingsApi.CreateAuthNMapping")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/authn_mappings"

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

type apiDeleteAuthNMappingRequest struct {
	ctx            _context.Context
	authnMappingId string
}

func (a *AuthNMappingsApi) buildDeleteAuthNMappingRequest(ctx _context.Context, authnMappingId string) (apiDeleteAuthNMappingRequest, error) {
	req := apiDeleteAuthNMappingRequest{
		ctx:            ctx,
		authnMappingId: authnMappingId,
	}
	return req, nil
}

// DeleteAuthNMapping Delete an AuthN Mapping.
// Delete an AuthN Mapping specified by AuthN Mapping UUID.
func (a *AuthNMappingsApi) DeleteAuthNMapping(ctx _context.Context, authnMappingId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteAuthNMappingRequest(ctx, authnMappingId)
	if err != nil {
		return nil, err
	}

	return a.deleteAuthNMappingExecute(req)
}

// deleteAuthNMappingExecute executes the request.
func (a *AuthNMappingsApi) deleteAuthNMappingExecute(r apiDeleteAuthNMappingRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuthNMappingsApi.DeleteAuthNMapping")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/authn_mappings/{authn_mapping_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"authn_mapping_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.authnMappingId, "")), -1)

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

type apiGetAuthNMappingRequest struct {
	ctx            _context.Context
	authnMappingId string
}

func (a *AuthNMappingsApi) buildGetAuthNMappingRequest(ctx _context.Context, authnMappingId string) (apiGetAuthNMappingRequest, error) {
	req := apiGetAuthNMappingRequest{
		ctx:            ctx,
		authnMappingId: authnMappingId,
	}
	return req, nil
}

// GetAuthNMapping Get an AuthN Mapping by UUID.
// Get an AuthN Mapping specified by the AuthN Mapping UUID.
func (a *AuthNMappingsApi) GetAuthNMapping(ctx _context.Context, authnMappingId string) (AuthNMappingResponse, *_nethttp.Response, error) {
	req, err := a.buildGetAuthNMappingRequest(ctx, authnMappingId)
	if err != nil {
		var localVarReturnValue AuthNMappingResponse
		return localVarReturnValue, nil, err
	}

	return a.getAuthNMappingExecute(req)
}

// getAuthNMappingExecute executes the request.
func (a *AuthNMappingsApi) getAuthNMappingExecute(r apiGetAuthNMappingRequest) (AuthNMappingResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue AuthNMappingResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuthNMappingsApi.GetAuthNMapping")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/authn_mappings/{authn_mapping_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"authn_mapping_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.authnMappingId, "")), -1)

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

type apiListAuthNMappingsRequest struct {
	ctx        _context.Context
	pageSize   *int64
	pageNumber *int64
	sort       *AuthNMappingsSort
	filter     *string
}

// ListAuthNMappingsOptionalParameters holds optional parameters for ListAuthNMappings.
type ListAuthNMappingsOptionalParameters struct {
	PageSize   *int64
	PageNumber *int64
	Sort       *AuthNMappingsSort
	Filter     *string
}

// NewListAuthNMappingsOptionalParameters creates an empty struct for parameters.
func NewListAuthNMappingsOptionalParameters() *ListAuthNMappingsOptionalParameters {
	this := ListAuthNMappingsOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListAuthNMappingsOptionalParameters) WithPageSize(pageSize int64) *ListAuthNMappingsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListAuthNMappingsOptionalParameters) WithPageNumber(pageNumber int64) *ListAuthNMappingsOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListAuthNMappingsOptionalParameters) WithSort(sort AuthNMappingsSort) *ListAuthNMappingsOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListAuthNMappingsOptionalParameters) WithFilter(filter string) *ListAuthNMappingsOptionalParameters {
	r.Filter = &filter
	return r
}

func (a *AuthNMappingsApi) buildListAuthNMappingsRequest(ctx _context.Context, o ...ListAuthNMappingsOptionalParameters) (apiListAuthNMappingsRequest, error) {
	req := apiListAuthNMappingsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListAuthNMappingsOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
	}
	return req, nil
}

// ListAuthNMappings List all AuthN Mappings.
// List all AuthN Mappings in the org.
func (a *AuthNMappingsApi) ListAuthNMappings(ctx _context.Context, o ...ListAuthNMappingsOptionalParameters) (AuthNMappingsResponse, *_nethttp.Response, error) {
	req, err := a.buildListAuthNMappingsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue AuthNMappingsResponse
		return localVarReturnValue, nil, err
	}

	return a.listAuthNMappingsExecute(req)
}

// listAuthNMappingsExecute executes the request.
func (a *AuthNMappingsApi) listAuthNMappingsExecute(r apiListAuthNMappingsRequest) (AuthNMappingsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue AuthNMappingsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuthNMappingsApi.ListAuthNMappings")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/authn_mappings"

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

type apiUpdateAuthNMappingRequest struct {
	ctx            _context.Context
	authnMappingId string
	body           *AuthNMappingUpdateRequest
}

func (a *AuthNMappingsApi) buildUpdateAuthNMappingRequest(ctx _context.Context, authnMappingId string, body AuthNMappingUpdateRequest) (apiUpdateAuthNMappingRequest, error) {
	req := apiUpdateAuthNMappingRequest{
		ctx:            ctx,
		authnMappingId: authnMappingId,
		body:           &body,
	}
	return req, nil
}

// UpdateAuthNMapping Edit an AuthN Mapping.
// Edit an AuthN Mapping.
func (a *AuthNMappingsApi) UpdateAuthNMapping(ctx _context.Context, authnMappingId string, body AuthNMappingUpdateRequest) (AuthNMappingResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateAuthNMappingRequest(ctx, authnMappingId, body)
	if err != nil {
		var localVarReturnValue AuthNMappingResponse
		return localVarReturnValue, nil, err
	}

	return a.updateAuthNMappingExecute(req)
}

// updateAuthNMappingExecute executes the request.
func (a *AuthNMappingsApi) updateAuthNMappingExecute(r apiUpdateAuthNMappingRequest) (AuthNMappingResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue AuthNMappingResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.AuthNMappingsApi.UpdateAuthNMapping")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/authn_mappings/{authn_mapping_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"authn_mapping_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.authnMappingId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 409 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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

// NewAuthNMappingsApi Returns NewAuthNMappingsApi.
func NewAuthNMappingsApi(client *datadog.APIClient) *AuthNMappingsApi {
	return &AuthNMappingsApi{
		Client: client,
	}
}
