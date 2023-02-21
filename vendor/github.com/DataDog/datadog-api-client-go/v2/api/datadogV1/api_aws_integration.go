// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// AWSIntegrationApi service type
type AWSIntegrationApi datadog.Service

type apiCreateAWSAccountRequest struct {
	ctx  _context.Context
	body *AWSAccount
}

func (a *AWSIntegrationApi) buildCreateAWSAccountRequest(ctx _context.Context, body AWSAccount) (apiCreateAWSAccountRequest, error) {
	req := apiCreateAWSAccountRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateAWSAccount Create an AWS integration.
// Create a Datadog-Amazon Web Services integration.
// Using the `POST` method updates your integration configuration
// by adding your new configuration to the existing one in your Datadog organization.
// A unique AWS Account ID for role based authentication.
func (a *AWSIntegrationApi) CreateAWSAccount(ctx _context.Context, body AWSAccount) (AWSAccountCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateAWSAccountRequest(ctx, body)
	if err != nil {
		var localVarReturnValue AWSAccountCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createAWSAccountExecute(req)
}

// createAWSAccountExecute executes the request.
func (a *AWSIntegrationApi) createAWSAccountExecute(r apiCreateAWSAccountRequest) (AWSAccountCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue AWSAccountCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.CreateAWSAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws"

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

type apiCreateAWSTagFilterRequest struct {
	ctx  _context.Context
	body *AWSTagFilterCreateRequest
}

func (a *AWSIntegrationApi) buildCreateAWSTagFilterRequest(ctx _context.Context, body AWSTagFilterCreateRequest) (apiCreateAWSTagFilterRequest, error) {
	req := apiCreateAWSTagFilterRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateAWSTagFilter Set an AWS tag filter.
// Set an AWS tag filter.
func (a *AWSIntegrationApi) CreateAWSTagFilter(ctx _context.Context, body AWSTagFilterCreateRequest) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildCreateAWSTagFilterRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.createAWSTagFilterExecute(req)
}

// createAWSTagFilterExecute executes the request.
func (a *AWSIntegrationApi) createAWSTagFilterExecute(r apiCreateAWSTagFilterRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.CreateAWSTagFilter")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws/filtering"

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

type apiCreateNewAWSExternalIDRequest struct {
	ctx  _context.Context
	body *AWSAccount
}

func (a *AWSIntegrationApi) buildCreateNewAWSExternalIDRequest(ctx _context.Context, body AWSAccount) (apiCreateNewAWSExternalIDRequest, error) {
	req := apiCreateNewAWSExternalIDRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateNewAWSExternalID Generate a new external ID.
// Generate a new AWS external ID for a given AWS account ID and role name pair.
func (a *AWSIntegrationApi) CreateNewAWSExternalID(ctx _context.Context, body AWSAccount) (AWSAccountCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateNewAWSExternalIDRequest(ctx, body)
	if err != nil {
		var localVarReturnValue AWSAccountCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createNewAWSExternalIDExecute(req)
}

// createNewAWSExternalIDExecute executes the request.
func (a *AWSIntegrationApi) createNewAWSExternalIDExecute(r apiCreateNewAWSExternalIDRequest) (AWSAccountCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue AWSAccountCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.CreateNewAWSExternalID")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws/generate_new_external_id"

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

type apiDeleteAWSAccountRequest struct {
	ctx  _context.Context
	body *AWSAccountDeleteRequest
}

func (a *AWSIntegrationApi) buildDeleteAWSAccountRequest(ctx _context.Context, body AWSAccountDeleteRequest) (apiDeleteAWSAccountRequest, error) {
	req := apiDeleteAWSAccountRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteAWSAccount Delete an AWS integration.
// Delete a Datadog-AWS integration matching the specified `account_id` and `role_name parameters`.
func (a *AWSIntegrationApi) DeleteAWSAccount(ctx _context.Context, body AWSAccountDeleteRequest) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildDeleteAWSAccountRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.deleteAWSAccountExecute(req)
}

// deleteAWSAccountExecute executes the request.
func (a *AWSIntegrationApi) deleteAWSAccountExecute(r apiDeleteAWSAccountRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.DeleteAWSAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws"

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

type apiDeleteAWSTagFilterRequest struct {
	ctx  _context.Context
	body *AWSTagFilterDeleteRequest
}

func (a *AWSIntegrationApi) buildDeleteAWSTagFilterRequest(ctx _context.Context, body AWSTagFilterDeleteRequest) (apiDeleteAWSTagFilterRequest, error) {
	req := apiDeleteAWSTagFilterRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteAWSTagFilter Delete a tag filtering entry.
// Delete a tag filtering entry.
func (a *AWSIntegrationApi) DeleteAWSTagFilter(ctx _context.Context, body AWSTagFilterDeleteRequest) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildDeleteAWSTagFilterRequest(ctx, body)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.deleteAWSTagFilterExecute(req)
}

// deleteAWSTagFilterExecute executes the request.
func (a *AWSIntegrationApi) deleteAWSTagFilterExecute(r apiDeleteAWSTagFilterRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.DeleteAWSTagFilter")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws/filtering"

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

type apiListAWSAccountsRequest struct {
	ctx         _context.Context
	accountId   *string
	roleName    *string
	accessKeyId *string
}

// ListAWSAccountsOptionalParameters holds optional parameters for ListAWSAccounts.
type ListAWSAccountsOptionalParameters struct {
	AccountId   *string
	RoleName    *string
	AccessKeyId *string
}

// NewListAWSAccountsOptionalParameters creates an empty struct for parameters.
func NewListAWSAccountsOptionalParameters() *ListAWSAccountsOptionalParameters {
	this := ListAWSAccountsOptionalParameters{}
	return &this
}

// WithAccountId sets the corresponding parameter name and returns the struct.
func (r *ListAWSAccountsOptionalParameters) WithAccountId(accountId string) *ListAWSAccountsOptionalParameters {
	r.AccountId = &accountId
	return r
}

// WithRoleName sets the corresponding parameter name and returns the struct.
func (r *ListAWSAccountsOptionalParameters) WithRoleName(roleName string) *ListAWSAccountsOptionalParameters {
	r.RoleName = &roleName
	return r
}

// WithAccessKeyId sets the corresponding parameter name and returns the struct.
func (r *ListAWSAccountsOptionalParameters) WithAccessKeyId(accessKeyId string) *ListAWSAccountsOptionalParameters {
	r.AccessKeyId = &accessKeyId
	return r
}

func (a *AWSIntegrationApi) buildListAWSAccountsRequest(ctx _context.Context, o ...ListAWSAccountsOptionalParameters) (apiListAWSAccountsRequest, error) {
	req := apiListAWSAccountsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListAWSAccountsOptionalParameters is allowed")
	}

	if o != nil {
		req.accountId = o[0].AccountId
		req.roleName = o[0].RoleName
		req.accessKeyId = o[0].AccessKeyId
	}
	return req, nil
}

// ListAWSAccounts List all AWS integrations.
// List all Datadog-AWS integrations available in your Datadog organization.
func (a *AWSIntegrationApi) ListAWSAccounts(ctx _context.Context, o ...ListAWSAccountsOptionalParameters) (AWSAccountListResponse, *_nethttp.Response, error) {
	req, err := a.buildListAWSAccountsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue AWSAccountListResponse
		return localVarReturnValue, nil, err
	}

	return a.listAWSAccountsExecute(req)
}

// listAWSAccountsExecute executes the request.
func (a *AWSIntegrationApi) listAWSAccountsExecute(r apiListAWSAccountsRequest) (AWSAccountListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue AWSAccountListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.ListAWSAccounts")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.accountId != nil {
		localVarQueryParams.Add("account_id", datadog.ParameterToString(*r.accountId, ""))
	}
	if r.roleName != nil {
		localVarQueryParams.Add("role_name", datadog.ParameterToString(*r.roleName, ""))
	}
	if r.accessKeyId != nil {
		localVarQueryParams.Add("access_key_id", datadog.ParameterToString(*r.accessKeyId, ""))
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

type apiListAWSTagFiltersRequest struct {
	ctx       _context.Context
	accountId *string
}

func (a *AWSIntegrationApi) buildListAWSTagFiltersRequest(ctx _context.Context, accountId string) (apiListAWSTagFiltersRequest, error) {
	req := apiListAWSTagFiltersRequest{
		ctx:       ctx,
		accountId: &accountId,
	}
	return req, nil
}

// ListAWSTagFilters Get all AWS tag filters.
// Get all AWS tag filters.
func (a *AWSIntegrationApi) ListAWSTagFilters(ctx _context.Context, accountId string) (AWSTagFilterListResponse, *_nethttp.Response, error) {
	req, err := a.buildListAWSTagFiltersRequest(ctx, accountId)
	if err != nil {
		var localVarReturnValue AWSTagFilterListResponse
		return localVarReturnValue, nil, err
	}

	return a.listAWSTagFiltersExecute(req)
}

// listAWSTagFiltersExecute executes the request.
func (a *AWSIntegrationApi) listAWSTagFiltersExecute(r apiListAWSTagFiltersRequest) (AWSTagFilterListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue AWSTagFilterListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.ListAWSTagFilters")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws/filtering"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.accountId == nil {
		return localVarReturnValue, nil, datadog.ReportError("accountId is required and must be specified")
	}
	localVarQueryParams.Add("account_id", datadog.ParameterToString(*r.accountId, ""))
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

type apiListAvailableAWSNamespacesRequest struct {
	ctx _context.Context
}

func (a *AWSIntegrationApi) buildListAvailableAWSNamespacesRequest(ctx _context.Context) (apiListAvailableAWSNamespacesRequest, error) {
	req := apiListAvailableAWSNamespacesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListAvailableAWSNamespaces List namespace rules.
// List all namespace rules for a given Datadog-AWS integration. This endpoint takes no arguments.
func (a *AWSIntegrationApi) ListAvailableAWSNamespaces(ctx _context.Context) ([]string, *_nethttp.Response, error) {
	req, err := a.buildListAvailableAWSNamespacesRequest(ctx)
	if err != nil {
		var localVarReturnValue []string
		return localVarReturnValue, nil, err
	}

	return a.listAvailableAWSNamespacesExecute(req)
}

// listAvailableAWSNamespacesExecute executes the request.
func (a *AWSIntegrationApi) listAvailableAWSNamespacesExecute(r apiListAvailableAWSNamespacesRequest) ([]string, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue []string
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.ListAvailableAWSNamespaces")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws/available_namespace_rules"

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

type apiUpdateAWSAccountRequest struct {
	ctx         _context.Context
	body        *AWSAccount
	accountId   *string
	roleName    *string
	accessKeyId *string
}

// UpdateAWSAccountOptionalParameters holds optional parameters for UpdateAWSAccount.
type UpdateAWSAccountOptionalParameters struct {
	AccountId   *string
	RoleName    *string
	AccessKeyId *string
}

// NewUpdateAWSAccountOptionalParameters creates an empty struct for parameters.
func NewUpdateAWSAccountOptionalParameters() *UpdateAWSAccountOptionalParameters {
	this := UpdateAWSAccountOptionalParameters{}
	return &this
}

// WithAccountId sets the corresponding parameter name and returns the struct.
func (r *UpdateAWSAccountOptionalParameters) WithAccountId(accountId string) *UpdateAWSAccountOptionalParameters {
	r.AccountId = &accountId
	return r
}

// WithRoleName sets the corresponding parameter name and returns the struct.
func (r *UpdateAWSAccountOptionalParameters) WithRoleName(roleName string) *UpdateAWSAccountOptionalParameters {
	r.RoleName = &roleName
	return r
}

// WithAccessKeyId sets the corresponding parameter name and returns the struct.
func (r *UpdateAWSAccountOptionalParameters) WithAccessKeyId(accessKeyId string) *UpdateAWSAccountOptionalParameters {
	r.AccessKeyId = &accessKeyId
	return r
}

func (a *AWSIntegrationApi) buildUpdateAWSAccountRequest(ctx _context.Context, body AWSAccount, o ...UpdateAWSAccountOptionalParameters) (apiUpdateAWSAccountRequest, error) {
	req := apiUpdateAWSAccountRequest{
		ctx:  ctx,
		body: &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type UpdateAWSAccountOptionalParameters is allowed")
	}

	if o != nil {
		req.accountId = o[0].AccountId
		req.roleName = o[0].RoleName
		req.accessKeyId = o[0].AccessKeyId
	}
	return req, nil
}

// UpdateAWSAccount Update an AWS integration.
// Update a Datadog-Amazon Web Services integration.
func (a *AWSIntegrationApi) UpdateAWSAccount(ctx _context.Context, body AWSAccount, o ...UpdateAWSAccountOptionalParameters) (interface{}, *_nethttp.Response, error) {
	req, err := a.buildUpdateAWSAccountRequest(ctx, body, o...)
	if err != nil {
		var localVarReturnValue interface{}
		return localVarReturnValue, nil, err
	}

	return a.updateAWSAccountExecute(req)
}

// updateAWSAccountExecute executes the request.
func (a *AWSIntegrationApi) updateAWSAccountExecute(r apiUpdateAWSAccountRequest) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.AWSIntegrationApi.UpdateAWSAccount")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/integration/aws"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.accountId != nil {
		localVarQueryParams.Add("account_id", datadog.ParameterToString(*r.accountId, ""))
	}
	if r.roleName != nil {
		localVarQueryParams.Add("role_name", datadog.ParameterToString(*r.roleName, ""))
	}
	if r.accessKeyId != nil {
		localVarQueryParams.Add("access_key_id", datadog.ParameterToString(*r.accessKeyId, ""))
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

// NewAWSIntegrationApi Returns NewAWSIntegrationApi.
func NewAWSIntegrationApi(client *datadog.APIClient) *AWSIntegrationApi {
	return &AWSIntegrationApi{
		Client: client,
	}
}
