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

// TagsApi service type
type TagsApi datadog.Service

type apiCreateHostTagsRequest struct {
	ctx      _context.Context
	hostName string
	body     *HostTags
	source   *string
}

// CreateHostTagsOptionalParameters holds optional parameters for CreateHostTags.
type CreateHostTagsOptionalParameters struct {
	Source *string
}

// NewCreateHostTagsOptionalParameters creates an empty struct for parameters.
func NewCreateHostTagsOptionalParameters() *CreateHostTagsOptionalParameters {
	this := CreateHostTagsOptionalParameters{}
	return &this
}

// WithSource sets the corresponding parameter name and returns the struct.
func (r *CreateHostTagsOptionalParameters) WithSource(source string) *CreateHostTagsOptionalParameters {
	r.Source = &source
	return r
}

func (a *TagsApi) buildCreateHostTagsRequest(ctx _context.Context, hostName string, body HostTags, o ...CreateHostTagsOptionalParameters) (apiCreateHostTagsRequest, error) {
	req := apiCreateHostTagsRequest{
		ctx:      ctx,
		hostName: hostName,
		body:     &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type CreateHostTagsOptionalParameters is allowed")
	}

	if o != nil {
		req.source = o[0].Source
	}
	return req, nil
}

// CreateHostTags Add tags to a host.
// This endpoint allows you to add new tags to a host,
// optionally specifying where these tags come from.
func (a *TagsApi) CreateHostTags(ctx _context.Context, hostName string, body HostTags, o ...CreateHostTagsOptionalParameters) (HostTags, *_nethttp.Response, error) {
	req, err := a.buildCreateHostTagsRequest(ctx, hostName, body, o...)
	if err != nil {
		var localVarReturnValue HostTags
		return localVarReturnValue, nil, err
	}

	return a.createHostTagsExecute(req)
}

// createHostTagsExecute executes the request.
func (a *TagsApi) createHostTagsExecute(r apiCreateHostTagsRequest) (HostTags, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue HostTags
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.TagsApi.CreateHostTags")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/tags/hosts/{host_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.hostName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.source != nil {
		localVarQueryParams.Add("source", datadog.ParameterToString(*r.source, ""))
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

type apiDeleteHostTagsRequest struct {
	ctx      _context.Context
	hostName string
	source   *string
}

// DeleteHostTagsOptionalParameters holds optional parameters for DeleteHostTags.
type DeleteHostTagsOptionalParameters struct {
	Source *string
}

// NewDeleteHostTagsOptionalParameters creates an empty struct for parameters.
func NewDeleteHostTagsOptionalParameters() *DeleteHostTagsOptionalParameters {
	this := DeleteHostTagsOptionalParameters{}
	return &this
}

// WithSource sets the corresponding parameter name and returns the struct.
func (r *DeleteHostTagsOptionalParameters) WithSource(source string) *DeleteHostTagsOptionalParameters {
	r.Source = &source
	return r
}

func (a *TagsApi) buildDeleteHostTagsRequest(ctx _context.Context, hostName string, o ...DeleteHostTagsOptionalParameters) (apiDeleteHostTagsRequest, error) {
	req := apiDeleteHostTagsRequest{
		ctx:      ctx,
		hostName: hostName,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type DeleteHostTagsOptionalParameters is allowed")
	}

	if o != nil {
		req.source = o[0].Source
	}
	return req, nil
}

// DeleteHostTags Remove host tags.
// This endpoint allows you to remove all user-assigned tags
// for a single host.
func (a *TagsApi) DeleteHostTags(ctx _context.Context, hostName string, o ...DeleteHostTagsOptionalParameters) (*_nethttp.Response, error) {
	req, err := a.buildDeleteHostTagsRequest(ctx, hostName, o...)
	if err != nil {
		return nil, err
	}

	return a.deleteHostTagsExecute(req)
}

// deleteHostTagsExecute executes the request.
func (a *TagsApi) deleteHostTagsExecute(r apiDeleteHostTagsRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.TagsApi.DeleteHostTags")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/tags/hosts/{host_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.hostName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.source != nil {
		localVarQueryParams.Add("source", datadog.ParameterToString(*r.source, ""))
	}
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

type apiGetHostTagsRequest struct {
	ctx      _context.Context
	hostName string
	source   *string
}

// GetHostTagsOptionalParameters holds optional parameters for GetHostTags.
type GetHostTagsOptionalParameters struct {
	Source *string
}

// NewGetHostTagsOptionalParameters creates an empty struct for parameters.
func NewGetHostTagsOptionalParameters() *GetHostTagsOptionalParameters {
	this := GetHostTagsOptionalParameters{}
	return &this
}

// WithSource sets the corresponding parameter name and returns the struct.
func (r *GetHostTagsOptionalParameters) WithSource(source string) *GetHostTagsOptionalParameters {
	r.Source = &source
	return r
}

func (a *TagsApi) buildGetHostTagsRequest(ctx _context.Context, hostName string, o ...GetHostTagsOptionalParameters) (apiGetHostTagsRequest, error) {
	req := apiGetHostTagsRequest{
		ctx:      ctx,
		hostName: hostName,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetHostTagsOptionalParameters is allowed")
	}

	if o != nil {
		req.source = o[0].Source
	}
	return req, nil
}

// GetHostTags Get host tags.
// Return the list of tags that apply to a given host.
func (a *TagsApi) GetHostTags(ctx _context.Context, hostName string, o ...GetHostTagsOptionalParameters) (HostTags, *_nethttp.Response, error) {
	req, err := a.buildGetHostTagsRequest(ctx, hostName, o...)
	if err != nil {
		var localVarReturnValue HostTags
		return localVarReturnValue, nil, err
	}

	return a.getHostTagsExecute(req)
}

// getHostTagsExecute executes the request.
func (a *TagsApi) getHostTagsExecute(r apiGetHostTagsRequest) (HostTags, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HostTags
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.TagsApi.GetHostTags")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/tags/hosts/{host_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.hostName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.source != nil {
		localVarQueryParams.Add("source", datadog.ParameterToString(*r.source, ""))
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

type apiListHostTagsRequest struct {
	ctx    _context.Context
	source *string
}

// ListHostTagsOptionalParameters holds optional parameters for ListHostTags.
type ListHostTagsOptionalParameters struct {
	Source *string
}

// NewListHostTagsOptionalParameters creates an empty struct for parameters.
func NewListHostTagsOptionalParameters() *ListHostTagsOptionalParameters {
	this := ListHostTagsOptionalParameters{}
	return &this
}

// WithSource sets the corresponding parameter name and returns the struct.
func (r *ListHostTagsOptionalParameters) WithSource(source string) *ListHostTagsOptionalParameters {
	r.Source = &source
	return r
}

func (a *TagsApi) buildListHostTagsRequest(ctx _context.Context, o ...ListHostTagsOptionalParameters) (apiListHostTagsRequest, error) {
	req := apiListHostTagsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListHostTagsOptionalParameters is allowed")
	}

	if o != nil {
		req.source = o[0].Source
	}
	return req, nil
}

// ListHostTags Get Tags.
// Return a mapping of tags to hosts for your whole infrastructure.
func (a *TagsApi) ListHostTags(ctx _context.Context, o ...ListHostTagsOptionalParameters) (TagToHosts, *_nethttp.Response, error) {
	req, err := a.buildListHostTagsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue TagToHosts
		return localVarReturnValue, nil, err
	}

	return a.listHostTagsExecute(req)
}

