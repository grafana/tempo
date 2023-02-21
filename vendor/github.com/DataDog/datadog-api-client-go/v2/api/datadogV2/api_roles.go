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

// RolesApi service type
type RolesApi datadog.Service

type apiAddPermissionToRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RelationshipToPermission
}

func (a *RolesApi) buildAddPermissionToRoleRequest(ctx _context.Context, roleId string, body RelationshipToPermission) (apiAddPermissionToRoleRequest, error) {
	req := apiAddPermissionToRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// AddPermissionToRole Grant permission to a role.
// Adds a permission to a role.
func (a *RolesApi) AddPermissionToRole(ctx _context.Context, roleId string, body RelationshipToPermission) (PermissionsResponse, *_nethttp.Response, error) {
	req, err := a.buildAddPermissionToRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue PermissionsResponse
		return localVarReturnValue, nil, err
	}

	return a.addPermissionToRoleExecute(req)
}

// addPermissionToRoleExecute executes the request.
func (a *RolesApi) addPermissionToRoleExecute(r apiAddPermissionToRoleRequest) (PermissionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue PermissionsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.AddPermissionToRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/permissions"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiAddUserToRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RelationshipToUser
}

func (a *RolesApi) buildAddUserToRoleRequest(ctx _context.Context, roleId string, body RelationshipToUser) (apiAddUserToRoleRequest, error) {
	req := apiAddUserToRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// AddUserToRole Add a user to a role.
// Adds a user to a role.
func (a *RolesApi) AddUserToRole(ctx _context.Context, roleId string, body RelationshipToUser) (UsersResponse, *_nethttp.Response, error) {
	req, err := a.buildAddUserToRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue UsersResponse
		return localVarReturnValue, nil, err
	}

	return a.addUserToRoleExecute(req)
}

// addUserToRoleExecute executes the request.
func (a *RolesApi) addUserToRoleExecute(r apiAddUserToRoleRequest) (UsersResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue UsersResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.AddUserToRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/users"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiCloneRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RoleCloneRequest
}

func (a *RolesApi) buildCloneRoleRequest(ctx _context.Context, roleId string, body RoleCloneRequest) (apiCloneRoleRequest, error) {
	req := apiCloneRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// CloneRole Create a new role by cloning an existing role.
// Clone an existing role
func (a *RolesApi) CloneRole(ctx _context.Context, roleId string, body RoleCloneRequest) (RoleResponse, *_nethttp.Response, error) {
	req, err := a.buildCloneRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue RoleResponse
		return localVarReturnValue, nil, err
	}

	return a.cloneRoleExecute(req)
}

// cloneRoleExecute executes the request.
func (a *RolesApi) cloneRoleExecute(r apiCloneRoleRequest) (RoleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue RoleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.CloneRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/clone"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiCreateRoleRequest struct {
	ctx  _context.Context
	body *RoleCreateRequest
}

