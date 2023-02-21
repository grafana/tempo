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

// ServiceLevelObjectivesApi service type
type ServiceLevelObjectivesApi datadog.Service

type apiCheckCanDeleteSLORequest struct {
	ctx _context.Context
	ids *string
}

func (a *ServiceLevelObjectivesApi) buildCheckCanDeleteSLORequest(ctx _context.Context, ids string) (apiCheckCanDeleteSLORequest, error) {
	req := apiCheckCanDeleteSLORequest{
		ctx: ctx,
		ids: &ids,
	}
	return req, nil
}

// CheckCanDeleteSLO Check if SLOs can be safely deleted.
// Check if an SLO can be safely deleted. For example,
// assure an SLO can be deleted without disrupting a dashboard.
func (a *ServiceLevelObjectivesApi) CheckCanDeleteSLO(ctx _context.Context, ids string) (CheckCanDeleteSLOResponse, *_nethttp.Response, error) {
	req, err := a.buildCheckCanDeleteSLORequest(ctx, ids)
	if err != nil {
		var localVarReturnValue CheckCanDeleteSLOResponse
		return localVarReturnValue, nil, err
	}

	return a.checkCanDeleteSLOExecute(req)
}

// checkCanDeleteSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) checkCanDeleteSLOExecute(r apiCheckCanDeleteSLORequest) (CheckCanDeleteSLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CheckCanDeleteSLOResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.CheckCanDeleteSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/can_delete"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.ids == nil {
		return localVarReturnValue, nil, datadog.ReportError("ids is required and must be specified")
	}
	localVarQueryParams.Add("ids", datadog.ParameterToString(*r.ids, ""))
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
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 409 {
			var v CheckCanDeleteSLOResponse
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

type apiCreateSLORequest struct {
	ctx  _context.Context
	body *ServiceLevelObjectiveRequest
}

func (a *ServiceLevelObjectivesApi) buildCreateSLORequest(ctx _context.Context, body ServiceLevelObjectiveRequest) (apiCreateSLORequest, error) {
	req := apiCreateSLORequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateSLO Create an SLO object.
// Create a service level objective object.
func (a *ServiceLevelObjectivesApi) CreateSLO(ctx _context.Context, body ServiceLevelObjectiveRequest) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateSLORequest(ctx, body)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return a.createSLOExecute(req)
}

// createSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) createSLOExecute(r apiCreateSLORequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.CreateSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo"

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

type apiDeleteSLORequest struct {
	ctx   _context.Context
	sloId string
	force *string
}

// DeleteSLOOptionalParameters holds optional parameters for DeleteSLO.
type DeleteSLOOptionalParameters struct {
	Force *string
}

// NewDeleteSLOOptionalParameters creates an empty struct for parameters.
func NewDeleteSLOOptionalParameters() *DeleteSLOOptionalParameters {
	this := DeleteSLOOptionalParameters{}
	return &this
}

// WithForce sets the corresponding parameter name and returns the struct.
func (r *DeleteSLOOptionalParameters) WithForce(force string) *DeleteSLOOptionalParameters {
	r.Force = &force
	return r
}

func (a *ServiceLevelObjectivesApi) buildDeleteSLORequest(ctx _context.Context, sloId string, o ...DeleteSLOOptionalParameters) (apiDeleteSLORequest, error) {
	req := apiDeleteSLORequest{
		ctx:   ctx,
		sloId: sloId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type DeleteSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.force = o[0].Force
	}
	return req, nil
}

// DeleteSLO Delete an SLO.
// Permanently delete the specified service level objective object.
//
// If an SLO is used in a dashboard, the `DELETE /v1/slo/` endpoint returns
// a 409 conflict error because the SLO is referenced in a dashboard.
func (a *ServiceLevelObjectivesApi) DeleteSLO(ctx _context.Context, sloId string, o ...DeleteSLOOptionalParameters) (SLODeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteSLORequest(ctx, sloId, o...)
	if err != nil {
		var localVarReturnValue SLODeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteSLOExecute(req)
}

// deleteSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) deleteSLOExecute(r apiDeleteSLORequest) (SLODeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue SLODeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.DeleteSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.force != nil {
		localVarQueryParams.Add("force", datadog.ParameterToString(*r.force, ""))
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
		if localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
			var v APIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 409 {
			var v SLODeleteResponse
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

type apiDeleteSLOTimeframeInBulkRequest struct {
	ctx  _context.Context
	body *map[string][]SLOTimeframe
}

func (a *ServiceLevelObjectivesApi) buildDeleteSLOTimeframeInBulkRequest(ctx _context.Context, body map[string][]SLOTimeframe) (apiDeleteSLOTimeframeInBulkRequest, error) {
	req := apiDeleteSLOTimeframeInBulkRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteSLOTimeframeInBulk Bulk Delete SLO Timeframes.
// Delete (or partially delete) multiple service level objective objects.
//
// This endpoint facilitates deletion of one or more thresholds for one or more
// service level objective objects. If all thresholds are deleted, the service level
// objective object is deleted as well.
func (a *ServiceLevelObjectivesApi) DeleteSLOTimeframeInBulk(ctx _context.Context, body map[string][]SLOTimeframe) (SLOBulkDeleteResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteSLOTimeframeInBulkRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SLOBulkDeleteResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteSLOTimeframeInBulkExecute(req)
}

// deleteSLOTimeframeInBulkExecute executes the request.
func (a *ServiceLevelObjectivesApi) deleteSLOTimeframeInBulkExecute(r apiDeleteSLOTimeframeInBulkRequest) (SLOBulkDeleteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SLOBulkDeleteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.DeleteSLOTimeframeInBulk")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/bulk_delete"

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

type apiGetSLORequest struct {
	ctx                    _context.Context
	sloId                  string
	withConfiguredAlertIds *bool
}

// GetSLOOptionalParameters holds optional parameters for GetSLO.
type GetSLOOptionalParameters struct {
	WithConfiguredAlertIds *bool
}

// NewGetSLOOptionalParameters creates an empty struct for parameters.
func NewGetSLOOptionalParameters() *GetSLOOptionalParameters {
	this := GetSLOOptionalParameters{}
	return &this
}

// WithWithConfiguredAlertIds sets the corresponding parameter name and returns the struct.
func (r *GetSLOOptionalParameters) WithWithConfiguredAlertIds(withConfiguredAlertIds bool) *GetSLOOptionalParameters {
	r.WithConfiguredAlertIds = &withConfiguredAlertIds
	return r
}

func (a *ServiceLevelObjectivesApi) buildGetSLORequest(ctx _context.Context, sloId string, o ...GetSLOOptionalParameters) (apiGetSLORequest, error) {
	req := apiGetSLORequest{
		ctx:   ctx,
		sloId: sloId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.withConfiguredAlertIds = o[0].WithConfiguredAlertIds
	}
	return req, nil
}

// GetSLO Get an SLO's details.
// Get a service level objective object.
func (a *ServiceLevelObjectivesApi) GetSLO(ctx _context.Context, sloId string, o ...GetSLOOptionalParameters) (SLOResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLORequest(ctx, sloId, o...)
	if err != nil {
		var localVarReturnValue SLOResponse
		return localVarReturnValue, nil, err
	}

	return a.getSLOExecute(req)
}

// getSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) getSLOExecute(r apiGetSLORequest) (SLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.GetSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.withConfiguredAlertIds != nil {
		localVarQueryParams.Add("with_configured_alert_ids", datadog.ParameterToString(*r.withConfiguredAlertIds, ""))
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

type apiGetSLOCorrectionsRequest struct {
	ctx   _context.Context
	sloId string
}

func (a *ServiceLevelObjectivesApi) buildGetSLOCorrectionsRequest(ctx _context.Context, sloId string) (apiGetSLOCorrectionsRequest, error) {
	req := apiGetSLOCorrectionsRequest{
		ctx:   ctx,
		sloId: sloId,
	}
	return req, nil
}

// GetSLOCorrections Get Corrections For an SLO.
// Get corrections applied to an SLO
func (a *ServiceLevelObjectivesApi) GetSLOCorrections(ctx _context.Context, sloId string) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLOCorrectionsRequest(ctx, sloId)
	if err != nil {
		var localVarReturnValue SLOCorrectionListResponse
		return localVarReturnValue, nil, err
	}

	return a.getSLOCorrectionsExecute(req)
}

// getSLOCorrectionsExecute executes the request.
func (a *ServiceLevelObjectivesApi) getSLOCorrectionsExecute(r apiGetSLOCorrectionsRequest) (SLOCorrectionListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOCorrectionListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.GetSLOCorrections")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}/corrections"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloId, "")), -1)

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

type apiGetSLOHistoryRequest struct {
	ctx             _context.Context
	sloId           string
	fromTs          *int64
	toTs            *int64
	target          *float64
	applyCorrection *bool
}

// GetSLOHistoryOptionalParameters holds optional parameters for GetSLOHistory.
type GetSLOHistoryOptionalParameters struct {
	Target          *float64
	ApplyCorrection *bool
}

// NewGetSLOHistoryOptionalParameters creates an empty struct for parameters.
func NewGetSLOHistoryOptionalParameters() *GetSLOHistoryOptionalParameters {
	this := GetSLOHistoryOptionalParameters{}
	return &this
}

// WithTarget sets the corresponding parameter name and returns the struct.
func (r *GetSLOHistoryOptionalParameters) WithTarget(target float64) *GetSLOHistoryOptionalParameters {
	r.Target = &target
	return r
}

// WithApplyCorrection sets the corresponding parameter name and returns the struct.
func (r *GetSLOHistoryOptionalParameters) WithApplyCorrection(applyCorrection bool) *GetSLOHistoryOptionalParameters {
	r.ApplyCorrection = &applyCorrection
	return r
}

func (a *ServiceLevelObjectivesApi) buildGetSLOHistoryRequest(ctx _context.Context, sloId string, fromTs int64, toTs int64, o ...GetSLOHistoryOptionalParameters) (apiGetSLOHistoryRequest, error) {
	req := apiGetSLOHistoryRequest{
		ctx:    ctx,
		sloId:  sloId,
		fromTs: &fromTs,
		toTs:   &toTs,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetSLOHistoryOptionalParameters is allowed")
	}

	if o != nil {
		req.target = o[0].Target
		req.applyCorrection = o[0].ApplyCorrection
	}
	return req, nil
}

// GetSLOHistory Get an SLO's history.
// Get a specific SLO’s history, regardless of its SLO type.
//
// The detailed history data is structured according to the source data type.
// For example, metric data is included for event SLOs that use
// the metric source, and monitor SLO types include the monitor transition history.
//
// **Note:** There are different response formats for event based and time based SLOs.
// Examples of both are shown.
func (a *ServiceLevelObjectivesApi) GetSLOHistory(ctx _context.Context, sloId string, fromTs int64, toTs int64, o ...GetSLOHistoryOptionalParameters) (SLOHistoryResponse, *_nethttp.Response, error) {
	req, err := a.buildGetSLOHistoryRequest(ctx, sloId, fromTs, toTs, o...)
	if err != nil {
		var localVarReturnValue SLOHistoryResponse
		return localVarReturnValue, nil, err
	}

	return a.getSLOHistoryExecute(req)
}

// getSLOHistoryExecute executes the request.
func (a *ServiceLevelObjectivesApi) getSLOHistoryExecute(r apiGetSLOHistoryRequest) (SLOHistoryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOHistoryResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.GetSLOHistory")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}/history"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.fromTs == nil {
		return localVarReturnValue, nil, datadog.ReportError("fromTs is required and must be specified")
	}
	if r.toTs == nil {
		return localVarReturnValue, nil, datadog.ReportError("toTs is required and must be specified")
	}
	localVarQueryParams.Add("from_ts", datadog.ParameterToString(*r.fromTs, ""))
	localVarQueryParams.Add("to_ts", datadog.ParameterToString(*r.toTs, ""))
	if r.target != nil {
		localVarQueryParams.Add("target", datadog.ParameterToString(*r.target, ""))
	}
	if r.applyCorrection != nil {
		localVarQueryParams.Add("apply_correction", datadog.ParameterToString(*r.applyCorrection, ""))
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

type apiListSLOsRequest struct {
	ctx          _context.Context
	ids          *string
	query        *string
	tagsQuery    *string
	metricsQuery *string
	limit        *int64
	offset       *int64
}

// ListSLOsOptionalParameters holds optional parameters for ListSLOs.
type ListSLOsOptionalParameters struct {
	Ids          *string
	Query        *string
	TagsQuery    *string
	MetricsQuery *string
	Limit        *int64
	Offset       *int64
}

// NewListSLOsOptionalParameters creates an empty struct for parameters.
func NewListSLOsOptionalParameters() *ListSLOsOptionalParameters {
	this := ListSLOsOptionalParameters{}
	return &this
}

// WithIds sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithIds(ids string) *ListSLOsOptionalParameters {
	r.Ids = &ids
	return r
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithQuery(query string) *ListSLOsOptionalParameters {
	r.Query = &query
	return r
}

// WithTagsQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithTagsQuery(tagsQuery string) *ListSLOsOptionalParameters {
	r.TagsQuery = &tagsQuery
	return r
}

// WithMetricsQuery sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithMetricsQuery(metricsQuery string) *ListSLOsOptionalParameters {
	r.MetricsQuery = &metricsQuery
	return r
}

// WithLimit sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithLimit(limit int64) *ListSLOsOptionalParameters {
	r.Limit = &limit
	return r
}

// WithOffset sets the corresponding parameter name and returns the struct.
func (r *ListSLOsOptionalParameters) WithOffset(offset int64) *ListSLOsOptionalParameters {
	r.Offset = &offset
	return r
}

func (a *ServiceLevelObjectivesApi) buildListSLOsRequest(ctx _context.Context, o ...ListSLOsOptionalParameters) (apiListSLOsRequest, error) {
	req := apiListSLOsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListSLOsOptionalParameters is allowed")
	}

	if o != nil {
		req.ids = o[0].Ids
		req.query = o[0].Query
		req.tagsQuery = o[0].TagsQuery
		req.metricsQuery = o[0].MetricsQuery
		req.limit = o[0].Limit
		req.offset = o[0].Offset
	}
	return req, nil
}

