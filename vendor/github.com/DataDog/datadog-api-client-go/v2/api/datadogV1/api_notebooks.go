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

// NotebooksApi service type
type NotebooksApi datadog.Service

type apiCreateNotebookRequest struct {
	ctx  _context.Context
	body *NotebookCreateRequest
}

func (a *NotebooksApi) buildCreateNotebookRequest(ctx _context.Context, body NotebookCreateRequest) (apiCreateNotebookRequest, error) {
	req := apiCreateNotebookRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateNotebook Create a notebook.
// Create a notebook using the specified options.
func (a *NotebooksApi) CreateNotebook(ctx _context.Context, body NotebookCreateRequest) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateNotebookRequest(ctx, body)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return a.createNotebookExecute(req)
}

// createNotebookExecute executes the request.
func (a *NotebooksApi) createNotebookExecute(r apiCreateNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.NotebooksApi.CreateNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

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

type apiDeleteNotebookRequest struct {
	ctx        _context.Context
	notebookId int64
}

func (a *NotebooksApi) buildDeleteNotebookRequest(ctx _context.Context, notebookId int64) (apiDeleteNotebookRequest, error) {
	req := apiDeleteNotebookRequest{
		ctx:        ctx,
		notebookId: notebookId,
	}
	return req, nil
}

// DeleteNotebook Delete a notebook.
// Delete a notebook using the specified ID.
func (a *NotebooksApi) DeleteNotebook(ctx _context.Context, notebookId int64) (*_nethttp.Response, error) {
	req, err := a.buildDeleteNotebookRequest(ctx, notebookId)
	if err != nil {
		return nil, err
	}

	return a.deleteNotebookExecute(req)
}

// deleteNotebookExecute executes the request.
func (a *NotebooksApi) deleteNotebookExecute(r apiDeleteNotebookRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.NotebooksApi.DeleteNotebook")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.notebookId, "")), -1)

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

type apiGetNotebookRequest struct {
	ctx        _context.Context
	notebookId int64
}

func (a *NotebooksApi) buildGetNotebookRequest(ctx _context.Context, notebookId int64) (apiGetNotebookRequest, error) {
	req := apiGetNotebookRequest{
		ctx:        ctx,
		notebookId: notebookId,
	}
	return req, nil
}

// GetNotebook Get a notebook.
// Get a notebook using the specified notebook ID.
func (a *NotebooksApi) GetNotebook(ctx _context.Context, notebookId int64) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildGetNotebookRequest(ctx, notebookId)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return a.getNotebookExecute(req)
}

// getNotebookExecute executes the request.
func (a *NotebooksApi) getNotebookExecute(r apiGetNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.NotebooksApi.GetNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.notebookId, "")), -1)

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

type apiListNotebooksRequest struct {
	ctx                 _context.Context
	authorHandle        *string
	excludeAuthorHandle *string
	start               *int64
	count               *int64
	sortField           *string
	sortDir             *string
	query               *string
	includeCells        *bool
	isTemplate          *bool
	typeVar             *string
}

// ListNotebooksOptionalParameters holds optional parameters for ListNotebooks.
type ListNotebooksOptionalParameters struct {
	AuthorHandle        *string
	ExcludeAuthorHandle *string
	Start               *int64
	Count               *int64
	SortField           *string
	SortDir             *string
	Query               *string
	IncludeCells        *bool
	IsTemplate          *bool
	Type                *string
}

// NewListNotebooksOptionalParameters creates an empty struct for parameters.
func NewListNotebooksOptionalParameters() *ListNotebooksOptionalParameters {
	this := ListNotebooksOptionalParameters{}
	return &this
}

// WithAuthorHandle sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithAuthorHandle(authorHandle string) *ListNotebooksOptionalParameters {
	r.AuthorHandle = &authorHandle
	return r
}

// WithExcludeAuthorHandle sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithExcludeAuthorHandle(excludeAuthorHandle string) *ListNotebooksOptionalParameters {
	r.ExcludeAuthorHandle = &excludeAuthorHandle
	return r
}

// WithStart sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithStart(start int64) *ListNotebooksOptionalParameters {
	r.Start = &start
	return r
}

// WithCount sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithCount(count int64) *ListNotebooksOptionalParameters {
	r.Count = &count
	return r
}

// WithSortField sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithSortField(sortField string) *ListNotebooksOptionalParameters {
	r.SortField = &sortField
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithSortDir(sortDir string) *ListNotebooksOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithQuery sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithQuery(query string) *ListNotebooksOptionalParameters {
	r.Query = &query
	return r
}

