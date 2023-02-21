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

// IncidentsApi service type
type IncidentsApi datadog.Service

type apiCreateIncidentRequest struct {
	ctx  _context.Context
	body *IncidentCreateRequest
}

func (a *IncidentsApi) buildCreateIncidentRequest(ctx _context.Context, body IncidentCreateRequest) (apiCreateIncidentRequest, error) {
	req := apiCreateIncidentRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateIncident Create an incident.
// Create an incident.
func (a *IncidentsApi) CreateIncident(ctx _context.Context, body IncidentCreateRequest) (IncidentResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateIncidentRequest(ctx, body)
	if err != nil {
		var localVarReturnValue IncidentResponse
		return localVarReturnValue, nil, err
	}

	return a.createIncidentExecute(req)
}

// createIncidentExecute executes the request.
func (a *IncidentsApi) createIncidentExecute(r apiCreateIncidentRequest) (IncidentResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IncidentResponse
	)

	operationId := "v2.CreateIncident"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.CreateIncident")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents"

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

type apiDeleteIncidentRequest struct {
	ctx        _context.Context
	incidentId string
}

func (a *IncidentsApi) buildDeleteIncidentRequest(ctx _context.Context, incidentId string) (apiDeleteIncidentRequest, error) {
	req := apiDeleteIncidentRequest{
		ctx:        ctx,
		incidentId: incidentId,
	}
	return req, nil
}

// DeleteIncident Delete an existing incident.
// Deletes an existing incident from the users organization.
func (a *IncidentsApi) DeleteIncident(ctx _context.Context, incidentId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteIncidentRequest(ctx, incidentId)
	if err != nil {
		return nil, err
	}

	return a.deleteIncidentExecute(req)
}

// deleteIncidentExecute executes the request.
func (a *IncidentsApi) deleteIncidentExecute(r apiDeleteIncidentRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	operationId := "v2.DeleteIncident"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.DeleteIncident")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/{incident_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"incident_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.incidentId, "")), -1)

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

type apiGetIncidentRequest struct {
	ctx        _context.Context
	incidentId string
	include    *[]IncidentRelatedObject
}

// GetIncidentOptionalParameters holds optional parameters for GetIncident.
type GetIncidentOptionalParameters struct {
	Include *[]IncidentRelatedObject
}

// NewGetIncidentOptionalParameters creates an empty struct for parameters.
func NewGetIncidentOptionalParameters() *GetIncidentOptionalParameters {
	this := GetIncidentOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *GetIncidentOptionalParameters) WithInclude(include []IncidentRelatedObject) *GetIncidentOptionalParameters {
	r.Include = &include
	return r
}

func (a *IncidentsApi) buildGetIncidentRequest(ctx _context.Context, incidentId string, o ...GetIncidentOptionalParameters) (apiGetIncidentRequest, error) {
	req := apiGetIncidentRequest{
		ctx:        ctx,
		incidentId: incidentId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetIncidentOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// GetIncident Get the details of an incident.
// Get the details of an incident by `incident_id`.
func (a *IncidentsApi) GetIncident(ctx _context.Context, incidentId string, o ...GetIncidentOptionalParameters) (IncidentResponse, *_nethttp.Response, error) {
	req, err := a.buildGetIncidentRequest(ctx, incidentId, o...)
	if err != nil {
		var localVarReturnValue IncidentResponse
		return localVarReturnValue, nil, err
	}

	return a.getIncidentExecute(req)
}

// getIncidentExecute executes the request.
func (a *IncidentsApi) getIncidentExecute(r apiGetIncidentRequest) (IncidentResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentResponse
	)

	operationId := "v2.GetIncident"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.GetIncident")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/{incident_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"incident_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.incidentId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, "csv"))
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

type apiListIncidentAttachmentsRequest struct {
	ctx                  _context.Context
	incidentId           string
	include              *[]IncidentAttachmentRelatedObject
	filterAttachmentType *[]IncidentAttachmentAttachmentType
}

// ListIncidentAttachmentsOptionalParameters holds optional parameters for ListIncidentAttachments.
type ListIncidentAttachmentsOptionalParameters struct {
	Include              *[]IncidentAttachmentRelatedObject
	FilterAttachmentType *[]IncidentAttachmentAttachmentType
}

