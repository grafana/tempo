// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"reflect"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// SyntheticsApi service type
type SyntheticsApi datadog.Service

type apiCreateGlobalVariableRequest struct {
	ctx  _context.Context
	body *SyntheticsGlobalVariable
}

func (a *SyntheticsApi) buildCreateGlobalVariableRequest(ctx _context.Context, body SyntheticsGlobalVariable) (apiCreateGlobalVariableRequest, error) {
	req := apiCreateGlobalVariableRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateGlobalVariable Create a global variable.
// Create a Synthetics global variable.
func (a *SyntheticsApi) CreateGlobalVariable(ctx _context.Context, body SyntheticsGlobalVariable) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	req, err := a.buildCreateGlobalVariableRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsGlobalVariable
		return localVarReturnValue, nil, err
	}

	return a.createGlobalVariableExecute(req)
}

// createGlobalVariableExecute executes the request.
func (a *SyntheticsApi) createGlobalVariableExecute(r apiCreateGlobalVariableRequest) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsGlobalVariable
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.CreateGlobalVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/variables"

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

type apiCreatePrivateLocationRequest struct {
	ctx  _context.Context
	body *SyntheticsPrivateLocation
}

func (a *SyntheticsApi) buildCreatePrivateLocationRequest(ctx _context.Context, body SyntheticsPrivateLocation) (apiCreatePrivateLocationRequest, error) {
	req := apiCreatePrivateLocationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreatePrivateLocation Create a private location.
// Create a new Synthetics private location.
func (a *SyntheticsApi) CreatePrivateLocation(ctx _context.Context, body SyntheticsPrivateLocation) (SyntheticsPrivateLocationCreationResponse, *_nethttp.Response, error) {
	req, err := a.buildCreatePrivateLocationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsPrivateLocationCreationResponse
		return localVarReturnValue, nil, err
	}

	return a.createPrivateLocationExecute(req)
}

// createPrivateLocationExecute executes the request.
func (a *SyntheticsApi) createPrivateLocationExecute(r apiCreatePrivateLocationRequest) (SyntheticsPrivateLocationCreationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsPrivateLocationCreationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.CreatePrivateLocation")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/private-locations"

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
		if localVarHTTPResponse.StatusCode == 402 || localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiCreateSyntheticsAPITestRequest struct {
	ctx  _context.Context
	body *SyntheticsAPITest
}

func (a *SyntheticsApi) buildCreateSyntheticsAPITestRequest(ctx _context.Context, body SyntheticsAPITest) (apiCreateSyntheticsAPITestRequest, error) {
	req := apiCreateSyntheticsAPITestRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateSyntheticsAPITest Create an API test.
// Create a Synthetic API test.
func (a *SyntheticsApi) CreateSyntheticsAPITest(ctx _context.Context, body SyntheticsAPITest) (SyntheticsAPITest, *_nethttp.Response, error) {
	req, err := a.buildCreateSyntheticsAPITestRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsAPITest
		return localVarReturnValue, nil, err
	}

	return a.createSyntheticsAPITestExecute(req)
}

// createSyntheticsAPITestExecute executes the request.
func (a *SyntheticsApi) createSyntheticsAPITestExecute(r apiCreateSyntheticsAPITestRequest) (SyntheticsAPITest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsAPITest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.CreateSyntheticsAPITest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/api"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 402 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
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

type apiCreateSyntheticsBrowserTestRequest struct {
	ctx  _context.Context
	body *SyntheticsBrowserTest
}

func (a *SyntheticsApi) buildCreateSyntheticsBrowserTestRequest(ctx _context.Context, body SyntheticsBrowserTest) (apiCreateSyntheticsBrowserTestRequest, error) {
	req := apiCreateSyntheticsBrowserTestRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateSyntheticsBrowserTest Create a browser test.
// Create a Synthetic browser test.
func (a *SyntheticsApi) CreateSyntheticsBrowserTest(ctx _context.Context, body SyntheticsBrowserTest) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	req, err := a.buildCreateSyntheticsBrowserTestRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsBrowserTest
		return localVarReturnValue, nil, err
	}

	return a.createSyntheticsBrowserTestExecute(req)
}

// createSyntheticsBrowserTestExecute executes the request.
func (a *SyntheticsApi) createSyntheticsBrowserTestExecute(r apiCreateSyntheticsBrowserTestRequest) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsBrowserTest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.CreateSyntheticsBrowserTest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/browser"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 402 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
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

type apiDeleteGlobalVariableRequest struct {
	ctx        _context.Context
	variableId string
}

func (a *SyntheticsApi) buildDeleteGlobalVariableRequest(ctx _context.Context, variableId string) (apiDeleteGlobalVariableRequest, error) {
	req := apiDeleteGlobalVariableRequest{
		ctx:        ctx,
		variableId: variableId,
	}
	return req, nil
}

// DeleteGlobalVariable Delete a global variable.
// Delete a Synthetics global variable.
func (a *SyntheticsApi) DeleteGlobalVariable(ctx _context.Context, variableId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteGlobalVariableRequest(ctx, variableId)
	if err != nil {
		return nil, err
	}

	return a.deleteGlobalVariableExecute(req)
}

// deleteGlobalVariableExecute executes the request.
func (a *SyntheticsApi) deleteGlobalVariableExecute(r apiDeleteGlobalVariableRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.DeleteGlobalVariable")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/variables/{variable_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"variable_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.variableId, "")), -1)

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

