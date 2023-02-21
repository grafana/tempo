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

// IncidentTeamsApi service type
type IncidentTeamsApi datadog.Service

type apiCreateIncidentTeamRequest struct {
	ctx  _context.Context
	body *IncidentTeamCreateRequest
}

func (a *IncidentTeamsApi) buildCreateIncidentTeamRequest(ctx _context.Context, body IncidentTeamCreateRequest) (apiCreateIncidentTeamRequest, error) {
	req := apiCreateIncidentTeamRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateIncidentTeam Create a new incident team.
// Creates a new incident team.
func (a *IncidentTeamsApi) CreateIncidentTeam(ctx _context.Context, body IncidentTeamCreateRequest) (IncidentTeamResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateIncidentTeamRequest(ctx, body)
	if err != nil {
		var localVarReturnValue IncidentTeamResponse
		return localVarReturnValue, nil, err
	}

	return a.createIncidentTeamExecute(req)
}

// createIncidentTeamExecute executes the request.
func (a *IncidentTeamsApi) createIncidentTeamExecute(r apiCreateIncidentTeamRequest) (IncidentTeamResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IncidentTeamResponse
	)

	operationId := "v2.CreateIncidentTeam"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentTeamsApi.CreateIncidentTeam")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/teams"

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

type apiDeleteIncidentTeamRequest struct {
	ctx    _context.Context
	teamId string
}

func (a *IncidentTeamsApi) buildDeleteIncidentTeamRequest(ctx _context.Context, teamId string) (apiDeleteIncidentTeamRequest, error) {
	req := apiDeleteIncidentTeamRequest{
		ctx:    ctx,
		teamId: teamId,
	}
	return req, nil
}

// DeleteIncidentTeam Delete an existing incident team.
// Deletes an existing incident team.
func (a *IncidentTeamsApi) DeleteIncidentTeam(ctx _context.Context, teamId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteIncidentTeamRequest(ctx, teamId)
	if err != nil {
		return nil, err
	}

	return a.deleteIncidentTeamExecute(req)
}

// deleteIncidentTeamExecute executes the request.
func (a *IncidentTeamsApi) deleteIncidentTeamExecute(r apiDeleteIncidentTeamRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	operationId := "v2.DeleteIncidentTeam"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentTeamsApi.DeleteIncidentTeam")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/teams/{team_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"team_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.teamId, "")), -1)

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

type apiGetIncidentTeamRequest struct {
	ctx     _context.Context
	teamId  string
	include *IncidentRelatedObject
}

// GetIncidentTeamOptionalParameters holds optional parameters for GetIncidentTeam.
type GetIncidentTeamOptionalParameters struct {
	Include *IncidentRelatedObject
}

// NewGetIncidentTeamOptionalParameters creates an empty struct for parameters.
func NewGetIncidentTeamOptionalParameters() *GetIncidentTeamOptionalParameters {
	this := GetIncidentTeamOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *GetIncidentTeamOptionalParameters) WithInclude(include IncidentRelatedObject) *GetIncidentTeamOptionalParameters {
	r.Include = &include
	return r
}

func (a *IncidentTeamsApi) buildGetIncidentTeamRequest(ctx _context.Context, teamId string, o ...GetIncidentTeamOptionalParameters) (apiGetIncidentTeamRequest, error) {
	req := apiGetIncidentTeamRequest{
		ctx:    ctx,
		teamId: teamId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetIncidentTeamOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
	}
	return req, nil
}

// GetIncidentTeam Get details of an incident team.
// Get details of an incident team. If the `include[users]` query parameter is provided,
// the included attribute will contain the users related to these incident teams.
func (a *IncidentTeamsApi) GetIncidentTeam(ctx _context.Context, teamId string, o ...GetIncidentTeamOptionalParameters) (IncidentTeamResponse, *_nethttp.Response, error) {
	req, err := a.buildGetIncidentTeamRequest(ctx, teamId, o...)
	if err != nil {
		var localVarReturnValue IncidentTeamResponse
		return localVarReturnValue, nil, err
	}

	return a.getIncidentTeamExecute(req)
}

// getIncidentTeamExecute executes the request.
func (a *IncidentTeamsApi) getIncidentTeamExecute(r apiGetIncidentTeamRequest) (IncidentTeamResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentTeamResponse
	)

	operationId := "v2.GetIncidentTeam"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentTeamsApi.GetIncidentTeam")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/teams/{team_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"team_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.teamId, "")), -1)

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

