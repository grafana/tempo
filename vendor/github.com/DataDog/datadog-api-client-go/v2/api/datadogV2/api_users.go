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

// UsersApi service type
type UsersApi datadog.Service

type apiCreateUserRequest struct {
	ctx  _context.Context
	body *UserCreateRequest
}

func (a *UsersApi) buildCreateUserRequest(ctx _context.Context, body UserCreateRequest) (apiCreateUserRequest, error) {
	req := apiCreateUserRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateUser Create a user.
// Create a user for your organization.
func (a *UsersApi) CreateUser(ctx _context.Context, body UserCreateRequest) (UserResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateUserRequest(ctx, body)
	if err != nil {
		var localVarReturnValue UserResponse
		return localVarReturnValue, nil, err
	}

	return a.createUserExecute(req)
}

// createUserExecute executes the request.
func (a *UsersApi) createUserExecute(r apiCreateUserRequest) (UserResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue UserResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.CreateUser")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users"

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

type apiDisableUserRequest struct {
	ctx    _context.Context
	userId string
}

func (a *UsersApi) buildDisableUserRequest(ctx _context.Context, userId string) (apiDisableUserRequest, error) {
	req := apiDisableUserRequest{
		ctx:    ctx,
		userId: userId,
	}
	return req, nil
}

// DisableUser Disable a user.
// Disable a user. Can only be used with an application key belonging
// to an administrator user.
func (a *UsersApi) DisableUser(ctx _context.Context, userId string) (*_nethttp.Response, error) {
	req, err := a.buildDisableUserRequest(ctx, userId)
	if err != nil {
		return nil, err
	}

	return a.disableUserExecute(req)
}

// disableUserExecute executes the request.
func (a *UsersApi) disableUserExecute(r apiDisableUserRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.DisableUser")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users/{user_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"user_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userId, "")), -1)

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

type apiGetInvitationRequest struct {
	ctx                _context.Context
	userInvitationUuid string
}

func (a *UsersApi) buildGetInvitationRequest(ctx _context.Context, userInvitationUuid string) (apiGetInvitationRequest, error) {
	req := apiGetInvitationRequest{
		ctx:                ctx,
		userInvitationUuid: userInvitationUuid,
	}
	return req, nil
}

// GetInvitation Get a user invitation.
// Returns a single user invitation by its UUID.
func (a *UsersApi) GetInvitation(ctx _context.Context, userInvitationUuid string) (UserInvitationResponse, *_nethttp.Response, error) {
	req, err := a.buildGetInvitationRequest(ctx, userInvitationUuid)
	if err != nil {
		var localVarReturnValue UserInvitationResponse
		return localVarReturnValue, nil, err
	}

	return a.getInvitationExecute(req)
}

// getInvitationExecute executes the request.
func (a *UsersApi) getInvitationExecute(r apiGetInvitationRequest) (UserInvitationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UserInvitationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.GetInvitation")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/user_invitations/{user_invitation_uuid}"
	localVarPath = strings.Replace(localVarPath, "{"+"user_invitation_uuid"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userInvitationUuid, "")), -1)

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

type apiGetUserRequest struct {
	ctx    _context.Context
	userId string
}

func (a *UsersApi) buildGetUserRequest(ctx _context.Context, userId string) (apiGetUserRequest, error) {
	req := apiGetUserRequest{
		ctx:    ctx,
		userId: userId,
	}
	return req, nil
}

// GetUser Get user details.
// Get a user in the organization specified by the user’s `user_id`.
func (a *UsersApi) GetUser(ctx _context.Context, userId string) (UserResponse, *_nethttp.Response, error) {
	req, err := a.buildGetUserRequest(ctx, userId)
	if err != nil {
		var localVarReturnValue UserResponse
		return localVarReturnValue, nil, err
	}

	return a.getUserExecute(req)
}

// getUserExecute executes the request.
func (a *UsersApi) getUserExecute(r apiGetUserRequest) (UserResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UserResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.GetUser")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users/{user_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"user_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userId, "")), -1)

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

type apiListUserOrganizationsRequest struct {
	ctx    _context.Context
	userId string
}

func (a *UsersApi) buildListUserOrganizationsRequest(ctx _context.Context, userId string) (apiListUserOrganizationsRequest, error) {
	req := apiListUserOrganizationsRequest{
		ctx:    ctx,
		userId: userId,
	}
	return req, nil
}

// ListUserOrganizations Get a user organization.
// Get a user organization. Returns the user information and all organizations
// joined by this user.
func (a *UsersApi) ListUserOrganizations(ctx _context.Context, userId string) (UserResponse, *_nethttp.Response, error) {
	req, err := a.buildListUserOrganizationsRequest(ctx, userId)
	if err != nil {
		var localVarReturnValue UserResponse
		return localVarReturnValue, nil, err
	}

	return a.listUserOrganizationsExecute(req)
}