func (a *RolesApi) buildCreateRoleRequest(ctx _context.Context, body RoleCreateRequest) (apiCreateRoleRequest, error) {
	req := apiCreateRoleRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateRole Create role.
// Create a new role for your organization.
func (a *RolesApi) CreateRole(ctx _context.Context, body RoleCreateRequest) (RoleCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateRoleRequest(ctx, body)
	if err != nil {
		var localVarReturnValue RoleCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createRoleExecute(req)
}

// createRoleExecute executes the request.
func (a *RolesApi) createRoleExecute(r apiCreateRoleRequest) (RoleCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue RoleCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.CreateRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles"

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

type apiDeleteRoleRequest struct {
	ctx    _context.Context
	roleId string
}

func (a *RolesApi) buildDeleteRoleRequest(ctx _context.Context, roleId string) (apiDeleteRoleRequest, error) {
	req := apiDeleteRoleRequest{
		ctx:    ctx,
		roleId: roleId,
	}
	return req, nil
}

// DeleteRole Delete role.
// Disables a role.
func (a *RolesApi) DeleteRole(ctx _context.Context, roleId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteRoleRequest(ctx, roleId)
	if err != nil {
		return nil, err
	}

	return a.deleteRoleExecute(req)
}

// deleteRoleExecute executes the request.
func (a *RolesApi) deleteRoleExecute(r apiDeleteRoleRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.DeleteRole")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiGetRoleRequest struct {
	ctx    _context.Context
	roleId string
}

func (a *RolesApi) buildGetRoleRequest(ctx _context.Context, roleId string) (apiGetRoleRequest, error) {
	req := apiGetRoleRequest{
		ctx:    ctx,
		roleId: roleId,
	}
	return req, nil
}

// GetRole Get a role.
// Get a role in the organization specified by the role’s `role_id`.
func (a *RolesApi) GetRole(ctx _context.Context, roleId string) (RoleResponse, *_nethttp.Response, error) {
	req, err := a.buildGetRoleRequest(ctx, roleId)
	if err != nil {
		var localVarReturnValue RoleResponse
		return localVarReturnValue, nil, err
	}

	return a.getRoleExecute(req)
}

// getRoleExecute executes the request.
func (a *RolesApi) getRoleExecute(r apiGetRoleRequest) (RoleResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RoleResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.GetRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiListPermissionsRequest struct {
	ctx _context.Context
}

func (a *RolesApi) buildListPermissionsRequest(ctx _context.Context) (apiListPermissionsRequest, error) {
	req := apiListPermissionsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListPermissions List permissions.
// Returns a list of all permissions, including name, description, and ID.
func (a *RolesApi) ListPermissions(ctx _context.Context) (PermissionsResponse, *_nethttp.Response, error) {
	req, err := a.buildListPermissionsRequest(ctx)
	if err != nil {
		var localVarReturnValue PermissionsResponse
		return localVarReturnValue, nil, err
	}

	return a.listPermissionsExecute(req)
}

// listPermissionsExecute executes the request.
func (a *RolesApi) listPermissionsExecute(r apiListPermissionsRequest) (PermissionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue PermissionsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.ListPermissions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/permissions"

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

type apiListRolePermissionsRequest struct {
	ctx    _context.Context
	roleId string
}

func (a *RolesApi) buildListRolePermissionsRequest(ctx _context.Context, roleId string) (apiListRolePermissionsRequest, error) {
	req := apiListRolePermissionsRequest{
		ctx:    ctx,
		roleId: roleId,
	}
	return req, nil
}

// ListRolePermissions List permissions for a role.
// Returns a list of all permissions for a single role.
func (a *RolesApi) ListRolePermissions(ctx _context.Context, roleId string) (PermissionsResponse, *_nethttp.Response, error) {
	req, err := a.buildListRolePermissionsRequest(ctx, roleId)
	if err != nil {
		var localVarReturnValue PermissionsResponse
		return localVarReturnValue, nil, err
	}

	return a.listRolePermissionsExecute(req)
}

// listRolePermissionsExecute executes the request.
func (a *RolesApi) listRolePermissionsExecute(r apiListRolePermissionsRequest) (PermissionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue PermissionsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.ListRolePermissions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/permissions"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiListRoleUsersRequest struct {
	ctx        _context.Context
	roleId     string
	pageSize   *int64
	pageNumber *int64
	sort       *string
	filter     *string
}

// ListRoleUsersOptionalParameters holds optional parameters for ListRoleUsers.
type ListRoleUsersOptionalParameters struct {
	PageSize   *int64
	PageNumber *int64
	Sort       *string
	Filter     *string
}

// NewListRoleUsersOptionalParameters creates an empty struct for parameters.
func NewListRoleUsersOptionalParameters() *ListRoleUsersOptionalParameters {
	this := ListRoleUsersOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListRoleUsersOptionalParameters) WithPageSize(pageSize int64) *ListRoleUsersOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListRoleUsersOptionalParameters) WithPageNumber(pageNumber int64) *ListRoleUsersOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListRoleUsersOptionalParameters) WithSort(sort string) *ListRoleUsersOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListRoleUsersOptionalParameters) WithFilter(filter string) *ListRoleUsersOptionalParameters {
	r.Filter = &filter
	return r
}

func (a *RolesApi) buildListRoleUsersRequest(ctx _context.Context, roleId string, o ...ListRoleUsersOptionalParameters) (apiListRoleUsersRequest, error) {
	req := apiListRoleUsersRequest{
		ctx:    ctx,
		roleId: roleId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListRoleUsersOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
	}
	return req, nil
}

// ListRoleUsers Get all users of a role.
// Gets all users of a role.
func (a *RolesApi) ListRoleUsers(ctx _context.Context, roleId string, o ...ListRoleUsersOptionalParameters) (UsersResponse, *_nethttp.Response, error) {
	req, err := a.buildListRoleUsersRequest(ctx, roleId, o...)
	if err != nil {
		var localVarReturnValue UsersResponse
		return localVarReturnValue, nil, err
	}

	return a.listRoleUsersExecute(req)
}

// listRoleUsersExecute executes the request.
func (a *RolesApi) listRoleUsersExecute(r apiListRoleUsersRequest) (UsersResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsersResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.ListRoleUsers")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/users"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiListRolesRequest struct {
	ctx        _context.Context
	pageSize   *int64
	pageNumber *int64
	sort       *RolesSort
	filter     *string
}

// ListRolesOptionalParameters holds optional parameters for ListRoles.
type ListRolesOptionalParameters struct {
	PageSize   *int64
	PageNumber *int64
	Sort       *RolesSort
	Filter     *string
}

// NewListRolesOptionalParameters creates an empty struct for parameters.
func NewListRolesOptionalParameters() *ListRolesOptionalParameters {
	this := ListRolesOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListRolesOptionalParameters) WithPageSize(pageSize int64) *ListRolesOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListRolesOptionalParameters) WithPageNumber(pageNumber int64) *ListRolesOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListRolesOptionalParameters) WithSort(sort RolesSort) *ListRolesOptionalParameters {
	r.Sort = &sort
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListRolesOptionalParameters) WithFilter(filter string) *ListRolesOptionalParameters {
	r.Filter = &filter
	return r
}

func (a *RolesApi) buildListRolesRequest(ctx _context.Context, o ...ListRolesOptionalParameters) (apiListRolesRequest, error) {
	req := apiListRolesRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListRolesOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.filter = o[0].Filter
	}
	return req, nil
}

// ListRoles List roles.
// Returns all roles, including their names and their unique identifiers.
func (a *RolesApi) ListRoles(ctx _context.Context, o ...ListRolesOptionalParameters) (RolesResponse, *_nethttp.Response, error) {
	req, err := a.buildListRolesRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue RolesResponse
		return localVarReturnValue, nil, err
	}

	return a.listRolesExecute(req)
}