// listHostTagsExecute executes the request.
func (a *TagsApi) listHostTagsExecute(r apiListHostTagsRequest) (TagToHosts, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue TagToHosts
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.TagsApi.ListHostTags")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/tags/hosts"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.source != nil {
		localVarQueryParams.Add("source", datadog.ParameterToString(*r.source, ""))
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

type apiUpdateHostTagsRequest struct {
	ctx      _context.Context
	hostName string
	body     *HostTags
	source   *string
}

// UpdateHostTagsOptionalParameters holds optional parameters for UpdateHostTags.
type UpdateHostTagsOptionalParameters struct {
	Source *string
}

// NewUpdateHostTagsOptionalParameters creates an empty struct for parameters.
func NewUpdateHostTagsOptionalParameters() *UpdateHostTagsOptionalParameters {
	this := UpdateHostTagsOptionalParameters{}
	return &this
}

// WithSource sets the corresponding parameter name and returns the struct.
func (r *UpdateHostTagsOptionalParameters) WithSource(source string) *UpdateHostTagsOptionalParameters {
	r.Source = &source
	return r
}

func (a *TagsApi) buildUpdateHostTagsRequest(ctx _context.Context, hostName string, body HostTags, o ...UpdateHostTagsOptionalParameters) (apiUpdateHostTagsRequest, error) {
	req := apiUpdateHostTagsRequest{
		ctx:      ctx,
		hostName: hostName,
		body:     &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type UpdateHostTagsOptionalParameters is allowed")
	}

	if o != nil {
		req.source = o[0].Source
	}
	return req, nil
}

// UpdateHostTags Update host tags.
// This endpoint allows you to update/replace all tags in
// an integration source with those supplied in the request.
func (a *TagsApi) UpdateHostTags(ctx _context.Context, hostName string, body HostTags, o ...UpdateHostTagsOptionalParameters) (HostTags, *_nethttp.Response, error) {
	req, err := a.buildUpdateHostTagsRequest(ctx, hostName, body, o...)
	if err != nil {
		var localVarReturnValue HostTags
		return localVarReturnValue, nil, err
	}

	return a.updateHostTagsExecute(req)
}

// updateHostTagsExecute executes the request.
func (a *TagsApi) updateHostTagsExecute(r apiUpdateHostTagsRequest) (HostTags, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue HostTags
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.TagsApi.UpdateHostTags")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/tags/hosts/{host_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.hostName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	if r.source != nil {
		localVarQueryParams.Add("source", datadog.ParameterToString(*r.source, ""))
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

// NewTagsApi Returns NewTagsApi.
func NewTagsApi(client *datadog.APIClient) *TagsApi {
	return &TagsApi{
		Client: client,
	}
}