// NewListIncidentAttachmentsOptionalParameters creates an empty struct for parameters.
func NewListIncidentAttachmentsOptionalParameters() *ListIncidentAttachmentsOptionalParameters {
	this := ListIncidentAttachmentsOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *ListIncidentAttachmentsOptionalParameters) WithInclude(include []IncidentAttachmentRelatedObject) *ListIncidentAttachmentsOptionalParameters {
	r.Include = &include
	return r
}

// WithFilterAttachmentType sets the corresponding parameter name and returns the struct.
func (r *ListIncidentAttachmentsOptionalParameters) WithFilterAttachmentType(filterAttachmentType []IncidentAttachmentAttachmentType) *ListIncidentAttachmentsOptionalParameters {
	r.FilterAttachmentType = &filterAttachmentType
	return r
}

func (a *IncidentsApi) buildListIncidentAttachmentsRequest(ctx _context.Context, incidentId string, o ...ListIncidentAttachmentsOptionalParameters) (apiListIncidentAttachmentsRequest, error) {
	req := apiListIncidentAttachmentsRequest{
		ctx:        ctx,
		incidentId: incidentId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListIncidentAttachmentsOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
		req.filterAttachmentType = o[0].FilterAttachmentType
	}
	return req, nil
}

// ListIncidentAttachments Get a list of attachments.
// Get all attachments for a given incident.
func (a *IncidentsApi) ListIncidentAttachments(ctx _context.Context, incidentId string, o ...ListIncidentAttachmentsOptionalParameters) (IncidentAttachmentsResponse, *_nethttp.Response, error) {
	req, err := a.buildListIncidentAttachmentsRequest(ctx, incidentId, o...)
	if err != nil {
		var localVarReturnValue IncidentAttachmentsResponse
		return localVarReturnValue, nil, err
	}

	return a.listIncidentAttachmentsExecute(req)
}

// listIncidentAttachmentsExecute executes the request.
func (a *IncidentsApi) listIncidentAttachmentsExecute(r apiListIncidentAttachmentsRequest) (IncidentAttachmentsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentAttachmentsResponse
	)

	operationId := "v2.ListIncidentAttachments"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.ListIncidentAttachments")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/{incident_id}/attachments"
	localVarPath = strings.Replace(localVarPath, "{"+"incident_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.incidentId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, "csv"))
	}
	if r.filterAttachmentType != nil {
		localVarQueryParams.Add("filter[attachment_type]", datadog.ParameterToString(*r.filterAttachmentType, "csv"))
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

type apiListIncidentsRequest struct {
	ctx        _context.Context
	include    *[]IncidentRelatedObject
	pageSize   *int64
	pageOffset *int64
}

// ListIncidentsOptionalParameters holds optional parameters for ListIncidents.
type ListIncidentsOptionalParameters struct {
	Include    *[]IncidentRelatedObject
	PageSize   *int64
	PageOffset *int64
}

// NewListIncidentsOptionalParameters creates an empty struct for parameters.
func NewListIncidentsOptionalParameters() *ListIncidentsOptionalParameters {
	this := ListIncidentsOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *ListIncidentsOptionalParameters) WithInclude(include []IncidentRelatedObject) *ListIncidentsOptionalParameters {
	r.Include = &include
	return r
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListIncidentsOptionalParameters) WithPageSize(pageSize int64) *ListIncidentsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageOffset sets the corresponding parameter name and returns the struct.
func (r *ListIncidentsOptionalParameters) WithPageOffset(pageOffset int64) *ListIncidentsOptionalParameters {
	r.PageOffset = &pageOffset
	return r
}

func (a *IncidentsApi) buildListIncidentsRequest(ctx _context.Context, o ...ListIncidentsOptionalParameters) (apiListIncidentsRequest, error) {
	req := apiListIncidentsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListIncidentsOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
		req.pageSize = o[0].PageSize
		req.pageOffset = o[0].PageOffset
	}
	return req, nil
}

// ListIncidents Get a list of incidents.
// Get all incidents for the user's organization.
func (a *IncidentsApi) ListIncidents(ctx _context.Context, o ...ListIncidentsOptionalParameters) (IncidentsResponse, *_nethttp.Response, error) {
	req, err := a.buildListIncidentsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue IncidentsResponse
		return localVarReturnValue, nil, err
	}

	return a.listIncidentsExecute(req)
}

