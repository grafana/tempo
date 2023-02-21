// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_io "io"
	_nethttp "net/http"
	_neturl "net/url"
	"os"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// OrganizationsApi service type
type OrganizationsApi datadog.Service

type apiCreateChildOrgRequest struct {
	ctx  _context.Context
	body *OrganizationCreateBody
}

func (a *OrganizationsApi) buildCreateChildOrgRequest(ctx _context.Context, body OrganizationCreateBody) (apiCreateChildOrgRequest, error) {
	req := apiCreateChildOrgRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateChildOrg Create a child organization.
// Create a child organization.
//
// This endpoint requires the
// [multi-organization account](https://docs.datadoghq.com/account_management/multi_organization/)
// feature and must be enabled by
// [contacting support](https://docs.datadoghq.com/help/).
//
// Once a new child organization is created, you can interact with it
// by using the `org.public_id`, `api_key.key`, and
// `application_key.hash` provided in the response.
func (a *OrganizationsApi) CreateChildOrg(ctx _context.Context, body OrganizationCreateBody) (OrganizationCreateResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateChildOrgRequest(ctx, body)
	if err != nil {
		var localVarReturnValue OrganizationCreateResponse
		return localVarReturnValue, nil, err
	}

	return a.createChildOrgExecute(req)
}

// createChildOrgExecute executes the request.
func (a *OrganizationsApi) createChildOrgExecute(r apiCreateChildOrgRequest) (OrganizationCreateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue OrganizationCreateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.CreateChildOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org"

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

type apiDowngradeOrgRequest struct {
	ctx      _context.Context
	publicId string
}

func (a *OrganizationsApi) buildDowngradeOrgRequest(ctx _context.Context, publicId string) (apiDowngradeOrgRequest, error) {
	req := apiDowngradeOrgRequest{
		ctx:      ctx,
		publicId: publicId,
	}
	return req, nil
}

// DowngradeOrg Spin-off Child Organization.
// Only available for MSP customers. Removes a child organization from the hierarchy of the master organization and places the child organization on a 30-day trial.
func (a *OrganizationsApi) DowngradeOrg(ctx _context.Context, publicId string) (OrgDowngradedResponse, *_nethttp.Response, error) {
	req, err := a.buildDowngradeOrgRequest(ctx, publicId)
	if err != nil {
		var localVarReturnValue OrgDowngradedResponse
		return localVarReturnValue, nil, err
	}

	return a.downgradeOrgExecute(req)
}

// downgradeOrgExecute executes the request.
func (a *OrganizationsApi) downgradeOrgExecute(r apiDowngradeOrgRequest) (OrgDowngradedResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue OrgDowngradedResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.DowngradeOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org/{public_id}/downgrade"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

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

type apiGetOrgRequest struct {
	ctx      _context.Context
	publicId string
}

func (a *OrganizationsApi) buildGetOrgRequest(ctx _context.Context, publicId string) (apiGetOrgRequest, error) {
	req := apiGetOrgRequest{
		ctx:      ctx,
		publicId: publicId,
	}
	return req, nil
}

// GetOrg Get organization information.
// Get organization information.
func (a *OrganizationsApi) GetOrg(ctx _context.Context, publicId string) (OrganizationResponse, *_nethttp.Response, error) {
	req, err := a.buildGetOrgRequest(ctx, publicId)
	if err != nil {
		var localVarReturnValue OrganizationResponse
		return localVarReturnValue, nil, err
	}

	return a.getOrgExecute(req)
}

// getOrgExecute executes the request.
func (a *OrganizationsApi) getOrgExecute(r apiGetOrgRequest) (OrganizationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue OrganizationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.GetOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org/{public_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

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

type apiListOrgsRequest struct {
	ctx _context.Context
}

func (a *OrganizationsApi) buildListOrgsRequest(ctx _context.Context) (apiListOrgsRequest, error) {
	req := apiListOrgsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListOrgs List your managed organizations.
// This endpoint returns data on your top-level organization.
func (a *OrganizationsApi) ListOrgs(ctx _context.Context) (OrganizationListResponse, *_nethttp.Response, error) {
	req, err := a.buildListOrgsRequest(ctx)
	if err != nil {
		var localVarReturnValue OrganizationListResponse
		return localVarReturnValue, nil, err
	}

	return a.listOrgsExecute(req)
}

// listOrgsExecute executes the request.
func (a *OrganizationsApi) listOrgsExecute(r apiListOrgsRequest) (OrganizationListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue OrganizationListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.ListOrgs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org"

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

type apiUpdateOrgRequest struct {
	ctx      _context.Context
	publicId string
	body     *Organization
}

func (a *OrganizationsApi) buildUpdateOrgRequest(ctx _context.Context, publicId string, body Organization) (apiUpdateOrgRequest, error) {
	req := apiUpdateOrgRequest{
		ctx:      ctx,
		publicId: publicId,
		body:     &body,
	}
	return req, nil
}

// UpdateOrg Update your organization.
// Update your organization.
func (a *OrganizationsApi) UpdateOrg(ctx _context.Context, publicId string, body Organization) (OrganizationResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateOrgRequest(ctx, publicId, body)
	if err != nil {
		var localVarReturnValue OrganizationResponse
		return localVarReturnValue, nil, err
	}

	return a.updateOrgExecute(req)
}

// updateOrgExecute executes the request.
func (a *OrganizationsApi) updateOrgExecute(r apiUpdateOrgRequest) (OrganizationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue OrganizationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.UpdateOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org/{public_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

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

type apiUploadIdPForOrgRequest struct {
	ctx      _context.Context
	publicId string
	idpFile  **os.File
}

func (a *OrganizationsApi) buildUploadIdPForOrgRequest(ctx _context.Context, publicId string, idpFile *os.File) (apiUploadIdPForOrgRequest, error) {
	req := apiUploadIdPForOrgRequest{
		ctx:      ctx,
		publicId: publicId,
		idpFile:  &idpFile,
	}
	return req, nil
}

// UploadIdPForOrg Upload IdP metadata.
// There are a couple of options for updating the Identity Provider (IdP)
// metadata from your SAML IdP.
//
// * **Multipart Form-Data**: Post the IdP metadata file using a form post.
//
// * **XML Body:** Post the IdP metadata file as the body of the request.
func (a *OrganizationsApi) UploadIdPForOrg(ctx _context.Context, publicId string, idpFile *os.File) (IdpResponse, *_nethttp.Response, error) {
	req, err := a.buildUploadIdPForOrgRequest(ctx, publicId, idpFile)
	if err != nil {
		var localVarReturnValue IdpResponse
		return localVarReturnValue, nil, err
	}

	return a.uploadIdPForOrgExecute(req)
}

// uploadIdPForOrgExecute executes the request.
func (a *OrganizationsApi) uploadIdPForOrgExecute(r apiUploadIdPForOrgRequest) (IdpResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IdpResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.OrganizationsApi.UploadIdPForOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/org/{public_id}/idp_metadata"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.idpFile == nil {
		return localVarReturnValue, nil, datadog.ReportError("idpFile is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "multipart/form-data"
	localVarHeaderParams["Accept"] = "application/json"

	formFile := datadog.FormFile{}
	formFile.FormFileName = "idp_file"
	localVarFile := *r.idpFile
	if localVarFile != nil {
		fbs, _ := _io.ReadAll(localVarFile)
		formFile.FileBytes = fbs
		formFile.FileName = localVarFile.Name()
		localVarFile.Close()
	}
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, &formFile)
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 415 || localVarHTTPResponse.StatusCode == 429 {
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

// NewOrganizationsApi Returns NewOrganizationsApi.
func NewOrganizationsApi(client *datadog.APIClient) *OrganizationsApi {
	return &OrganizationsApi{
		Client: client,
	}
}