// listUserOrganizationsExecute executes the request.
func (a *UsersApi) listUserOrganizationsExecute(r apiListUserOrganizationsRequest) (UserResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UserResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.ListUserOrganizations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users/{user_id}/orgs"
	localVarPath = strings.Replace(localVarPath, "{"+"user_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userId, "")), -1)

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

type apiListUserPermissionsRequest struct {
	ctx    _context.Context
	userId string
}

func (a *UsersApi) buildListUserPermissionsRequest(ctx _context.Context, userId string) (apiListUserPermissionsRequest, error) {
	req := apiListUserPermissionsRequest{
		ctx:    ctx,
		userId: userId,
	}
	return req, nil
}

// ListUserPermissions Get a user permissions.
// Get a user permission set. Returns a list of the user’s permissions
// granted by the associated user's roles.
func (a *UsersApi) ListUserPermissions(ctx _context.Context, userId string) (PermissionsResponse, *_nethttp.Response, error) {
	req, err := a.buildListUserPermissionsRequest(ctx, userId)
	if err != nil {
		var localVarReturnValue PermissionsResponse
		return localVarReturnValue, nil, err
	}

	return a.listUserPermissionsExecute(req)
}

// listUserPermissionsExecute executes the request.
func (a *UsersApi) listUserPermissionsExecute(r apiListUserPermissionsRequest) (PermissionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue PermissionsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.ListUserPermissions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users/{user_id}/permissions"
	localVarPath = strings.Replace(localVarPath, "{"+"user_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userId, "")), -1)

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

type apiListUsersRequest struct {
	ctx          _context.Context
	pageSize     *int64
	pageNumber   *int64
	sort         *string
	sortDir      *QuerySortOrder
	filter       *string
	filterStatus *string
}

// ListUsersOptionalParameters holds optional parameters for ListUsers.
type ListUsersOptionalParameters struct {
	PageSize     *int64
	PageNumber   *int64
	Sort         *string
	SortDir      *QuerySortOrder
	Filter       *string
	FilterStatus *string
}

// NewListUsersOptionalParameters creates an empty struct for parameters.
func NewListUsersOptionalParameters() *ListUsersOptionalParameters {
	this := ListUsersOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithPageSize(pageSize int64) *ListUsersOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithPageNumber(pageNumber int64) *ListUsersOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithSort(sort string) *ListUsersOptionalParameters {
	r.Sort = &sort
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithSortDir(sortDir QuerySortOrder) *ListUsersOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithFilter(filter string) *ListUsersOptionalParameters {
	r.Filter = &filter
	return r
}

// WithFilterStatus sets the corresponding parameter name and returns the struct.
func (r *ListUsersOptionalParameters) WithFilterStatus(filterStatus string) *ListUsersOptionalParameters {
	r.FilterStatus = &filterStatus
	return r
}

func (a *UsersApi) buildListUsersRequest(ctx _context.Context, o ...ListUsersOptionalParameters) (apiListUsersRequest, error) {
	req := apiListUsersRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListUsersOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
		req.sort = o[0].Sort
		req.sortDir = o[0].SortDir
		req.filter = o[0].Filter
		req.filterStatus = o[0].FilterStatus
	}
	return req, nil
}

// ListUsers List all users.
// Get the list of all users in the organization. This list includes
// all users even if they are deactivated or unverified.
func (a *UsersApi) ListUsers(ctx _context.Context, o ...ListUsersOptionalParameters) (UsersResponse, *_nethttp.Response, error) {
	req, err := a.buildListUsersRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue UsersResponse
		return localVarReturnValue, nil, err
	}

	return a.listUsersExecute(req)
}

// listUsersExecute executes the request.
func (a *UsersApi) listUsersExecute(r apiListUsersRequest) (UsersResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsersResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.ListUsers")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users"

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
	if r.sortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*r.sortDir, ""))
	}
	if r.filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*r.filter, ""))
	}
	if r.filterStatus != nil {
		localVarQueryParams.Add("filter[status]", datadog.ParameterToString(*r.filterStatus, ""))
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

type apiSendInvitationsRequest struct {
	ctx  _context.Context
	body *UserInvitationsRequest
}

func (a *UsersApi) buildSendInvitationsRequest(ctx _context.Context, body UserInvitationsRequest) (apiSendInvitationsRequest, error) {
	req := apiSendInvitationsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// SendInvitations Send invitation emails.
// Sends emails to one or more users inviting them to join the organization.
func (a *UsersApi) SendInvitations(ctx _context.Context, body UserInvitationsRequest) (UserInvitationsResponse, *_nethttp.Response, error) {
	req, err := a.buildSendInvitationsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue UserInvitationsResponse
		return localVarReturnValue, nil, err
	}

	return a.sendInvitationsExecute(req)
}

// sendInvitationsExecute executes the request.
func (a *UsersApi) sendInvitationsExecute(r apiSendInvitationsRequest) (UserInvitationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue UserInvitationsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.SendInvitations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/user_invitations"

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

type apiUpdateUserRequest struct {
	ctx    _context.Context
	userId string
	body   *UserUpdateRequest
}

func (a *UsersApi) buildUpdateUserRequest(ctx _context.Context, userId string, body UserUpdateRequest) (apiUpdateUserRequest, error) {
	req := apiUpdateUserRequest{
		ctx:    ctx,
		userId: userId,
		body:   &body,
	}
	return req, nil
}

// UpdateUser Update a user.
// Edit a user. Can only be used with an application key belonging
// to an administrator user.
func (a *UsersApi) UpdateUser(ctx _context.Context, userId string, body UserUpdateRequest) (UserResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateUserRequest(ctx, userId, body)
	if err != nil {
		var localVarReturnValue UserResponse
		return localVarReturnValue, nil, err
	}

	return a.updateUserExecute(req)
}

// updateUserExecute executes the request.
func (a *UsersApi) updateUserExecute(r apiUpdateUserRequest) (UserResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue UserResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.UsersApi.UpdateUser")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/users/{user_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"user_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.userId, "")), -1)

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

// NewUsersApi Returns NewUsersApi.
func NewUsersApi(client *datadog.APIClient) *UsersApi {
	return &UsersApi{
		Client: client,
	}
}
