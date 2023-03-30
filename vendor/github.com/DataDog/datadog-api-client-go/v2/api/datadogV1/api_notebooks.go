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

// CreateNotebook Create a notebook.
// Create a notebook using the specified options.
func (a *NotebooksApi) CreateNotebook(ctx _context.Context, body NotebookCreateRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.NotebooksApi.CreateNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	// body params
	localVarPostBody = &body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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

// DeleteNotebook Delete a notebook.
// Delete a notebook using the specified ID.
func (a *NotebooksApi) DeleteNotebook(ctx _context.Context, notebookId int64) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.NotebooksApi.DeleteNotebook")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Accept"] = "*/*"

	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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

// GetNotebook Get a notebook.
// Get a notebook using the specified notebook ID.
func (a *NotebooksApi) GetNotebook(ctx _context.Context, notebookId int64) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.NotebooksApi.GetNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Accept"] = "application/json"

	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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

// ListNotebooks Get all notebooks.
// Get all notebooks. This can also be used to search for notebooks with a particular `query` in the notebook
// `name` or author `handle`.
func (a *NotebooksApi) ListNotebooks(ctx _context.Context, o ...ListNotebooksOptionalParameters) (NotebooksResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue NotebooksResponse
		optionalParams      ListNotebooksOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListNotebooksOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.NotebooksApi.ListNotebooks")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.AuthorHandle != nil {
		localVarQueryParams.Add("author_handle", datadog.ParameterToString(*optionalParams.AuthorHandle, ""))
	}
	if optionalParams.ExcludeAuthorHandle != nil {
		localVarQueryParams.Add("exclude_author_handle", datadog.ParameterToString(*optionalParams.ExcludeAuthorHandle, ""))
	}
	if optionalParams.Start != nil {
		localVarQueryParams.Add("start", datadog.ParameterToString(*optionalParams.Start, ""))
	}
	if optionalParams.Count != nil {
		localVarQueryParams.Add("count", datadog.ParameterToString(*optionalParams.Count, ""))
	}
	if optionalParams.SortField != nil {
		localVarQueryParams.Add("sort_field", datadog.ParameterToString(*optionalParams.SortField, ""))
	}
	if optionalParams.SortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*optionalParams.SortDir, ""))
	}
	if optionalParams.Query != nil {
		localVarQueryParams.Add("query", datadog.ParameterToString(*optionalParams.Query, ""))
	}
	if optionalParams.IncludeCells != nil {
		localVarQueryParams.Add("include_cells", datadog.ParameterToString(*optionalParams.IncludeCells, ""))
	}
	if optionalParams.IsTemplate != nil {
		localVarQueryParams.Add("is_template", datadog.ParameterToString(*optionalParams.IsTemplate, ""))
	}
	if optionalParams.Type != nil {
		localVarQueryParams.Add("type", datadog.ParameterToString(*optionalParams.Type, ""))
	}
	localVarHeaderParams["Accept"] = "application/json"

	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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

// UpdateNotebook Update a notebook.
// Update a notebook using the specified ID.
func (a *NotebooksApi) UpdateNotebook(ctx _context.Context, notebookId int64, body NotebookUpdateRequest) (NotebookResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue NotebookResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.NotebooksApi.UpdateNotebook")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/notebooks/{notebook_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"notebook_id"+"}", _neturl.PathEscape(datadog.ParameterToString(notebookId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	// body params
	localVarPostBody = &body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
		[2]string{"appKeyAuth", "DD-APPLICATION-KEY"},
	)
	req, err := a.Client.PrepareRequest(ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, nil)
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