// ListIncidentsWithPagination provides a paginated version of ListIncidents returning a channel with all items.
func (a *IncidentsApi) ListIncidentsWithPagination(ctx _context.Context, o ...ListIncidentsOptionalParameters) (<-chan datadog.PaginationResult[IncidentResponseData], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int64(10)
	if len(o) == 0 {
		o = append(o, ListIncidentsOptionalParameters{})
	}
	if o[0].PageSize != nil {
		pageSize_ = *o[0].PageSize
	}
	o[0].PageSize = &pageSize_

	items := make(chan datadog.PaginationResult[IncidentResponseData], pageSize_)
	go func() {
		for {
			req, err := a.buildListIncidentsRequest(ctx, o...)
			if err != nil {
				var returnItem IncidentResponseData
				items <- datadog.PaginationResult[IncidentResponseData]{returnItem, err}
				break
			}

			resp, _, err := a.listIncidentsExecute(req)
			if err != nil {
				var returnItem IncidentResponseData
				items <- datadog.PaginationResult[IncidentResponseData]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[IncidentResponseData]{item, nil}:
				case <-ctx.Done():
					close(items)
					return
				}
			}
			if len(results) < int(pageSize_) {
				break
			}
			if o[0].PageOffset == nil {
				o[0].PageOffset = &pageSize_
			} else {
				pageOffset_ := *o[0].PageOffset + pageSize_
				o[0].PageOffset = &pageOffset_
			}
		}
		close(items)
	}()
	return items, cancel
}

// listIncidentsExecute executes the request.
func (a *IncidentsApi) listIncidentsExecute(r apiListIncidentsRequest) (IncidentsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentsResponse
	)

	operationId := "v2.ListIncidents"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.ListIncidents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, "csv"))
	}
	if r.pageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageOffset != nil {
		localVarQueryParams.Add("page[offset]", datadog.ParameterToString(*r.pageOffset, ""))
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

type apiSearchIncidentsRequest struct {
	ctx     _context.Context
	query   *string
	include *IncidentRelatedObject
	sort    *IncidentSearchSortOrder
}

// SearchIncidentsOptionalParameters holds optional parameters for SearchIncidents.
type SearchIncidentsOptionalParameters struct {
	Include *IncidentRelatedObject
	Sort    *IncidentSearchSortOrder
}

// NewSearchIncidentsOptionalParameters creates an empty struct for parameters.
func NewSearchIncidentsOptionalParameters() *SearchIncidentsOptionalParameters {
	this := SearchIncidentsOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *SearchIncidentsOptionalParameters) WithInclude(include IncidentRelatedObject) *SearchIncidentsOptionalParameters {
	r.Include = &include
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *SearchIncidentsOptionalParameters) WithSort(sort IncidentSearchSortOrder) *SearchIncidentsOptionalParameters {
	r.Sort = &sort
	return r
}

func (a *IncidentsApi) buildSearchIncidentsRequest(ctx _context.Context, query string, o ...SearchIncidentsOptionalParameters) (apiSearchIncidentsRequest, error) {
	req := apiSearchIncidentsRequest{
		ctx:   ctx,
		query: &query,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SearchIncidentsOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
		req.sort = o[0].Sort
	}
	return req, nil
}

// SearchIncidents Search for incidents.
// Search for incidents matching a certain query.
func (a *IncidentsApi) SearchIncidents(ctx _context.Context, query string, o ...SearchIncidentsOptionalParameters) (IncidentSearchResponse, *_nethttp.Response, error) {
	req, err := a.buildSearchIncidentsRequest(ctx, query, o...)
	if err != nil {
		var localVarReturnValue IncidentSearchResponse
		return localVarReturnValue, nil, err
	}

	return a.searchIncidentsExecute(req)
}

// searchIncidentsExecute executes the request.
func (a *IncidentsApi) searchIncidentsExecute(r apiSearchIncidentsRequest) (IncidentSearchResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentSearchResponse
	)

	operationId := "v2.SearchIncidents"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.SearchIncidents")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/search"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.query == nil {
		return localVarReturnValue, nil, datadog.ReportError("query is required and must be specified")
	}
	localVarQueryParams.Add("query", datadog.ParameterToString(*r.query, ""))
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, ""))
	}
	if r.sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*r.sort, ""))
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

type apiUpdateIncidentRequest struct {
	ctx        _context.Context
	incidentId string
	body       *IncidentUpdateRequest
	include    *[]IncidentRelatedObject
}

// UpdateIncidentOptionalParameters holds optional parameters for UpdateIncident.
type UpdateIncidentOptionalParameters struct {
	Include *[]IncidentRelatedObject
}