type apiListIncidentTeamsRequest struct {
	ctx        _context.Context
	include    *IncidentRelatedObject
	pageSize   *int64
	pageOffset *int64
	filter     *string
}

// ListIncidentTeamsOptionalParameters holds optional parameters for ListIncidentTeams.
type ListIncidentTeamsOptionalParameters struct {
	Include    *IncidentRelatedObject
	PageSize   *int64
	PageOffset *int64
	Filter     *string
}

// NewListIncidentTeamsOptionalParameters creates an empty struct for parameters.
func NewListIncidentTeamsOptionalParameters() *ListIncidentTeamsOptionalParameters {
	this := ListIncidentTeamsOptionalParameters{}
	return &this
}

// WithInclude sets the corresponding parameter name and returns the struct.
func (r *ListIncidentTeamsOptionalParameters) WithInclude(include IncidentRelatedObject) *ListIncidentTeamsOptionalParameters {
	r.Include = &include
	return r
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListIncidentTeamsOptionalParameters) WithPageSize(pageSize int64) *ListIncidentTeamsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageOffset sets the corresponding parameter name and returns the struct.
func (r *ListIncidentTeamsOptionalParameters) WithPageOffset(pageOffset int64) *ListIncidentTeamsOptionalParameters {
	r.PageOffset = &pageOffset
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListIncidentTeamsOptionalParameters) WithFilter(filter string) *ListIncidentTeamsOptionalParameters {
	r.Filter = &filter
	return r
}

func (a *IncidentTeamsApi) buildListIncidentTeamsRequest(ctx _context.Context, o ...ListIncidentTeamsOptionalParameters) (apiListIncidentTeamsRequest, error) {
	req := apiListIncidentTeamsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListIncidentTeamsOptionalParameters is allowed")
	}

	if o != nil {
		req.include = o[0].Include
		req.pageSize = o[0].PageSize
		req.pageOffset = o[0].PageOffset
		req.filter = o[0].Filter
	}
	return req, nil
}

// ListIncidentTeams Get a list of all incident teams.
// Get all incident teams for the requesting user's organization. If the `include[users]` query parameter is provided, the included attribute will contain the users related to these incident teams.
func (a *IncidentTeamsApi) ListIncidentTeams(ctx _context.Context, o ...ListIncidentTeamsOptionalParameters) (IncidentTeamsResponse, *_nethttp.Response, error) {
	req, err := a.buildListIncidentTeamsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue IncidentTeamsResponse
		return localVarReturnValue, nil, err
	}

	return a.listIncidentTeamsExecute(req)
}

// listIncidentTeamsExecute executes the request.
func (a *IncidentTeamsApi) listIncidentTeamsExecute(r apiListIncidentTeamsRequest) (IncidentTeamsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue IncidentTeamsResponse
	)

	operationId := "v2.ListIncidentTeams"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentTeamsApi.ListIncidentTeams")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/teams"

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

type apiUpdateIncidentTeamRequest struct {
	ctx    _context.Context
	teamId string
	body   *IncidentTeamUpdateRequest
}

func (a *IncidentTeamsApi) buildUpdateIncidentTeamRequest(ctx _context.Context, teamId string, body IncidentTeamUpdateRequest) (apiUpdateIncidentTeamRequest, error) {
	req := apiUpdateIncidentTeamRequest{
		ctx:    ctx,
		teamId: teamId,
		body:   &body,
	}
	return req, nil
}

// UpdateIncidentTeam Update an existing incident team.
// Updates an existing incident team. Only provide the attributes which should be updated as this request is a partial update.
func (a *IncidentTeamsApi) UpdateIncidentTeam(ctx _context.Context, teamId string, body IncidentTeamUpdateRequest) (IncidentTeamResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateIncidentTeamRequest(ctx, teamId, body)
	if err != nil {
		var localVarReturnValue IncidentTeamResponse
		return localVarReturnValue, nil, err
	}

	return a.updateIncidentTeamExecute(req)
}

// updateIncidentTeamExecute executes the request.
func (a *IncidentTeamsApi) updateIncidentTeamExecute(r apiUpdateIncidentTeamRequest) (IncidentTeamResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue IncidentTeamResponse
	)

	operationId := "v2.UpdateIncidentTeam"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.IncidentTeamsApi.UpdateIncidentTeam")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/teams/{team_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"team_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.teamId, "")), -1)

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

// NewIncidentTeamsApi Returns NewIncidentTeamsApi.
func NewIncidentTeamsApi(client *datadog.APIClient) *IncidentTeamsApi {
	return &IncidentTeamsApi{
		Client: client,
	}
}
