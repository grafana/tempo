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

// ServiceLevelObjectiveCorrectionsApi service type
type ServiceLevelObjectiveCorrectionsApi datadog.Service

type apiCreateSLOCorrectionRequest struct {
	ctx  _context.Context
	body *SLOCorrectionCreateRequest
}

func (a *ServiceLevelObjectiveCorrectionsApi) buildCreateSLOCorrectionRequest(ctx _context.Context, body SLOCorrectionCreateRequest) (apiCreateSLOCorrectionRequest, error) {
	req := apiCreateSLOCorrectionRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateSLOCorrection Create an SLO correction.
// Create an SLO Correction.
func (a *ServiceLevelObjectiveCorrectionsApi) CreateSLOCorrection(ctx _context.Context, body SLOCorrectionCreateRequest) (SLOCorrectionResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateSLOCorrectionRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SLOCorrectionResponse
		return localVarReturnValue, nil, err
	}

	return a.createSLOCorrectionExecute(req)
}

// createSLOCorrectionExecute executes the request.
func (a *ServiceLevelObjectiveCorrectionsApi) createSLOCorrectionExecute(r apiCreateSLOCorrectionRequest) (SLOCorrectionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectiveCorrectionsApi.CreateSLOCorrection")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/correction"

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

type apiDeleteSLOCorrectionRequest struct {
	ctx             _context.Context
	sloCorrectionId string
}

func (a *ServiceLevelObjectiveCorrectionsApi) buildDeleteSLOCorrectionRequest(ctx _context.Context, sloCorrectionId string) (apiDeleteSLOCorrectionRequest, error) {
	req := apiDeleteSLOCorrectionRequest{
		ctx:             ctx,
		sloCorrectionId: sloCorrectionId,
	}
	return req, nil
}

// DeleteSLOCorrection Delete an SLO correction.
// Permanently delete the specified SLO correction object.
func (a *ServiceLevelObjectiveCorrectionsApi) DeleteSLOCorrection(ctx _context.Context, sloCorrectionId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteSLOCorrectionRequest(ctx, sloCorrectionId)
	if err != nil {
		return nil, err
	}

	return a.deleteSLOCorrectionExecute(req)
}

// deleteSLOCorrectionExecute executes the request.
func (a *ServiceLevelObjectiveCorrectionsApi) deleteSLOCorrectionExecute(r apiDeleteSLOCorrectionRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectiveCorrectionsApi.DeleteSLOCorrection")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/correction/{slo_correction_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_correction_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloCorrectionId, "")), -1)

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

type apiGetSLOCorrectionRequest struct {
	ctx             _context.Context
	sloCorrectionId string
}

func (a *ServiceLevelObjectiveCorrectionsApi) buildGetSLOCorrectionRequest(ctx _context.Context, sloCorrectionId string) (apiGetSLOCorrectionRequest, error) {
	req := apiGetSLOCorrectionRequest{
		ctx:             ctx,
		sloCorrectionId: sloCorrectionId,
	}
	return req, nil
}

// GetSLOCorrection Get an SLO correction for an SLO.
// Get an SLO correction.
func (a *ServiceLevelObjectiveCorrectionsApi) GetSLOCorrection(ctx _context.Context, sloCorrectionId string) (SLOCorrectionResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLOCorrectionRequest(ctx, sloCorrectionId)
	if err != nil {
		var localVarReturnValue SLOCorrectionResponse
		return localVarReturnValue, nil, err
	}

	return a.getSLOCorrectionExecute(req)
}

// getSLOCorrectionExecute executes the request.
func (a *ServiceLevelObjectiveCorrectionsApi) getSLOCorrectionExecute(r apiGetSLOCorrectionRequest) (SLOCorrectionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectiveCorrectionsApi.GetSLOCorrection")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/correction/{slo_correction_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_correction_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloCorrectionId, "")), -1)

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

type apiListSLOCorrectionRequest struct {
	ctx    _context.Context
	offset *int64
	limit  *int64
}

// ListSLOCorrectionOptionalParameters holds optional parameters for ListSLOCorrection.
type ListSLOCorrectionOptionalParameters struct {
	Offset *int64
	Limit  *int64
}

// NewListSLOCorrectionOptionalParameters creates an empty struct for parameters.
func NewListSLOCorrectionOptionalParameters() *ListSLOCorrectionOptionalParameters {
	this := ListSLOCorrectionOptionalParameters{}
	return &this
}

// WithOffset sets the corresponding parameter name and returns the struct.
func (r *ListSLOCorrectionOptionalParameters) WithOffset(offset int64) *ListSLOCorrectionOptionalParameters {
	r.Offset = &offset
	return r
}

// WithLimit sets the corresponding parameter name and returns the struct.
func (r *ListSLOCorrectionOptionalParameters) WithLimit(limit int64) *ListSLOCorrectionOptionalParameters {
	r.Limit = &limit
	return r
}

func (a *ServiceLevelObjectiveCorrectionsApi) buildListSLOCorrectionRequest(ctx _context.Context, o ...ListSLOCorrectionOptionalParameters) (apiListSLOCorrectionRequest, error) {
	req := apiListSLOCorrectionRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListSLOCorrectionOptionalParameters is allowed")
	}

	if o != nil {
		req.offset = o[0].Offset
		req.limit = o[0].Limit
	}
	return req, nil
}

// ListSLOCorrection Get all SLO corrections.
// Get all Service Level Objective corrections.
func (a *ServiceLevelObjectiveCorrectionsApi) ListSLOCorrection(ctx _context.Context, o ...ListSLOCorrectionOptionalParameters) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	req, err := a.buildListSLOCorrectionRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SLOCorrectionListResponse
		return localVarReturnValue, nil, err
	}

	return a.listSLOCorrectionExecute(req)
}

