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

// LogsMetricsApi service type
type LogsMetricsApi datadog.Service

type apiCreateLogsMetricRequest struct {
	ctx  _context.Context
	body *LogsMetricCreateRequest
}

func (a *LogsMetricsApi) buildCreateLogsMetricRequest(ctx _context.Context, body LogsMetricCreateRequest) (apiCreateLogsMetricRequest, error) {
	req := apiCreateLogsMetricRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateLogsMetric Create a log-based metric.
// Create a metric based on your ingested logs in your organization.
// Returns the log-based metric object from the request body when the request is successful.
func (a *LogsMetricsApi) CreateLogsMetric(ctx _context.Context, body LogsMetricCreateRequest) (LogsMetricResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateLogsMetricRequest(ctx, body)
	if err != nil {
		var localVarReturnValue LogsMetricResponse
		return localVarReturnValue, nil, err
	}

	return a.createLogsMetricExecute(req)
}

// createLogsMetricExecute executes the request.
func (a *LogsMetricsApi) createLogsMetricExecute(r apiCreateLogsMetricRequest) (LogsMetricResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsMetricResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsMetricsApi.CreateLogsMetric")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/metrics"

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

type apiDeleteLogsMetricRequest struct {
	ctx      _context.Context
	metricId string
}

func (a *LogsMetricsApi) buildDeleteLogsMetricRequest(ctx _context.Context, metricId string) (apiDeleteLogsMetricRequest, error) {
	req := apiDeleteLogsMetricRequest{
		ctx:      ctx,
		metricId: metricId,
	}
	return req, nil
}

// DeleteLogsMetric Delete a log-based metric.
// Delete a specific log-based metric from your organization.
func (a *LogsMetricsApi) DeleteLogsMetric(ctx _context.Context, metricId string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteLogsMetricRequest(ctx, metricId)
	if err != nil {
		return nil, err
	}

	return a.deleteLogsMetricExecute(req)
}

// deleteLogsMetricExecute executes the request.
func (a *LogsMetricsApi) deleteLogsMetricExecute(r apiDeleteLogsMetricRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsMetricsApi.DeleteLogsMetric")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/metrics/{metric_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricId, "")), -1)

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

type apiGetLogsMetricRequest struct {
	ctx      _context.Context
	metricId string
}

func (a *LogsMetricsApi) buildGetLogsMetricRequest(ctx _context.Context, metricId string) (apiGetLogsMetricRequest, error) {
	req := apiGetLogsMetricRequest{
		ctx:      ctx,
		metricId: metricId,
	}
	return req, nil
}

// GetLogsMetric Get a log-based metric.
// Get a specific log-based metric from your organization.
func (a *LogsMetricsApi) GetLogsMetric(ctx _context.Context, metricId string) (LogsMetricResponse, *_nethttp.Response, error) {
	req, err := a.buildGetLogsMetricRequest(ctx, metricId)
	if err != nil {
		var localVarReturnValue LogsMetricResponse
		return localVarReturnValue, nil, err
	}

	return a.getLogsMetricExecute(req)
}

// getLogsMetricExecute executes the request.
func (a *LogsMetricsApi) getLogsMetricExecute(r apiGetLogsMetricRequest) (LogsMetricResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsMetricResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsMetricsApi.GetLogsMetric")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/metrics/{metric_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricId, "")), -1)

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

type apiListLogsMetricsRequest struct {
	ctx _context.Context
}

func (a *LogsMetricsApi) buildListLogsMetricsRequest(ctx _context.Context) (apiListLogsMetricsRequest, error) {
	req := apiListLogsMetricsRequest{
		ctx: ctx,
	}
	return req, nil
}

// ListLogsMetrics Get all log-based metrics.
// Get the list of configured log-based metrics with their definitions.
func (a *LogsMetricsApi) ListLogsMetrics(ctx _context.Context) (LogsMetricsResponse, *_nethttp.Response, error) {
	req, err := a.buildListLogsMetricsRequest(ctx)
	if err != nil {
		var localVarReturnValue LogsMetricsResponse
		return localVarReturnValue, nil, err
	}

	return a.listLogsMetricsExecute(req)
}

// listLogsMetricsExecute executes the request.
func (a *LogsMetricsApi) listLogsMetricsExecute(r apiListLogsMetricsRequest) (LogsMetricsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue LogsMetricsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsMetricsApi.ListLogsMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/metrics"

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

type apiUpdateLogsMetricRequest struct {
	ctx      _context.Context
	metricId string
	body     *LogsMetricUpdateRequest
}

func (a *LogsMetricsApi) buildUpdateLogsMetricRequest(ctx _context.Context, metricId string, body LogsMetricUpdateRequest) (apiUpdateLogsMetricRequest, error) {
	req := apiUpdateLogsMetricRequest{
		ctx:      ctx,
		metricId: metricId,
		body:     &body,
	}
	return req, nil
}

// UpdateLogsMetric Update a log-based metric.
// Update a specific log-based metric from your organization.
// Returns the log-based metric object from the request body when the request is successful.
func (a *LogsMetricsApi) UpdateLogsMetric(ctx _context.Context, metricId string, body LogsMetricUpdateRequest) (LogsMetricResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateLogsMetricRequest(ctx, metricId, body)
	if err != nil {
		var localVarReturnValue LogsMetricResponse
		return localVarReturnValue, nil, err
	}

	return a.updateLogsMetricExecute(req)
}

// updateLogsMetricExecute executes the request.
func (a *LogsMetricsApi) updateLogsMetricExecute(r apiUpdateLogsMetricRequest) (LogsMetricResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue LogsMetricResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.LogsMetricsApi.UpdateLogsMetric")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/logs/config/metrics/{metric_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_id"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricId, "")), -1)

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

// NewLogsMetricsApi Returns NewLogsMetricsApi.
func NewLogsMetricsApi(client *datadog.APIClient) *LogsMetricsApi {
	return &LogsMetricsApi{
		Client: client,
	}
}