// NewUpdateIncidentOptionalParameters creates an empty struct for parameters.
func NewUpdateIncidentOptionalParameters() *UpdateIncidentOptionalParameters {
	this := UpdateIncidentOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *UpdateIncidentOptionalParameters) WithInclude(include []IncidentRelatedObject) *UpdateIncidentOptionalParameters {
	r.Include = &include
	return r
}

func (a *IncidentsApi) buildUpdateIncidentRequest(ctx _context.Context, incidentId string, body IncidentUpdateRequest, o ...UpdateIncidentOptionalParameters) (apiUpdateIncidentRequest, error) {
	req := apiUpdateIncidentRequest{
		ctx:        ctx,
		incidentId: incidentId,
		body:       &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type UpdateIncidentOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// UpdateIncident Update an existing incident.
// Updates an incident. Provide only the attributes that should be updated as this request is a partial update.
func (a *IncidentsApi) UpdateIncident(ctx _context.Context, incidentId string, body IncidentUpdateRequest, o ...UpdateIncidentOptionalParameters) (IncidentResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateIncidentRequest(ctx, incidentId, body, o...)
	if err != nil {
		var localVarReturnValue IncidentResponse
		return localVarReturnValue, nil, err
	}

	return a.updateIncidentExecute(req)
}

// updateIncidentExecute executes the request.
func (a *IncidentsApi) updateIncidentExecute(r apiUpdateIncidentRequest) (IncidentResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue IncidentResponse
	)

	operationId := "v2.UpdateIncident"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.UpdateIncident")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/{incident_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"incident_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.incidentId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, "csv"))
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

type apiUpdateIncidentAttachmentsRequest struct {
	ctx        _context.Context
	incidentId string
	body       *IncidentAttachmentUpdateRequest
	include    *[]IncidentAttachmentRelatedObject
}

// UpdateIncidentAttachmentsOptionalParameters holds optional parameters for UpdateIncidentAttachments.
type UpdateIncidentAttachmentsOptionalParameters struct {
	Include *[]IncidentAttachmentRelatedObject
}

// NewUpdateIncidentAttachmentsOptionalParameters creates an empty struct for parameters.
func NewUpdateIncidentAttachmentsOptionalParameters() *UpdateIncidentAttachmentsOptionalParameters {
	this := UpdateIncidentAttachmentsOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *UpdateIncidentAttachmentsOptionalParameters) WithInclude(include []IncidentAttachmentRelatedObject) *UpdateIncidentAttachmentsOptionalParameters {
	r.Include = &include
	return r
}

func (a *IncidentsApi) buildUpdateIncidentAttachmentsRequest(ctx _context.Context, incidentId string, body IncidentAttachmentUpdateRequest, o ...UpdateIncidentAttachmentsOptionalParameters) (apiUpdateIncidentAttachmentsRequest, error) {
	req := apiUpdateIncidentAttachmentsRequest{
		ctx:        ctx,
		incidentId: incidentId,
		body:       &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type UpdateIncidentAttachmentsOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// UpdateIncidentAttachments Create, update, and delete incident attachments.
// The bulk update endpoint for creating, updating, and deleting attachments for a given incident.
func (a *IncidentsApi) UpdateIncidentAttachments(ctx _context.Context, incidentId string, body IncidentAttachmentUpdateRequest, o ...UpdateIncidentAttachmentsOptionalParameters) (IncidentAttachmentUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateIncidentAttachmentsRequest(ctx, incidentId, body, o...)
	if err != nil {
		var localVarReturnValue IncidentAttachmentUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.updateIncidentAttachmentsExecute(req)
}

// updateIncidentAttachmentsExecute executes the request.
func (a *IncidentsApi) updateIncidentAttachmentsExecute(r apiUpdateIncidentAttachmentsRequest) (IncidentAttachmentUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue IncidentAttachmentUpdateResponse
	)

	operationId := "v2.UpdateIncidentAttachments"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentsApi.UpdateIncidentAttachments")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/incidents/{incident_id}/attachments"
	localVarPath = strings.Replace(localVarPath, "{"+"incident_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.incidentId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.include != nil {
		localVarQueryParams.Add("include", datadog.ParameterToString(*r.include, "csv"))
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

// NewIncidentsApi Returns NewIncidentsApi.
func NewIncidentsApi(client *datadog.APIClient) *IncidentsApi {
	return &IncidentsApi{
		Client: client,
	}
}