// listSLOCorrectionExecute executes the request.
func (a *ServiceLevelObjectiveCorrectionsApi) listSLOCorrectionExecute(r apiListSLOCorrectionRequest) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectiveCorrectionsApi.ListSLOCorrection")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/correction"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.offset != nil {
		localVarQueryParams.Add("offset", datadog.ParameterToString(*r.offset, ""))
	}
	if r.limit != nil {
		localVarQueryParams.Add("limit", datadog.ParameterToString(*r.limit, ""))
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

type apiUpdateSLOCorrectionRequest struct {
	ctx             _context.Context
	sloCorrectionId string
	body            *SLOCorrectionUpdateRequest
}

func (a *ServiceLevelObjectiveCorrectionsApi) buildUpdateSLOCorrectionRequest(ctx _context.Context, sloCorrectionId string, body SLOCorrectionUpdateRequest) (apiUpdateSLOCorrectionRequest, error) {
	req := apiUpdateSLOCorrectionRequest{
		ctx:             ctx,
		sloCorrectionId: sloCorrectionId,
		body:            &body,
	}
	return req, nil
}

// UpdateSLOCorrection Update an SLO correction.
// Update the specified SLO correction object.
func (a *ServiceLevelObjectiveCorrectionsApi) UpdateSLOCorrection(ctx _context.Context, sloCorrectionId string, body SLOCorrectionUpdateRequest) (SLOCorrectionResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateSLOCorrectionRequest(ctx, sloCorrectionId, body)
	if err != nil {
		var localVarReturnValue SLOCorrectionResponse
		return localVarReturnValue, nil, err
	}

	return a.updateSLOCorrectionExecute(req)
}

// updateSLOCorrectionExecute executes the request.
func (a *ServiceLevelObjectiveCorrectionsApi) updateSLOCorrectionExecute(r apiUpdateSLOCorrectionRequest) (SLOCorrectionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectiveCorrectionsApi.UpdateSLOCorrection")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/correction/{slo_correction_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_correction_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloCorrectionId, "")), -1)

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

// NewServiceLevelObjectiveCorrectionsApi Returns NewServiceLevelObjectiveCorrectionsApi.
func NewServiceLevelObjectiveCorrectionsApi(client *datadog.APIClient) *ServiceLevelObjectiveCorrectionsApi {
	return &ServiceLevelObjectiveCorrectionsApi{
		Client: client,
	}
}