type apiDeletePrivateLocationRequest struct {
	ctx        _context.Context
	locationId string
}

func (a *SyntheticsApi) buildDeletePrivateLocationRequest(ctx _context.Context, locationId string) (apiDeletePrivateLocationRequest, error) {
	req := apiDeletePrivateLocationRequest{
		ctx:        ctx,
		locationId: locationId,
	}
	return req, nil
}

// DeletePrivateLocation Delete a private location.
// Delete a Synthetics private location.
func (a *SyntheticsApi) DeletePrivateLocation(ctx _context.Context, locationId string) (*_nethttp.Response, error) {
	req, err := a.buildDeletePrivateLocationRequest(ctx, locationId)
	if err != nil {
		return nil, err
	}

	return a.deletePrivateLocationExecute(req)
}

// deletePrivateLocationExecute executes the request.
func (a *SyntheticsApi) deletePrivateLocationExecute(r apiDeletePrivateLocationRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.DeletePrivateLocation")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/private-locations/{location_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"location_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.locationId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiDeleteTestsRequest struct {
	ctx  _context.Context
	body *SyntheticsDeleteTestsPayload
}

func (a *SyntheticsApi) buildDeleteTestsRequest(ctx _context.Context, body SyntheticsDeleteTestsPayload) (apiDeleteTestsRequest, error) {
	req := apiDeleteTestsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteTests Delete tests.
// Delete multiple Synthetic tests by ID.
func (a *SyntheticsApi) DeleteTests(ctx _context.Context, body SyntheticsDeleteTestsPayload) (SyntheticsDeleteTestsResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteTestsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsDeleteTestsResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteTestsExecute(req)
}

// deleteTestsExecute executes the request.
func (a *SyntheticsApi) deleteTestsExecute(r apiDeleteTestsRequest) (SyntheticsDeleteTestsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsDeleteTestsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.DeleteTests")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/delete"

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

type apiEditGlobalVariableRequest struct {
	ctx        _context.Context
	variableId string
	body       *SyntheticsGlobalVariable
}

func (a *SyntheticsApi) buildEditGlobalVariableRequest(ctx _context.Context, variableId string, body SyntheticsGlobalVariable) (apiEditGlobalVariableRequest, error) {
	req := apiEditGlobalVariableRequest{
		ctx:        ctx,
		variableId: variableId,
		body:       &body,
	}
	return req, nil
}

// EditGlobalVariable Edit a global variable.
// Edit a Synthetics global variable.
func (a *SyntheticsApi) EditGlobalVariable(ctx _context.Context, variableId string, body SyntheticsGlobalVariable) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	req, err := a.buildEditGlobalVariableRequest(ctx, variableId, body)
	if err != nil {
		var localVarReturnValue SyntheticsGlobalVariable
		return localVarReturnValue, nil, err
	}

	return a.editGlobalVariableExecute(req)
}

// editGlobalVariableExecute executes the request.
func (a *SyntheticsApi) editGlobalVariableExecute(r apiEditGlobalVariableRequest) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsGlobalVariable
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.EditGlobalVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/variables/{variable_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"variable_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.variableId, "")), -1)

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

type apiGetAPITestRequest struct {
	ctx      _context.Context
	publicId string
}

func (a *SyntheticsApi) buildGetAPITestRequest(ctx _context.Context, publicId string) (apiGetAPITestRequest, error) {
	req := apiGetAPITestRequest{
		ctx:      ctx,
		publicId: publicId,
	}
	return req, nil
}

// GetAPITest Get an API test.
// Get the detailed configuration associated with
// a Synthetic API test.
func (a *SyntheticsApi) GetAPITest(ctx _context.Context, publicId string) (SyntheticsAPITest, *_nethttp.Response, error) {
	req, err := a.buildGetAPITestRequest(ctx, publicId)
	if err != nil {
		var localVarReturnValue SyntheticsAPITest
		return localVarReturnValue, nil, err
	}

	return a.getAPITestExecute(req)
}

// getAPITestExecute executes the request.
func (a *SyntheticsApi) getAPITestExecute(r apiGetAPITestRequest) (SyntheticsAPITest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsAPITest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetAPITest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/api/{public_id}"
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

type apiGetAPITestLatestResultsRequest struct {
	ctx      _context.Context
	publicId string
	fromTs   *int64
	toTs     *int64
	probeDc  *[]string
}

// GetAPITestLatestResultsOptionalParameters holds optional parameters for GetAPITestLatestResults.
type GetAPITestLatestResultsOptionalParameters struct {
	FromTs  *int64
	ToTs    *int64
	ProbeDc *[]string
}

// NewGetAPITestLatestResultsOptionalParameters creates an empty struct for parameters.
func NewGetAPITestLatestResultsOptionalParameters() *GetAPITestLatestResultsOptionalParameters {
	this := GetAPITestLatestResultsOptionalParameters{}
	return &this
}

// WithFromTs sets the corresponding parameter name and returns the struct.
func (r *GetAPITestLatestResultsOptionalParameters) WithFromTs(fromTs int64) *GetAPITestLatestResultsOptionalParameters {
	r.FromTs = &fromTs
	return r
}

// WithToTs sets the corresponding parameter name and returns the struct.
func (r *GetAPITestLatestResultsOptionalParameters) WithToTs(toTs int64) *GetAPITestLatestResultsOptionalParameters {
	r.ToTs = &toTs
	return r
}

// WithProbeDc sets the corresponding parameter name and returns the struct.
func (r *GetAPITestLatestResultsOptionalParameters) WithProbeDc(probeDc []string) *GetAPITestLatestResultsOptionalParameters {
	r.ProbeDc = &probeDc
	return r
}

func (a *SyntheticsApi) buildGetAPITestLatestResultsRequest(ctx _context.Context, publicId string, o ...GetAPITestLatestResultsOptionalParameters) (apiGetAPITestLatestResultsRequest, error) {
	req := apiGetAPITestLatestResultsRequest{
		ctx:      ctx,
		publicId: publicId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetAPITestLatestResultsOptionalParameters is allowed")
	}

	if o != nil {
		req.fromTs = o[0].FromTs
		req.toTs = o[0].ToTs
		req.probeDc = o[0].ProbeDc
	}
	return req, nil
}

// GetAPITestLatestResults Get an API test's latest results summaries.
// Get the last 50 test results summaries for a given Synthetics API test.
func (a *SyntheticsApi) GetAPITestLatestResults(ctx _context.Context, publicId string, o ...GetAPITestLatestResultsOptionalParameters) (SyntheticsGetAPITestLatestResultsResponse, *_nethttp.Response, error) {
	req, err := a.buildGetAPITestLatestResultsRequest(ctx, publicId, o...)
	if err != nil {
		var localVarReturnValue SyntheticsGetAPITestLatestResultsResponse
		return localVarReturnValue, nil, err
	}

	return a.getAPITestLatestResultsExecute(req)
}

// getAPITestLatestResultsExecute executes the request.
func (a *SyntheticsApi) getAPITestLatestResultsExecute(r apiGetAPITestLatestResultsRequest) (SyntheticsGetAPITestLatestResultsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsGetAPITestLatestResultsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetAPITestLatestResults")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/{public_id}/results"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.fromTs != nil {
		localVarQueryParams.Add("from_ts", datadog.ParameterToString(*r.fromTs, ""))
	}
	if r.toTs != nil {
		localVarQueryParams.Add("to_ts", datadog.ParameterToString(*r.toTs, ""))
	}
	if r.probeDc != nil {
		t := *r.probeDc
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				localVarQueryParams.Add("probe_dc", datadog.ParameterToString(s.Index(i), "multi"))
			}
		} else {
			localVarQueryParams.Add("probe_dc", datadog.ParameterToString(t, "multi"))
		}
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

type apiGetAPITestResultRequest struct {
	ctx      _context.Context
	publicId string
	resultId string
}

func (a *SyntheticsApi) buildGetAPITestResultRequest(ctx _context.Context, publicId string, resultId string) (apiGetAPITestResultRequest, error) {
	req := apiGetAPITestResultRequest{
		ctx:      ctx,
		publicId: publicId,
		resultId: resultId,
	}
	return req, nil
}

// GetAPITestResult Get an API test result.
// Get a specific full result from a given (API) Synthetic test.
func (a *SyntheticsApi) GetAPITestResult(ctx _context.Context, publicId string, resultId string) (SyntheticsAPITestResultFull, *_nethttp.Response, error) {
	req, err := a.buildGetAPITestResultRequest(ctx, publicId, resultId)
	if err != nil {
		var localVarReturnValue SyntheticsAPITestResultFull
		return localVarReturnValue, nil, err
	}

	return a.getAPITestResultExecute(req)
}

// getAPITestResultExecute executes the request.
func (a *SyntheticsApi) getAPITestResultExecute(r apiGetAPITestResultRequest) (SyntheticsAPITestResultFull, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsAPITestResultFull
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetAPITestResult")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/{public_id}/results/{result_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"result_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.resultId, "")), -1)

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

type apiGetBrowserTestRequest struct {
	ctx      _context.Context
	publicId string
}

func (a *SyntheticsApi) buildGetBrowserTestRequest(ctx _context.Context, publicId string) (apiGetBrowserTestRequest, error) {
	req := apiGetBrowserTestRequest{
		ctx:      ctx,
		publicId: publicId,
	}
	return req, nil
}

// GetBrowserTest Get a browser test.
// Get the detailed configuration (including steps) associated with
// a Synthetic browser test.
func (a *SyntheticsApi) GetBrowserTest(ctx _context.Context, publicId string) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	req, err := a.buildGetBrowserTestRequest(ctx, publicId)
	if err != nil {
		var localVarReturnValue SyntheticsBrowserTest
		return localVarReturnValue, nil, err
	}

	return a.getBrowserTestExecute(req)
}

// getBrowserTestExecute executes the request.
func (a *SyntheticsApi) getBrowserTestExecute(r apiGetBrowserTestRequest) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsBrowserTest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetBrowserTest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/browser/{public_id}"
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

type apiGetBrowserTestLatestResultsRequest struct {
	ctx      _context.Context
	publicId string
	fromTs   *int64
	toTs     *int64
	probeDc  *[]string
}

// GetBrowserTestLatestResultsOptionalParameters holds optional parameters for GetBrowserTestLatestResults.
type GetBrowserTestLatestResultsOptionalParameters struct {
	FromTs  *int64
	ToTs    *int64
	ProbeDc *[]string
}

// NewGetBrowserTestLatestResultsOptionalParameters creates an empty struct for parameters.
func NewGetBrowserTestLatestResultsOptionalParameters() *GetBrowserTestLatestResultsOptionalParameters {
	this := GetBrowserTestLatestResultsOptionalParameters{}
	return &this
}

// WithFromTs sets the corresponding parameter name and returns the struct.
func (r *GetBrowserTestLatestResultsOptionalParameters) WithFromTs(fromTs int64) *GetBrowserTestLatestResultsOptionalParameters {
	r.FromTs = &fromTs
	return r
}

// WithToTs sets the corresponding parameter name and returns the struct.
func (r *GetBrowserTestLatestResultsOptionalParameters) WithToTs(toTs int64) *GetBrowserTestLatestResultsOptionalParameters {
	r.ToTs = &toTs
	return r
}

// WithProbeDc sets the corresponding parameter name and returns the struct.
func (r *GetBrowserTestLatestResultsOptionalParameters) WithProbeDc(probeDc []string) *GetBrowserTestLatestResultsOptionalParameters {
	r.ProbeDc = &probeDc
	return r
}

func (a *SyntheticsApi) buildGetBrowserTestLatestResultsRequest(ctx _context.Context, publicId string, o ...GetBrowserTestLatestResultsOptionalParameters) (apiGetBrowserTestLatestResultsRequest, error) {
	req := apiGetBrowserTestLatestResultsRequest{
		ctx:      ctx,
		publicId: publicId,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetBrowserTestLatestResultsOptionalParameters is allowed")
	}

	if o != nil {
		req.fromTs = o[0].FromTs
		req.toTs = o[0].ToTs
		req.probeDc = o[0].ProbeDc
	}
	return req, nil
}

// GetBrowserTestLatestResults Get a browser test's latest results summaries.
// Get the last 50 test results summaries for a given Synthetics Browser test.
func (a *SyntheticsApi) GetBrowserTestLatestResults(ctx _context.Context, publicId string, o ...GetBrowserTestLatestResultsOptionalParameters) (SyntheticsGetBrowserTestLatestResultsResponse, *_nethttp.Response, error) {
	req, err := a.buildGetBrowserTestLatestResultsRequest(ctx, publicId, o...)
	if err != nil {
		var localVarReturnValue SyntheticsGetBrowserTestLatestResultsResponse
		return localVarReturnValue, nil, err
	}

	return a.getBrowserTestLatestResultsExecute(req)
}

// getBrowserTestLatestResultsExecute executes the request.
func (a *SyntheticsApi) getBrowserTestLatestResultsExecute(r apiGetBrowserTestLatestResultsRequest) (SyntheticsGetBrowserTestLatestResultsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsGetBrowserTestLatestResultsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetBrowserTestLatestResults")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/browser/{public_id}/results"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.fromTs != nil {
		localVarQueryParams.Add("from_ts", datadog.ParameterToString(*r.fromTs, ""))
	}
	if r.toTs != nil {
		localVarQueryParams.Add("to_ts", datadog.ParameterToString(*r.toTs, ""))
	}
	if r.probeDc != nil {
		t := *r.probeDc
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				localVarQueryParams.Add("probe_dc", datadog.ParameterToString(s.Index(i), "multi"))
			}
		} else {
			localVarQueryParams.Add("probe_dc", datadog.ParameterToString(t, "multi"))
		}
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

type apiGetBrowserTestResultRequest struct {
	ctx      _context.Context
	publicId string
	resultId string
}

func (a *SyntheticsApi) buildGetBrowserTestResultRequest(ctx _context.Context, publicId string, resultId string) (apiGetBrowserTestResultRequest, error) {
	req := apiGetBrowserTestResultRequest{
		ctx:      ctx,
		publicId: publicId,
		resultId: resultId,
	}
	return req, nil
}

// GetBrowserTestResult Get a browser test result.
// Get a specific full result from a given (browser) Synthetic test.
func (a *SyntheticsApi) GetBrowserTestResult(ctx _context.Context, publicId string, resultId string) (SyntheticsBrowserTestResultFull, *_nethttp.Response, error) {
	req, err := a.buildGetBrowserTestResultRequest(ctx, publicId, resultId)
	if err != nil {
		var localVarReturnValue SyntheticsBrowserTestResultFull
		return localVarReturnValue, nil, err
	}

	return a.getBrowserTestResultExecute(req)
}

// getBrowserTestResultExecute executes the request.
func (a *SyntheticsApi) getBrowserTestResultExecute(r apiGetBrowserTestResultRequest) (SyntheticsBrowserTestResultFull, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsBrowserTestResultFull
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetBrowserTestResult")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/browser/{public_id}/results/{result_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"public_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.publicId, "")), -1)
	localVarPath = strings.Replace(localVarPath, "{"+"result_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.resultId, "")), -1)

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

type apiGetGlobalVariableRequest struct {
	ctx        _context.Context
	variableId string
}

func (a *SyntheticsApi) buildGetGlobalVariableRequest(ctx _context.Context, variableId string) (apiGetGlobalVariableRequest, error) {
	req := apiGetGlobalVariableRequest{
		ctx:        ctx,
		variableId: variableId,
	}
	return req, nil
}

// GetGlobalVariable Get a global variable.
// Get the detailed configuration of a global variable.
func (a *SyntheticsApi) GetGlobalVariable(ctx _context.Context, variableId string) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	req, err := a.buildGetGlobalVariableRequest(ctx, variableId)
	if err != nil {
		var localVarReturnValue SyntheticsGlobalVariable
		return localVarReturnValue, nil, err
	}

	return a.getGlobalVariableExecute(req)
}

