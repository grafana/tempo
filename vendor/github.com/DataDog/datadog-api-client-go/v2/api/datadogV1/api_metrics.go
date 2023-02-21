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

// MetricsApi service type
type MetricsApi datadog.Service

type apiGetMetricMetadataRequest struct {
	ctx        _context.Context
	metricName string
}

func (a *MetricsApi) buildGetMetricMetadataRequest(ctx _context.Context, metricName string) (apiGetMetricMetadataRequest, error) {
	req := apiGetMetricMetadataRequest{
		ctx:        ctx,
		metricName: metricName,
	}
	return req, nil
}

// GetMetricMetadata Get metric metadata.
// Get metadata about a specific metric.
func (a *MetricsApi) GetMetricMetadata(ctx _context.Context, metricName string) (MetricMetadata, *_nethttp.Response, error) {
	req, err := a.buildGetMetricMetadataRequest(ctx, metricName)
	if err != nil {
		var localVarReturnValue MetricMetadata
		return localVarReturnValue, nil, err
	}

	return a.getMetricMetadataExecute(req)
}

// getMetricMetadataExecute executes the request.
func (a *MetricsApi) getMetricMetadataExecute(r apiGetMetricMetadataRequest) (MetricMetadata, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricMetadata
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.GetMetricMetadata")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/metrics/{metric_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricName, "")), -1)

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

type apiListActiveMetricsRequest struct {
	ctx       _context.Context
	from      *int64
	host      *string
	tagFilter *string
}

// ListActiveMetricsOptionalParameters holds optional parameters for ListActiveMetrics.
type ListActiveMetricsOptionalParameters struct {
	Host      *string
	TagFilter *string
}

// NewListActiveMetricsOptionalParameters creates an empty struct for parameters.
func NewListActiveMetricsOptionalParameters() *ListActiveMetricsOptionalParameters {
	this := ListActiveMetricsOptionalParameters{}
	return &this
}

// WithHost sets the corresponding parameter name and returns the struct.
func (r *ListActiveMetricsOptionalParameters) WithHost(host string) *ListActiveMetricsOptionalParameters {
	r.Host = &host
	return r
}

// WithTagFilter sets the corresponding parameter name and returns the struct.
func (r *ListActiveMetricsOptionalParameters) WithTagFilter(tagFilter string) *ListActiveMetricsOptionalParameters {
	r.TagFilter = &tagFilter
	return r
}

func (a *MetricsApi) buildListActiveMetricsRequest(ctx _context.Context, from int64, o ...ListActiveMetricsOptionalParameters) (apiListActiveMetricsRequest, error) {
	req := apiListActiveMetricsRequest{
		ctx:  ctx,
		from: &from,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListActiveMetricsOptionalParameters is allowed")
	}

	if o != nil {
		req.host = o[0].Host
		req.tagFilter = o[0].TagFilter
	}
	return req, nil
}

// ListActiveMetrics Get active metrics list.
// Get the list of actively reporting metrics from a given time until now.
func (a *MetricsApi) ListActiveMetrics(ctx _context.Context, from int64, o ...ListActiveMetricsOptionalParameters) (MetricsListResponse, *_nethttp.Response, error) {
	req, err := a.buildListActiveMetricsRequest(ctx, from, o...)
	if err != nil {
		var localVarReturnValue MetricsListResponse
		return localVarReturnValue, nil, err
	}

	return a.listActiveMetricsExecute(req)
}

// listActiveMetricsExecute executes the request.
func (a *MetricsApi) listActiveMetricsExecute(r apiListActiveMetricsRequest) (MetricsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricsListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.ListActiveMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/metrics"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.from == nil {
		return localVarReturnValue, nil, datadog.ReportError("from is required and must be specified")
	}
	localVarQueryParams.Add("from", datadog.ParameterToString(*r.from, ""))
	if r.host != nil {
		localVarQueryParams.Add("host", datadog.ParameterToString(*r.host, ""))
	}
	if r.tagFilter != nil {
		localVarQueryParams.Add("tag_filter", datadog.ParameterToString(*r.tagFilter, ""))
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

type apiListMetricsRequest struct {
	ctx _context.Context
	q   *string
}

func (a *MetricsApi) buildListMetricsRequest(ctx _context.Context, q string) (apiListMetricsRequest, error) {
	req := apiListMetricsRequest{
		ctx: ctx,
		q:   &q,
	}
	return req, nil
}

// ListMetrics Search metrics.
// Search for metrics from the last 24 hours in Datadog.
func (a *MetricsApi) ListMetrics(ctx _context.Context, q string) (MetricSearchResponse, *_nethttp.Response, error) {
	req, err := a.buildListMetricsRequest(ctx, q)
	if err != nil {
		var localVarReturnValue MetricSearchResponse
		return localVarReturnValue, nil, err
	}

	return a.listMetricsExecute(req)
}

// listMetricsExecute executes the request.
func (a *MetricsApi) listMetricsExecute(r apiListMetricsRequest) (MetricSearchResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricSearchResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.ListMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/search"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.q == nil {
		return localVarReturnValue, nil, datadog.ReportError("q is required and must be specified")
	}
	localVarQueryParams.Add("q", datadog.ParameterToString(*r.q, ""))
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

type apiQueryMetricsRequest struct {
	ctx   _context.Context
	from  *int64
	to    *int64
	query *string
}

func (a *MetricsApi) buildQueryMetricsRequest(ctx _context.Context, from int64, to int64, query string) (apiQueryMetricsRequest, error) {
	req := apiQueryMetricsRequest{
		ctx:   ctx,
		from:  &from,
		to:    &to,
		query: &query,
	}
	return req, nil
}

// QueryMetrics Query timeseries points.
// Query timeseries points.
func (a *MetricsApi) QueryMetrics(ctx _context.Context, from int64, to int64, query string) (MetricsQueryResponse, *_nethttp.Response, error) {
	req, err := a.buildQueryMetricsRequest(ctx, from, to, query)
	if err != nil {
		var localVarReturnValue MetricsQueryResponse
		return localVarReturnValue, nil, err
	}

	return a.queryMetricsExecute(req)
}

// queryMetricsExecute executes the request.
func (a *MetricsApi) queryMetricsExecute(r apiQueryMetricsRequest) (MetricsQueryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricsQueryResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.QueryMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/query"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.from == nil {
		return localVarReturnValue, nil, datadog.ReportError("from is required and must be specified")
	}
	if r.to == nil {
		return localVarReturnValue, nil, datadog.ReportError("to is required and must be specified")
	}
	if r.query == nil {
		return localVarReturnValue, nil, datadog.ReportError("query is required and must be specified")
	}
	localVarQueryParams.Add("from", datadog.ParameterToString(*r.from, ""))
	localVarQueryParams.Add("to", datadog.ParameterToString(*r.to, ""))
	localVarQueryParams.Add("query", datadog.ParameterToString(*r.query, ""))
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

type apiSubmitDistributionPointsRequest struct {
	ctx             _context.Context
	body            *DistributionPointsPayload
	contentEncoding *DistributionPointsContentEncoding
}

// SubmitDistributionPointsOptionalParameters holds optional parameters for SubmitDistributionPoints.
type SubmitDistributionPointsOptionalParameters struct {
	ContentEncoding *DistributionPointsContentEncoding
}

// NewSubmitDistributionPointsOptionalParameters creates an empty struct for parameters.
func NewSubmitDistributionPointsOptionalParameters() *SubmitDistributionPointsOptionalParameters {
	this := SubmitDistributionPointsOptionalParameters{}
	return &this
}

// WithContentEncoding sets the corresponding parameter name and returns the struct.
func (r *SubmitDistributionPointsOptionalParameters) WithContentEncoding(contentEncoding DistributionPointsContentEncoding) *SubmitDistributionPointsOptionalParameters {
	r.ContentEncoding = &contentEncoding
	return r
}

func (a *MetricsApi) buildSubmitDistributionPointsRequest(ctx _context.Context, body DistributionPointsPayload, o ...SubmitDistributionPointsOptionalParameters) (apiSubmitDistributionPointsRequest, error) {
	req := apiSubmitDistributionPointsRequest{
		ctx:  ctx,
		body: &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SubmitDistributionPointsOptionalParameters is allowed")
	}

	if o != nil {
		req.contentEncoding = o[0].ContentEncoding
	}
	return req, nil
}

// SubmitDistributionPoints Submit distribution points.
// The distribution points end-point allows you to post distribution data that can be graphed on Datadog’s dashboards.
func (a *MetricsApi) SubmitDistributionPoints(ctx _context.Context, body DistributionPointsPayload, o ...SubmitDistributionPointsOptionalParameters) (IntakePayloadAccepted, *_nethttp.Response, error) {
	req, err := a.buildSubmitDistributionPointsRequest(ctx, body, o...)
	if err != nil {
		var localVarReturnValue IntakePayloadAccepted
		return localVarReturnValue, nil, err
	}

	return a.submitDistributionPointsExecute(req)
}

// submitDistributionPointsExecute executes the request.
func (a *MetricsApi) submitDistributionPointsExecute(r apiSubmitDistributionPointsRequest) (IntakePayloadAccepted, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IntakePayloadAccepted
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.SubmitDistributionPoints")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/distribution_points"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "text/json"
	localVarHeaderParams["Accept"] = "application/json"

	if r.contentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = datadog.ParameterToString(*r.contentEncoding, "")
	}

	// body params
	localVarPostBody = r.body
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 408 || localVarHTTPResponse.StatusCode == 413 || localVarHTTPResponse.StatusCode == 429 {
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

type apiSubmitMetricsRequest struct {
	ctx             _context.Context
	body            *MetricsPayload
	contentEncoding *MetricContentEncoding
}

// SubmitMetricsOptionalParameters holds optional parameters for SubmitMetrics.
type SubmitMetricsOptionalParameters struct {
	ContentEncoding *MetricContentEncoding
}

// NewSubmitMetricsOptionalParameters creates an empty struct for parameters.
func NewSubmitMetricsOptionalParameters() *SubmitMetricsOptionalParameters {
	this := SubmitMetricsOptionalParameters{}
	return &this
}

// WithContentEncoding sets the corresponding parameter name and returns the struct.
func (r *SubmitMetricsOptionalParameters) WithContentEncoding(contentEncoding MetricContentEncoding) *SubmitMetricsOptionalParameters {
	r.ContentEncoding = &contentEncoding
	return r
}

func (a *MetricsApi) buildSubmitMetricsRequest(ctx _context.Context, body MetricsPayload, o ...SubmitMetricsOptionalParameters) (apiSubmitMetricsRequest, error) {
	req := apiSubmitMetricsRequest{
		ctx:  ctx,
		body: &body,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type SubmitMetricsOptionalParameters is allowed")
	}

	if o != nil {
		req.contentEncoding = o[0].ContentEncoding
	}
	return req, nil
}

// SubmitMetrics Submit metrics.
// The metrics end-point allows you to post time-series data that can be graphed on Datadog’s dashboards.
// The maximum payload size is 3.2 megabytes (3200000 bytes). Compressed payloads must have a decompressed size of less than 62 megabytes (62914560 bytes).
//
// If you’re submitting metrics directly to the Datadog API without using DogStatsD, expect:
//
// - 64 bits for the timestamp
// - 64 bits for the value
// - 40 bytes for the metric names
// - 50 bytes for the timeseries
// - The full payload is approximately 100 bytes. However, with the DogStatsD API,
// compression is applied, which reduces the payload size.
func (a *MetricsApi) SubmitMetrics(ctx _context.Context, body MetricsPayload, o ...SubmitMetricsOptionalParameters) (IntakePayloadAccepted, *_nethttp.Response, error) {
	req, err := a.buildSubmitMetricsRequest(ctx, body, o...)
	if err != nil {
		var localVarReturnValue IntakePayloadAccepted
		return localVarReturnValue, nil, err
	}

	return a.submitMetricsExecute(req)
}

// submitMetricsExecute executes the request.
func (a *MetricsApi) submitMetricsExecute(r apiSubmitMetricsRequest) (IntakePayloadAccepted, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IntakePayloadAccepted
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.SubmitMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/series"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "text/json"
	localVarHeaderParams["Accept"] = "application/json"

	if r.contentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = datadog.ParameterToString(*r.contentEncoding, "")
	}

	// body params
	localVarPostBody = r.body
	datadog.SetAuthKeys(
		r.ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 408 || localVarHTTPResponse.StatusCode == 413 || localVarHTTPResponse.StatusCode == 429 {
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

type apiUpdateMetricMetadataRequest struct {
	ctx        _context.Context
	metricName string
	body       *MetricMetadata
}

func (a *MetricsApi) buildUpdateMetricMetadataRequest(ctx _context.Context, metricName string, body MetricMetadata) (apiUpdateMetricMetadataRequest, error) {
	req := apiUpdateMetricMetadataRequest{
		ctx:        ctx,
		metricName: metricName,
		body:       &body,
	}
	return req, nil
}

// UpdateMetricMetadata Edit metric metadata.
// Edit metadata of a specific metric. Find out more about [supported types](https://docs.datadoghq.com/developers/metrics).
func (a *MetricsApi) UpdateMetricMetadata(ctx _context.Context, metricName string, body MetricMetadata) (MetricMetadata, *_nethttp.Response, error) {
	req, err := a.buildUpdateMetricMetadataRequest(ctx, metricName, body)
	if err != nil {
		var localVarReturnValue MetricMetadata
		return localVarReturnValue, nil, err
	}

	return a.updateMetricMetadataExecute(req)
}

// updateMetricMetadataExecute executes the request.
func (a *MetricsApi) updateMetricMetadataExecute(r apiUpdateMetricMetadataRequest) (MetricMetadata, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPut
		localVarPostBody    interface{}
		localVarReturnValue MetricMetadata
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.MetricsApi.UpdateMetricMetadata")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/metrics/{metric_name}"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricName, "")), -1)

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

// NewMetricsApi Returns NewMetricsApi.
func NewMetricsApi(client *datadog.APIClient) *MetricsApi {
	return &MetricsApi{
		Client: client,
	}
}