// ListSLOs Get all SLOs.
// Get a list of service level objective objects for your organization.
func (a *ServiceLevelObjectivesApi) ListSLOs(ctx _context.Context, o ...ListSLOsOptionalParameters) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildListSLOsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return a.listSLOsExecute(req)
}

// listSLOsExecute executes the request.
func (a *ServiceLevelObjectivesApi) listSLOsExecute(r apiListSLOsRequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.ListSLOs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.ids != nil {
		localVarQueryParams.Add("ids", datadog.ParameterToString(*r.ids, ""))
	}
	if r.query != nil {
		localVarQueryParams.Add("query", datadog.ParameterToString(*r.query, ""))
	}
	if r.tagsQuery != nil {
		localVarQueryParams.Add("tags_query", datadog.ParameterToString(*r.tagsQuery, ""))
	}
	if r.metricsQuery != nil {
		localVarQueryParams.Add("metrics_query", datadog.ParameterToString(*r.metricsQuery, ""))
	}
	if r.limit != nil {
		localVarQueryParams.Add("limit", datadog.ParameterToString(*r.limit, ""))
	}
	if r.offset != nil {
		localVarQueryParams.Add("offset", datadog.ParameterToString(*r.offset, ""))
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

type apiSearchSLORequest struct {
	ctx           _context.Context
	query         *string
	pageSize      *int64
	pageNumber    *int64
	includeFacets *bool
}

// SearchSLOOptionalParameters holds optional parameters for SearchSLO.
type SearchSLOOptionalParameters struct {
	Query         *string
	PageSize      *int64
	PageNumber    *int64
	IncludeFacets *bool
}

// NewSearchSLOOptionalParameters creates an empty struct for parameters.
func NewSearchSLOOptionalParameters() *SearchSLOOptionalParameters {
	this := SearchSLOOptionalParameters{}
	return &this
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithQuery(query string) *SearchSLOOptionalParameters {
	r.Query = &query
	return r
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithPageSize(pageSize int64) *SearchSLOOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithPageNumber(pageNumber int64) *SearchSLOOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithIncludeFacets sets the corresponding parameter name and returns the struct.
func (r *SearchSLOOptionalParameters) WithIncludeFacets(includeFacets bool) *SearchSLOOptionalParameters {
	r.IncludeFacets = &includeFacets
	return r
}

func (a *ServiceLevelObjectivesApi) buildSearchSLORequest(ctx _context.Context, o ...SearchSLOOptionalParameters) (apiSearchSLORequest, error) {
	req := apiSearchSLORequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SearchSLOOptionalParameters is allowed")
	}

	if o != nil {
		req.query = o[0].Query
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.includeFacets = o[0].IncludeFacets
	}
	return req, nil
}

// SearchSLO Search for SLOs.
// Get a list of service level objective objects for your organization.
func (a *ServiceLevelObjectivesApi) SearchSLO(ctx _context.Context, o ...SearchSLOOptionalParameters) (SearchSLOResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchSLORequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SearchSLOResponse
		return localVarReturnValue, nil, err
	}

	return a.searchSLOExecute(req)
}

// searchSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) searchSLOExecute(r apiSearchSLORequest) (SearchSLOResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SearchSLOResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.SearchSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/search"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.query != nil {
		localVarQueryParams.Add("query", datadog.ParameterToString(*r.query, ""))
	}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*r.pageNumber, ""))
	}
	if r.includeFacets != nil {
		localVarQueryParams.Add("include_facets", datadog.ParameterToString(*r.includeFacets, ""))
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

type apiUpdateSLORequest struct {
	ctx   _context.Context
	sloId string
	body  *ServiceLevelObjective
}

func (a *ServiceLevelObjectivesApi) buildUpdateSLORequest(ctx _context.Context, sloId string, body ServiceLevelObjective) (apiUpdateSLORequest, error) {
	req := apiUpdateSLORequest{
		ctx:   ctx,
		sloId: sloId,
		body:  &body,
	}
	return req, nil
}

// UpdateSLO Update an SLO.
// Update the specified service level objective object.
func (a *ServiceLevelObjectivesApi) UpdateSLO(ctx _context.Context, sloId string, body ServiceLevelObjective) (SLOListResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateSLORequest(ctx, sloId, body)
	if err != nil {
		var localVarReturnValue SLOListResponse
		return localVarReturnValue, nil, err
	}

	return a.updateSLOExecute(req)
}

// updateSLOExecute executes the request.
func (a *ServiceLevelObjectivesApi) updateSLOExecute(r apiUpdateSLORequest) (SLOListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SLOListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.ServiceLevelObjectivesApi.UpdateSLO")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/slo/{slo_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"slo_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.sloId, "")), -1)

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

// NewServiceLevelObjectivesApi Returns NewServiceLevelObjectivesApi.
func NewServiceLevelObjectivesApi(client *datadog.APIClient) *ServiceLevelObjectivesApi {
	return &ServiceLevelObjectivesApi{
		Client: client,
	}
}