// listRolesExecute executes the request.
func (a *RolesApi) listRolesExecute(r apiListRolesRequest) (RolesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue RolesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.ListRoles")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles"

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

type apiRemovePermissionFromRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RelationshipToPermission
}

func (a *RolesApi) buildRemovePermissionFromRoleRequest(ctx _context.Context, roleId string, body RelationshipToPermission) (apiRemovePermissionFromRoleRequest, error) {
	req := apiRemovePermissionFromRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// RemovePermissionFromRole Revoke permission.
// Removes a permission from a role.
func (a *RolesApi) RemovePermissionFromRole(ctx _context.Context, roleId string, body RelationshipToPermission) (PermissionsResponse, *_nethttp.Response, error) {
	req, err := a.buildRemovePermissionFromRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue PermissionsResponse
		return localVarReturnValue, nil, err
	}

	return a.removePermissionFromRoleExecute(req)
}

// removePermissionFromRoleExecute executes the request.
func (a *RolesApi) removePermissionFromRoleExecute(r apiRemovePermissionFromRoleRequest) (PermissionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue PermissionsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.RemovePermissionFromRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/permissions"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiRemoveUserFromRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RelationshipToUser
}

func (a *RolesApi) buildRemoveUserFromRoleRequest(ctx _context.Context, roleId string, body RelationshipToUser) (apiRemoveUserFromRoleRequest, error) {
	req := apiRemoveUserFromRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// RemoveUserFromRole Remove a user from a role.
// Removes a user from a role.
func (a *RolesApi) RemoveUserFromRole(ctx _context.Context, roleId string, body RelationshipToUser) (UsersResponse, *_nethttp.Response, error) {
	req, err := a.buildRemoveUserFromRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue UsersResponse
		return localVarReturnValue, nil, err
	}

	return a.removeUserFromRoleExecute(req)
}

// removeUserFromRoleExecute executes the request.
func (a *RolesApi) removeUserFromRoleExecute(r apiRemoveUserFromRoleRequest) (UsersResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue UsersResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.RemoveUserFromRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}/users"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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

type apiUpdateRoleRequest struct {
	ctx    _context.Context
	roleId string
	body   *RoleUpdateRequest
}

func (a *RolesApi) buildUpdateRoleRequest(ctx _context.Context, roleId string, body RoleUpdateRequest) (apiUpdateRoleRequest, error) {
	req := apiUpdateRoleRequest{
		ctx:    ctx,
		roleId: roleId,
		body:   &body,
	}
	return req, nil
}

// UpdateRole Update a role.
// Edit a role. Can only be used with application keys belonging to administrators.
func (a *RolesApi) UpdateRole(ctx _context.Context, roleId string, body RoleUpdateRequest) (RoleUpdateResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateRoleRequest(ctx, roleId, body)
	if err != nil {
		var localVarReturnValue RoleUpdateResponse
		return localVarReturnValue, nil, err
	}

	return a.updateRoleExecute(req)
}

// updateRoleExecute executes the request.
func (a *RolesApi) updateRoleExecute(r apiUpdateRoleRequest) (RoleUpdateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue RoleUpdateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.RolesApi.UpdateRole")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/roles/{role_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"role_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.roleId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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

// NewRolesApi Returns NewRolesApi.
func NewRolesApi(client *datadog.APIClient) *RolesApi {
	return &RolesApi{
		Client: client,
	}
}
