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

// LogsArchivesApi service type
type LogsArchivesApi datadog.Service

type apiAddReadRoleToArchiveRequest struct {
	ctx       _context.Context
	archiveId string
	body      *RelationshipToRole
}

func (a *LogsArchivesApi) buildAddReadRoleToArchiveRequest(ctx _context.Context, archiveId string, body RelationshipToRole) (apiAddReadRoleToArchiveRequest, error) {
	req := apiAddReadRoleToArchiveRequest{
		ctx:       ctx,
		archiveId: archiveId,
		body:      &body,
	}
	return req, nil
}

// AddReadRoleToArchive Grant role to an archive.
// Adds a read role to an archive. ([Roles API](https://docs.datadoghq.com/api/v2/roles/))
func (a *LogsArchivesApi) AddReadRoleToArchive(ctx _context.Context, archiveId string, body RelationshipToRole) (*_nethttp.Response, error) {
	req, err := a.buildAddReadRoleToArchiveRequest(ctx, archiveId, body)
	if err != nil {
		return nil, err
	}

	return a.addReadRoleToArchiveExecute(req)
}

// addReadRoleToArchiveExecute executes the request.
func (a *LogsArchivesApi) addReadRoleToArchiveExecute(r apiAddReadRoleToArchiveRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodPost
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.AddReadRoleToArchive")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}/readers"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiCreateLogsArchiveRequest struct {
	ctx  _context.Context
	body *LogsArchiveCreateRequest
}

func (a *LogsArchivesApi) buildCreateLogsArchiveRequest(ctx _context.Context, body LogsArchiveCreateRequest) (apiCreateLogsArchiveRequest, error) {
	req := apiCreateLogsArchiveRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateLogsArchive Create an archive.
// Create an archive in your organization.
func (a *LogsArchivesApi) CreateLogsArchive(ctx _context.Context, body LogsArchiveCreateRequest) (LogsArchive, *_nethttp.Response, error) {
	req, err := a.buildCreateLogsArchiveRequest(ctx, body)
	if err != nil {
		var localVarReturnValue LogsArchive
		return localVarReturnValue, nil, err
	}

	return a.createLogsArchiveExecute(req)
}

// createLogsArchiveExecute executes the request.
func (a *LogsArchivesApi) createLogsArchiveExecute(r apiCreateLogsArchiveRequest) (LogsArchive, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsArchive
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.CreateLogsArchive")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives"

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

type apiDeleteLogsArchiveRequest struct {
	ctx       _context.Context
	archiveId string
}

func (a *LogsArchivesApi) buildDeleteLogsArchiveRequest(ctx _context.Context, archiveId string) (apiDeleteLogsArchiveRequest, error) {
	req := apiDeleteLogsArchiveRequest{
		ctx:       ctx,
		archiveId: archiveId,
	}
	return req, nil
}

// DeleteLogsArchive Delete an archive.
// Delete a given archive from your organization.
func (a *LogsArchivesApi) DeleteLogsArchive(ctx _context.Context, archiveId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteLogsArchiveRequest(ctx, archiveId)
	if err != nil {
		return nil, err
	}

	return a.deleteLogsArchiveExecute(req)
}

// deleteLogsArchiveExecute executes the request.
func (a *LogsArchivesApi) deleteLogsArchiveExecute(r apiDeleteLogsArchiveRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.DeleteLogsArchive")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiGetLogsArchiveRequest struct {
	ctx       _context.Context
	archiveId string
}

func (a *LogsArchivesApi) buildGetLogsArchiveRequest(ctx _context.Context, archiveId string) (apiGetLogsArchiveRequest, error) {
	req := apiGetLogsArchiveRequest{
		ctx:       ctx,
		archiveId: archiveId,
	}
	return req, nil
}

// GetLogsArchive Get an archive.
// Get a specific archive from your organization.
func (a *LogsArchivesApi) GetLogsArchive(ctx _context.Context, archiveId string) (LogsArchive, *_nethttp.Response, error) {
	req, err := a.buildGetLogsArchiveRequest(ctx, archiveId)
	if err != nil {
		var localVarReturnValue LogsArchive
		return localVarReturnValue, nil, err
	}

	return a.getLogsArchiveExecute(req)
}

// getLogsArchiveExecute executes the request.
func (a *LogsArchivesApi) getLogsArchiveExecute(r apiGetLogsArchiveRequest) (LogsArchive, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsArchive
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.GetLogsArchive")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiGetLogsArchiveOrderRequest struct {
	ctx _context.Context
}

func (a *LogsArchivesApi) buildGetLogsArchiveOrderRequest(ctx _context.Context) (apiGetLogsArchiveOrderRequest, error) {
	req := apiGetLogsArchiveOrderRequest{
		ctx: ctx,
	}
	return req, nil
}

// GetLogsArchiveOrder Get archive order.
// Get the current order of your archives.
// This endpoint takes no JSON arguments.
func (a *LogsArchivesApi) GetLogsArchiveOrder(ctx _context.Context) (LogsArchiveOrder, *_nethttp.Response, error) {
	req, err := a.buildGetLogsArchiveOrderRequest(ctx)
	if err != nil {
		var localVarReturnValue LogsArchiveOrder
		return localVarReturnValue, nil, err
	}

	return a.getLogsArchiveOrderExecute(req)
}

// getLogsArchiveOrderExecute executes the request.
func (a *LogsArchivesApi) getLogsArchiveOrderExecute(r apiGetLogsArchiveOrderRequest) (LogsArchiveOrder, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsArchiveOrder
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.GetLogsArchiveOrder")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archive-order"

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

type apiListArchiveReadRolesRequest struct {
	ctx       _context.Context
	archiveId string
}

func (a *LogsArchivesApi) buildListArchiveReadRolesRequest(ctx _context.Context, archiveId string) (apiListArchiveReadRolesRequest, error) {
	req := apiListArchiveReadRolesRequest{
		ctx:       ctx,
		archiveId: archiveId,
	}
	return req, nil
}

// ListArchiveReadRoles List read roles for an archive.
// Returns all read roles a given archive is restricted to.
func (a *LogsArchivesApi) ListArchiveReadRoles(ctx _context.Context, archiveId string) (RolesResponse, *_nethttp.Response, error) {
	req, err := a.buildListArchiveReadRolesRequest(ctx, archiveId)
	if err != nil {
		var localVarReturnValue RolesResponse
		return localVarReturnValue, nil, err
	}

	return a.listArchiveReadRolesExecute(req)
}

// listArchiveReadRolesExecute executes the request.
func (a *LogsArchivesApi) listArchiveReadRolesExecute(r apiListArchiveReadRolesRequest) (RolesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RolesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.ListArchiveReadRoles")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}/readers"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiListLogsArchivesRequest struct {
	ctx _context.Context
}

func (a *LogsArchivesApi) buildListLogsArchivesRequest(ctx _context.Context) (apiListLogsArchivesRequest, error) {
	req := apiListLogsArchivesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListLogsArchives Get all archives.
// Get the list of configured logs archives with their definitions.
func (a *LogsArchivesApi) ListLogsArchives(ctx _context.Context) (LogsArchives, *_nethttp.Response, error) {
	req, err := a.buildListLogsArchivesRequest(ctx)
	if err != nil {
		var localVarReturnValue LogsArchives
		return localVarReturnValue, nil, err
	}

	return a.listLogsArchivesExecute(req)
}

// listLogsArchivesExecute executes the request.
func (a *LogsArchivesApi) listLogsArchivesExecute(r apiListLogsArchivesRequest) (LogsArchives, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsArchives
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.ListLogsArchives")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives"

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

type apiRemoveRoleFromArchiveRequest struct {
	ctx       _context.Context
	archiveId string
	body      *RelationshipToRole
}

func (a *LogsArchivesApi) buildRemoveRoleFromArchiveRequest(ctx _context.Context, archiveId string, body RelationshipToRole) (apiRemoveRoleFromArchiveRequest, error) {
	req := apiRemoveRoleFromArchiveRequest{
		ctx:       ctx,
		archiveId: archiveId,
		body:      &body,
	}
	return req, nil
}

// RemoveRoleFromArchive Revoke role from an archive.
// Removes a role from an archive. ([Roles API](https://docs.datadoghq.com/api/v2/roles/))
func (a *LogsArchivesApi) RemoveRoleFromArchive(ctx _context.Context, archiveId string, body RelationshipToRole) (*_nethttp.Response, error) {
	req, err := a.buildRemoveRoleFromArchiveRequest(ctx, archiveId, body)
	if err != nil {
		return nil, err
	}

	return a.removeRoleFromArchiveExecute(req)
}

// removeRoleFromArchiveExecute executes the request.
func (a *LogsArchivesApi) removeRoleFromArchiveExecute(r apiRemoveRoleFromArchiveRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.RemoveRoleFromArchive")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}/readers"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiUpdateLogsArchiveRequest struct {
	ctx       _context.Context
	archiveId string
	body      *LogsArchiveCreateRequest
}

func (a *LogsArchivesApi) buildUpdateLogsArchiveRequest(ctx _context.Context, archiveId string, body LogsArchiveCreateRequest) (apiUpdateLogsArchiveRequest, error) {
	req := apiUpdateLogsArchiveRequest{
		ctx:       ctx,
		archiveId: archiveId,
		body:      &body,
	}
	return req, nil
}

// UpdateLogsArchive Update an archive.
// Update a given archive configuration.
//
// **Note**: Using this method updates your archive configuration by **replacing**
// your current configuration with the new one sent to your Datadog organization.
func (a *LogsArchivesApi) UpdateLogsArchive(ctx _context.Context, archiveId string, body LogsArchiveCreateRequest) (LogsArchive, *_nethttp.Response, error) {
	req, err := a.buildUpdateLogsArchiveRequest(ctx, archiveId, body)
	if err != nil {
		var localVarReturnValue LogsArchive
		return localVarReturnValue, nil, err
	}

	return a.updateLogsArchiveExecute(req)
}

// updateLogsArchiveExecute executes the request.
func (a *LogsArchivesApi) updateLogsArchiveExecute(r apiUpdateLogsArchiveRequest) (LogsArchive, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue LogsArchive
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.UpdateLogsArchive")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archives/{archive_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"archive_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.archiveId, "")), -1)

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

type apiUpdateLogsArchiveOrderRequest struct {
	ctx  _context.Context
	body *LogsArchiveOrder
}

func (a *LogsArchivesApi) buildUpdateLogsArchiveOrderRequest(ctx _context.Context, body LogsArchiveOrder) (apiUpdateLogsArchiveOrderRequest, error) {
	req := apiUpdateLogsArchiveOrderRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// UpdateLogsArchiveOrder Update archive order.
// Update the order of your archives. Since logs are processed sequentially, reordering an archive may change
// the structure and content of the data processed by other archives.
//
// **Note**: Using the `PUT` method updates your archive's order by replacing the current order
// with the new one.
func (a *LogsArchivesApi) UpdateLogsArchiveOrder(ctx _context.Context, body LogsArchiveOrder) (LogsArchiveOrder, *_nethttp.Response, error) {
	req, err := a.buildUpdateLogsArchiveOrderRequest(ctx, body)
	if err != nil {
		var localVarReturnValue LogsArchiveOrder
		return localVarReturnValue, nil, err
	}

	return a.updateLogsArchiveOrderExecute(req)
}

// updateLogsArchiveOrderExecute executes the request.
func (a *LogsArchivesApi) updateLogsArchiveOrderExecute(r apiUpdateLogsArchiveOrderRequest) (LogsArchiveOrder, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue LogsArchiveOrder
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsArchivesApi.UpdateLogsArchiveOrder")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/archive-order"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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

// NewLogsArchivesApi Returns NewLogsArchivesApi.
func NewLogsArchivesApi(client *datadog.APIClient) *LogsArchivesApi {
	return &LogsArchivesApi{
		Client: client,
	}
}