// WithIncludeCells sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithIncludeCells(includeCells bool) *ListNotebooksOptionalParameters {
	r.IncludeCells = &includeCells
	return r
}

// WithIsTemplate sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithIsTemplate(isTemplate bool) *ListNotebooksOptionalParameters {
	r.IsTemplate = &isTemplate
	return r
}

// WithType sets the corresponding parameter name and returns the struct.
func (r *ListNotebooksOptionalParameters) WithType(typeVar string) *ListNotebooksOptionalParameters {
	r.Type = &typeVar
	return r
}

func (a *NotebooksApi) buildListNotebooksRequest(ctx _context.Context, o ...ListNotebooksOptionalParameters) (apiListNotebooksRequest, error) {
	req := apiListNotebooksRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListNotebooksOptionalParameters is allowed")
	}

	if o != nil {
		req.authorHandle = o[0].AuthorHandle
		req.excludeAuthorHandle = o[0].ExcludeAuthorHandle
		req.start = o[0].Start
		req.count = o[0].Count
		req.sortField = o[0].SortField
		req.sortDir = o[0].SortDir
		req.query = o[0].Query
		req.includeCells = o[0].IncludeCells
		req.isTemplate = o[0].IsTemplate
		req.typeVar = o[0].Type
	}
	return req, nil
}

// ListNotebooks Get all notebooks.
// Get all notebooks. This can also be used to search for notebooks with a particular `query` in the notebook
// `name` or author `handle`.
func (a *NotebooksApi) ListNotebooks(ctx _context.Context, o ...ListNotebooksOptionalParameters) (NotebooksResponse, *_nethttp.Response, error) {
	req, err := a.buildListNotebooksRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue NotebooksResponse
		return localVarReturnValue, nil, err
	}

	return a.listNotebooksExecute(req)
}

// listNotebooksExecute executes the request.
func (a *NotebooksApi) listNotebooksExecute(r apiListNotebooksRequest) (NotebooksResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebooksResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.NotebooksApi.ListNotebooks")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.authorHandle != nil {
		localVarQueryParams.Add("author_handle", datadog.ParameterToString(*r.authorHandle, ""))
	}
	if r.excludeAuthorHandle != nil {
		localVarQueryParams.Add("exclude_author_handle", datadog.ParameterToString(*r.excludeAuthorHandle, ""))
	}
	if r.start != nil {
		localVarQueryParams.Add("start", datadog.ParameterToString(*r.start, ""))
	}
	if r.count != nil {
		localVarQueryParams.Add("count", datadog.ParameterToString(*r.count, ""))
	}
	if r.sortField != nil {
		localVarQueryParams.Add("sort_field", datadog.ParameterToString(*r.sortField, ""))
	}
	if r.sortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*r.sortDir, ""))
	}
	if r.query != nil {
		localVarQueryParams.Add("query", datadog.ParameterToString(*r.query, ""))
	}
	if r.includeCells != nil {
		localVarQueryParams.Add("include_cells", datadog.ParameterToString(*r.includeCells, ""))
	}
	if r.isTemplate != nil {
		localVarQueryParams.Add("is_template", datadog.ParameterToString(*r.isTemplate, ""))
	}
	if r.typeVar != nil {
		localVarQueryParams.Add("type", datadog.ParameterToString(*r.typeVar, ""))
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

type apiUpdateNotebookRequest struct {
	ctx        _context.Context
	notebookId int64
	body       *NotebookUpdateRequest
}

func (a *NotebooksApi) buildUpdateNotebookRequest(ctx _context.Context, notebookId int64, body NotebookUpdateRequest) (apiUpdateNotebookRequest, error) {
	req := apiUpdateNotebookRequest{
		ctx:        ctx,
		notebookId: notebookId,
		body:       &body,
	}
	return req, nil
}

// UpdateNotebook Update a notebook.
// Update a notebook using the specified ID.
func (a *NotebooksApi) UpdateNotebook(ctx _context.Context, notebookId int64, body NotebookUpdateRequest) (NotebookResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateNotebookRequest(ctx, notebookId, body)
	if err != nil {
		var localVarReturnValue NotebookResponse
		return localVarReturnValue, nil, err
	}

	return a.updateNotebookExecute(req)
}

// updateNotebookExecute executes the request.
func (a *NotebooksApi) updateNotebookExecute(r apiUpdateNotebookRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.NotebooksApi.UpdateNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.notebookId, "")), -1)

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

// NewNotebooksApi Returns NewNotebooksApi.
func NewNotebooksApi(client *datadog.APIClient) *NotebooksApi {
	return &NotebooksApi{
		Client: client,
	}
}