// getGlobalVariableExecute executes the request.
func (a *SyntheticsApi) getGlobalVariableExecute(r apiGetGlobalVariableRequest) (SyntheticsGlobalVariable, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsGlobalVariable
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetGlobalVariable")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/variables/{variable_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"variable_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.variableId, "")), -1)

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

type apiGetPrivateLocationRequest struct {
	ctx        _context.Context
	locationId string
}

func (a *SyntheticsApi) buildGetPrivateLocationRequest(ctx _context.Context, locationId string) (apiGetPrivateLocationRequest, error) {
	req := apiGetPrivateLocationRequest{
		ctx:        ctx,
		locationId: locationId,
	}
	return req, nil
}

// GetPrivateLocation Get a private location.
// Get a Synthetics private location.
func (a *SyntheticsApi) GetPrivateLocation(ctx _context.Context, locationId string) (SyntheticsPrivateLocation, *_nethttp.Response, error) {
	req, err := a.buildGetPrivateLocationRequest(ctx, locationId)
	if err != nil {
		var localVarReturnValue SyntheticsPrivateLocation
		return localVarReturnValue, nil, err
	}

	return a.getPrivateLocationExecute(req)
}

// getPrivateLocationExecute executes the request.
func (a *SyntheticsApi) getPrivateLocationExecute(r apiGetPrivateLocationRequest) (SyntheticsPrivateLocation, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsPrivateLocation
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetPrivateLocation")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/private-locations/{location_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"location_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.locationId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiGetSyntheticsCIBatchRequest struct {
	ctx     _context.Context
	batchId string
}

func (a *SyntheticsApi) buildGetSyntheticsCIBatchRequest(ctx _context.Context, batchId string) (apiGetSyntheticsCIBatchRequest, error) {
	req := apiGetSyntheticsCIBatchRequest{
		ctx:     ctx,
		batchId: batchId,
	}
	return req, nil
}

// GetSyntheticsCIBatch Get details of batch.
// Get a batch's updated details.
func (a *SyntheticsApi) GetSyntheticsCIBatch(ctx _context.Context, batchId string) (SyntheticsBatchDetails, *_nethttp.Response, error) {
	req, err := a.buildGetSyntheticsCIBatchRequest(ctx, batchId)
	if err != nil {
		var localVarReturnValue SyntheticsBatchDetails
		return localVarReturnValue, nil, err
	}

	return a.getSyntheticsCIBatchExecute(req)
}

// getSyntheticsCIBatchExecute executes the request.
func (a *SyntheticsApi) getSyntheticsCIBatchExecute(r apiGetSyntheticsCIBatchRequest) (SyntheticsBatchDetails, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsBatchDetails
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetSyntheticsCIBatch")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/ci/batch/{batch_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"batch_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.batchId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiGetTestRequest struct {
	ctx      _context.Context
	publicId string
}

func (a *SyntheticsApi) buildGetTestRequest(ctx _context.Context, publicId string) (apiGetTestRequest, error) {
	req := apiGetTestRequest{
		ctx:      ctx,
		publicId: publicId,
	}
	return req, nil
}

// GetTest Get a test configuration.
// Get the detailed configuration associated with a Synthetics test.
func (a *SyntheticsApi) GetTest(ctx _context.Context, publicId string) (SyntheticsTestDetails, *_nethttp.Response, error) {
	req, err := a.buildGetTestRequest(ctx, publicId)
	if err != nil {
		var localVarReturnValue SyntheticsTestDetails
		return localVarReturnValue, nil, err
	}

	return a.getTestExecute(req)
}

// getTestExecute executes the request.
func (a *SyntheticsApi) getTestExecute(r apiGetTestRequest) (SyntheticsTestDetails, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsTestDetails
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.GetTest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/{public_id}"
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

type apiListGlobalVariablesRequest struct {
	ctx _context.Context
}

func (a *SyntheticsApi) buildListGlobalVariablesRequest(ctx _context.Context) (apiListGlobalVariablesRequest, error) {
	req := apiListGlobalVariablesRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListGlobalVariables Get all global variables.
// Get the list of all Synthetics global variables.
func (a *SyntheticsApi) ListGlobalVariables(ctx _context.Context) (SyntheticsListGlobalVariablesResponse, *_nethttp.Response, error) {
	req, err := a.buildListGlobalVariablesRequest(ctx)
	if err != nil {
		var localVarReturnValue SyntheticsListGlobalVariablesResponse
		return localVarReturnValue, nil, err
	}

	return a.listGlobalVariablesExecute(req)
}

// listGlobalVariablesExecute executes the request.
func (a *SyntheticsApi) listGlobalVariablesExecute(r apiListGlobalVariablesRequest) (SyntheticsListGlobalVariablesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsListGlobalVariablesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.ListGlobalVariables")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/variables"

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

type apiListLocationsRequest struct {
	ctx _context.Context
}

func (a *SyntheticsApi) buildListLocationsRequest(ctx _context.Context) (apiListLocationsRequest, error) {
	req := apiListLocationsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListLocations Get all locations (public and private).
// Get the list of public and private locations available for Synthetic
// tests. No arguments required.
func (a *SyntheticsApi) ListLocations(ctx _context.Context) (SyntheticsLocations, *_nethttp.Response, error) {
	req, err := a.buildListLocationsRequest(ctx)
	if err != nil {
		var localVarReturnValue SyntheticsLocations
		return localVarReturnValue, nil, err
	}

	return a.listLocationsExecute(req)
}

// listLocationsExecute executes the request.
func (a *SyntheticsApi) listLocationsExecute(r apiListLocationsRequest) (SyntheticsLocations, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsLocations
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.ListLocations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/locations"

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
		if localVarHTTPResponse.StatusCode == 429 {
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

type apiListTestsRequest struct {
	ctx        _context.Context
	pageSize   *string
	pageNumber *string
}

// ListTestsOptionalParameters holds optional parameters for ListTests.
type ListTestsOptionalParameters struct {
	PageSize   *string
	PageNumber *string
}

// NewListTestsOptionalParameters creates an empty struct for parameters.
func NewListTestsOptionalParameters() *ListTestsOptionalParameters {
	this := ListTestsOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *ListTestsOptionalParameters) WithPageSize(pageSize string) *ListTestsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *ListTestsOptionalParameters) WithPageNumber(pageNumber string) *ListTestsOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

func (a *SyntheticsApi) buildListTestsRequest(ctx _context.Context, o ...ListTestsOptionalParameters) (apiListTestsRequest, error) {
	req := apiListTestsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListTestsOptionalParameters is allowed")
	}

	if o != nil {
		req.pageSize = o[0].PageSize
		req.pageNumber = o[0].PageNumber
	}
	return req, nil
}

// ListTests Get the list of all Synthetic tests.
// Get the list of all Synthetic tests.
func (a *SyntheticsApi) ListTests(ctx _context.Context, o ...ListTestsOptionalParameters) (SyntheticsListTestsResponse, *_nethttp.Response, error) {
	req, err := a.buildListTestsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue SyntheticsListTestsResponse
		return localVarReturnValue, nil, err
	}

	return a.listTestsExecute(req)
}

// listTestsExecute executes the request.
func (a *SyntheticsApi) listTestsExecute(r apiListTestsRequest) (SyntheticsListTestsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsListTestsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.ListTests")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.pageSize != nil {
		localVarQueryParams.Add("page_size", datadog.ParameterToString(*r.pageSize, ""))
	}
	if r.pageNumber != nil {
		localVarQueryParams.Add("page_number", datadog.ParameterToString(*r.pageNumber, ""))
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

type apiTriggerCITestsRequest struct {
	ctx  _context.Context
	body *SyntheticsCITestBody
}

func (a *SyntheticsApi) buildTriggerCITestsRequest(ctx _context.Context, body SyntheticsCITestBody) (apiTriggerCITestsRequest, error) {
	req := apiTriggerCITestsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// TriggerCITests Trigger tests from CI/CD pipelines.
// Trigger a set of Synthetics tests for continuous integration.
func (a *SyntheticsApi) TriggerCITests(ctx _context.Context, body SyntheticsCITestBody) (SyntheticsTriggerCITestsResponse, *_nethttp.Response, error) {
	req, err := a.buildTriggerCITestsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsTriggerCITestsResponse
		return localVarReturnValue, nil, err
	}

	return a.triggerCITestsExecute(req)
}

// triggerCITestsExecute executes the request.
func (a *SyntheticsApi) triggerCITestsExecute(r apiTriggerCITestsRequest) (SyntheticsTriggerCITestsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsTriggerCITestsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.TriggerCITests")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/trigger/ci"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 429 {
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

type apiTriggerTestsRequest struct {
	ctx  _context.Context
	body *SyntheticsTriggerBody
}

func (a *SyntheticsApi) buildTriggerTestsRequest(ctx _context.Context, body SyntheticsTriggerBody) (apiTriggerTestsRequest, error) {
	req := apiTriggerTestsRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// TriggerTests Trigger Synthetics tests.
// Trigger a set of Synthetics tests.
func (a *SyntheticsApi) TriggerTests(ctx _context.Context, body SyntheticsTriggerBody) (SyntheticsTriggerCITestsResponse, *_nethttp.Response, error) {
	req, err := a.buildTriggerTestsRequest(ctx, body)
	if err != nil {
		var localVarReturnValue SyntheticsTriggerCITestsResponse
		return localVarReturnValue, nil, err
	}

	return a.triggerTestsExecute(req)
}

// triggerTestsExecute executes the request.
func (a *SyntheticsApi) triggerTestsExecute(r apiTriggerTestsRequest) (SyntheticsTriggerCITestsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsTriggerCITestsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.TriggerTests")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/trigger"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateAPITestRequest struct {
	ctx      _context.Context
	publicId string
	body     *SyntheticsAPITest
}

func (a *SyntheticsApi) buildUpdateAPITestRequest(ctx _context.Context, publicId string, body SyntheticsAPITest) (apiUpdateAPITestRequest, error) {
	req := apiUpdateAPITestRequest{
		ctx:      ctx,
		publicId: publicId,
		body:     &body,
	}
	return req, nil
}

// UpdateAPITest Edit an API test.
// Edit the configuration of a Synthetic API test.
func (a *SyntheticsApi) UpdateAPITest(ctx _context.Context, publicId string, body SyntheticsAPITest) (SyntheticsAPITest, *_nethttp.Response, error) {
	req, err := a.buildUpdateAPITestRequest(ctx, publicId, body)
	if err != nil {
		var localVarReturnValue SyntheticsAPITest
		return localVarReturnValue, nil, err
	}

	return a.updateAPITestExecute(req)
}

// updateAPITestExecute executes the request.
func (a *SyntheticsApi) updateAPITestExecute(r apiUpdateAPITestRequest) (SyntheticsAPITest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsAPITest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.UpdateAPITest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/api/{public_id}"
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

type apiUpdateBrowserTestRequest struct {
	ctx      _context.Context
	publicId string
	body     *SyntheticsBrowserTest
}

func (a *SyntheticsApi) buildUpdateBrowserTestRequest(ctx _context.Context, publicId string, body SyntheticsBrowserTest) (apiUpdateBrowserTestRequest, error) {
	req := apiUpdateBrowserTestRequest{
		ctx:      ctx,
		publicId: publicId,
		body:     &body,
	}
	return req, nil
}

// UpdateBrowserTest Edit a browser test.
// Edit the configuration of a Synthetic browser test.
func (a *SyntheticsApi) UpdateBrowserTest(ctx _context.Context, publicId string, body SyntheticsBrowserTest) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	req, err := a.buildUpdateBrowserTestRequest(ctx, publicId, body)
	if err != nil {
		var localVarReturnValue SyntheticsBrowserTest
		return localVarReturnValue, nil, err
	}

	return a.updateBrowserTestExecute(req)
}

// updateBrowserTestExecute executes the request.
func (a *SyntheticsApi) updateBrowserTestExecute(r apiUpdateBrowserTestRequest) (SyntheticsBrowserTest, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsBrowserTest
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.UpdateBrowserTest")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/browser/{public_id}"
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

type apiUpdatePrivateLocationRequest struct {
	ctx        _context.Context
	locationId string
	body       *SyntheticsPrivateLocation
}

func (a *SyntheticsApi) buildUpdatePrivateLocationRequest(ctx _context.Context, locationId string, body SyntheticsPrivateLocation) (apiUpdatePrivateLocationRequest, error) {
	req := apiUpdatePrivateLocationRequest{
		ctx:        ctx,
		locationId: locationId,
		body:       &body,
	}
	return req, nil
}

// UpdatePrivateLocation Edit a private location.
// Edit a Synthetics private location.
func (a *SyntheticsApi) UpdatePrivateLocation(ctx _context.Context, locationId string, body SyntheticsPrivateLocation) (SyntheticsPrivateLocation, *_nethttp.Response, error) {
	req, err := a.buildUpdatePrivateLocationRequest(ctx, locationId, body)
	if err != nil {
		var localVarReturnValue SyntheticsPrivateLocation
		return localVarReturnValue, nil, err
	}

	return a.updatePrivateLocationExecute(req)
}

// updatePrivateLocationExecute executes the request.
func (a *SyntheticsApi) updatePrivateLocationExecute(r apiUpdatePrivateLocationRequest) (SyntheticsPrivateLocation, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue SyntheticsPrivateLocation
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.UpdatePrivateLocation")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/private-locations/{location_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"location_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.locationId, "")), -1)

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
		if localVarHTTPResponse.StatusCode == 404 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateTestPauseStatusRequest struct {
	ctx      _context.Context
	publicId string
	body     *SyntheticsUpdateTestPauseStatusPayload
}

func (a *SyntheticsApi) buildUpdateTestPauseStatusRequest(ctx _context.Context, publicId string, body SyntheticsUpdateTestPauseStatusPayload) (apiUpdateTestPauseStatusRequest, error) {
	req := apiUpdateTestPauseStatusRequest{
		ctx:      ctx,
		publicId: publicId,
		body:     &body,
	}
	return req, nil
}

// UpdateTestPauseStatus Pause or start a test.
// Pause or start a Synthetics test by changing the status.
func (a *SyntheticsApi) UpdateTestPauseStatus(ctx _context.Context, publicId string, body SyntheticsUpdateTestPauseStatusPayload) (bool, *_nethttp.Response, error) {
	req, err := a.buildUpdateTestPauseStatusRequest(ctx, publicId, body)
	if err != nil {
		var localVarReturnValue bool
		return localVarReturnValue, nil, err
	}

	return a.updateTestPauseStatusExecute(req)
}

// updateTestPauseStatusExecute executes the request.
func (a *SyntheticsApi) updateTestPauseStatusExecute(r apiUpdateTestPauseStatusRequest) (bool, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue bool
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SyntheticsApi.UpdateTestPauseStatus")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/synthetics/tests/{public_id}/status"
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

// NewSyntheticsApi Returns NewSyntheticsApi.
func NewSyntheticsApi(client *datadog.APIClient) *SyntheticsApi {
	return &SyntheticsApi{
		Client: client,
	}
}
