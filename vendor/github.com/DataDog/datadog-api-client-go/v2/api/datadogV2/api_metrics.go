// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_fmt "fmt"
	_log "log"
	_nethttp "net/http"
	_neturl "net/url"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// MetricsApi service type
type MetricsApi datadog.Service

type apiCreateBulkTagsMetricsConfigurationRequest struct {
	ctx  _context.Context
	body *MetricBulkTagConfigCreateRequest
}

func (a *MetricsApi) buildCreateBulkTagsMetricsConfigurationRequest(ctx _context.Context, body MetricBulkTagConfigCreateRequest) (apiCreateBulkTagsMetricsConfigurationRequest, error) {
	req := apiCreateBulkTagsMetricsConfigurationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// CreateBulkTagsMetricsConfiguration Configure tags for multiple metrics.
// Create and define a list of queryable tag keys for a set of existing count, gauge, rate, and distribution metrics.
// Metrics are selected by passing a metric name prefix. Use the Delete method of this API path to remove tag configurations.
// Results can be sent to a set of account email addresses, just like the same operation in the Datadog web app.
// If multiple calls include the same metric, the last configuration applied (not by submit order) is used, do not
// expect deterministic ordering of concurrent calls.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) CreateBulkTagsMetricsConfiguration(ctx _context.Context, body MetricBulkTagConfigCreateRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateBulkTagsMetricsConfigurationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue MetricBulkTagConfigResponse
		return localVarReturnValue, nil, err
	}

	return a.createBulkTagsMetricsConfigurationExecute(req)
}

// createBulkTagsMetricsConfigurationExecute executes the request.
func (a *MetricsApi) createBulkTagsMetricsConfigurationExecute(r apiCreateBulkTagsMetricsConfigurationRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue MetricBulkTagConfigResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.CreateBulkTagsMetricsConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/config/bulk-tags"

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

type apiCreateTagConfigurationRequest struct {
	ctx        _context.Context
	metricName string
	body       *MetricTagConfigurationCreateRequest
}

func (a *MetricsApi) buildCreateTagConfigurationRequest(ctx _context.Context, metricName string, body MetricTagConfigurationCreateRequest) (apiCreateTagConfigurationRequest, error) {
	req := apiCreateTagConfigurationRequest{
		ctx:        ctx,
		metricName: metricName,
		body:       &body,
	}
	return req, nil
}

// CreateTagConfiguration Create a tag configuration.
// Create and define a list of queryable tag keys for an existing count/gauge/rate/distribution metric.
// Optionally, include percentile aggregations on any distribution metric or configure custom aggregations
// on any count, rate, or gauge metric.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) CreateTagConfiguration(ctx _context.Context, metricName string, body MetricTagConfigurationCreateRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	req, err := a.buildCreateTagConfigurationRequest(ctx, metricName, body)
	if err != nil {
		var localVarReturnValue MetricTagConfigurationResponse
		return localVarReturnValue, nil, err
	}

	return a.createTagConfigurationExecute(req)
}

// createTagConfigurationExecute executes the request.
func (a *MetricsApi) createTagConfigurationExecute(r apiCreateTagConfigurationRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.CreateTagConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
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

type apiDeleteBulkTagsMetricsConfigurationRequest struct {
	ctx  _context.Context
	body *MetricBulkTagConfigDeleteRequest
}

func (a *MetricsApi) buildDeleteBulkTagsMetricsConfigurationRequest(ctx _context.Context, body MetricBulkTagConfigDeleteRequest) (apiDeleteBulkTagsMetricsConfigurationRequest, error) {
	req := apiDeleteBulkTagsMetricsConfigurationRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// DeleteBulkTagsMetricsConfiguration Configure tags for multiple metrics.
// Delete all custom lists of queryable tag keys for a set of existing count, gauge, rate, and distribution metrics.
// Metrics are selected by passing a metric name prefix.
// Results can be sent to a set of account email addresses, just like the same operation in the Datadog web app.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) DeleteBulkTagsMetricsConfiguration(ctx _context.Context, body MetricBulkTagConfigDeleteRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	req, err := a.buildDeleteBulkTagsMetricsConfigurationRequest(ctx, body)
	if err != nil {
		var localVarReturnValue MetricBulkTagConfigResponse
		return localVarReturnValue, nil, err
	}

	return a.deleteBulkTagsMetricsConfigurationExecute(req)
}

// deleteBulkTagsMetricsConfigurationExecute executes the request.
func (a *MetricsApi) deleteBulkTagsMetricsConfigurationExecute(r apiDeleteBulkTagsMetricsConfigurationRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue MetricBulkTagConfigResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.DeleteBulkTagsMetricsConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/config/bulk-tags"

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

type apiDeleteTagConfigurationRequest struct {
	ctx        _context.Context
	metricName string
}

func (a *MetricsApi) buildDeleteTagConfigurationRequest(ctx _context.Context, metricName string) (apiDeleteTagConfigurationRequest, error) {
	req := apiDeleteTagConfigurationRequest{
		ctx:        ctx,
		metricName: metricName,
	}
	return req, nil
}

// DeleteTagConfiguration Delete a tag configuration.
// Deletes a metric's tag configuration. Can only be used with application
// keys from users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) DeleteTagConfiguration(ctx _context.Context, metricName string) (*_nethttp.Response, error) {
	req, err := a.buildDeleteTagConfigurationRequest(ctx, metricName)
	if err != nil {
		return nil, err
	}

	return a.deleteTagConfigurationExecute(req)
}

// deleteTagConfigurationExecute executes the request.
func (a *MetricsApi) deleteTagConfigurationExecute(r apiDeleteTagConfigurationRequest) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.DeleteTagConfiguration")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricName, "")), -1)

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

type apiEstimateMetricsOutputSeriesRequest struct {
	ctx                   _context.Context
	metricName            string
	filterGroups          *string
	filterHoursAgo        *int32
	filterNumAggregations *int32
	filterPct             *bool
	filterTimespanH       *int32
}

// EstimateMetricsOutputSeriesOptionalParameters holds optional parameters for EstimateMetricsOutputSeries.
type EstimateMetricsOutputSeriesOptionalParameters struct {
	FilterGroups          *string
	FilterHoursAgo        *int32
	FilterNumAggregations *int32
	FilterPct             *bool
	FilterTimespanH       *int32
}

// NewEstimateMetricsOutputSeriesOptionalParameters creates an empty struct for parameters.
func NewEstimateMetricsOutputSeriesOptionalParameters() *EstimateMetricsOutputSeriesOptionalParameters {
	this := EstimateMetricsOutputSeriesOptionalParameters{}
	return &this
}

// WithFilterGroups sets the corresponding parameter name and returns the struct.
func (r *EstimateMetricsOutputSeriesOptionalParameters) WithFilterGroups(filterGroups string) *EstimateMetricsOutputSeriesOptionalParameters {
	r.FilterGroups = &filterGroups
	return r
}

// WithFilterHoursAgo sets the corresponding parameter name and returns the struct.
func (r *EstimateMetricsOutputSeriesOptionalParameters) WithFilterHoursAgo(filterHoursAgo int32) *EstimateMetricsOutputSeriesOptionalParameters {
	r.FilterHoursAgo = &filterHoursAgo
	return r
}

// WithFilterNumAggregations sets the corresponding parameter name and returns the struct.
func (r *EstimateMetricsOutputSeriesOptionalParameters) WithFilterNumAggregations(filterNumAggregations int32) *EstimateMetricsOutputSeriesOptionalParameters {
	r.FilterNumAggregations = &filterNumAggregations
	return r
}

// WithFilterPct sets the corresponding parameter name and returns the struct.
func (r *EstimateMetricsOutputSeriesOptionalParameters) WithFilterPct(filterPct bool) *EstimateMetricsOutputSeriesOptionalParameters {
	r.FilterPct = &filterPct
	return r
}

// WithFilterTimespanH sets the corresponding parameter name and returns the struct.
func (r *EstimateMetricsOutputSeriesOptionalParameters) WithFilterTimespanH(filterTimespanH int32) *EstimateMetricsOutputSeriesOptionalParameters {
	r.FilterTimespanH = &filterTimespanH
	return r
}

func (a *MetricsApi) buildEstimateMetricsOutputSeriesRequest(ctx _context.Context, metricName string, o ...EstimateMetricsOutputSeriesOptionalParameters) (apiEstimateMetricsOutputSeriesRequest, error) {
	req := apiEstimateMetricsOutputSeriesRequest{
		ctx:        ctx,
		metricName: metricName,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type EstimateMetricsOutputSeriesOptionalParameters is allowed")
	}

	if o != nil {
		req.filterGroups = o[0].FilterGroups
		req.filterHoursAgo = o[0].FilterHoursAgo
		req.filterNumAggregations = o[0].FilterNumAggregations
		req.filterPct = o[0].FilterPct
		req.filterTimespanH = o[0].FilterTimespanH
	}
	return req, nil
}

// EstimateMetricsOutputSeries Tag Configuration Cardinality Estimator.
// Returns the estimated cardinality for a metric with a given tag, percentile and number of aggregations configuration using Metrics without Limits&trade;.
func (a *MetricsApi) EstimateMetricsOutputSeries(ctx _context.Context, metricName string, o ...EstimateMetricsOutputSeriesOptionalParameters) (MetricEstimateResponse, *_nethttp.Response, error) {
	req, err := a.buildEstimateMetricsOutputSeriesRequest(ctx, metricName, o...)
	if err != nil {
		var localVarReturnValue MetricEstimateResponse
		return localVarReturnValue, nil, err
	}

	return a.estimateMetricsOutputSeriesExecute(req)
}

// estimateMetricsOutputSeriesExecute executes the request.
func (a *MetricsApi) estimateMetricsOutputSeriesExecute(r apiEstimateMetricsOutputSeriesRequest) (MetricEstimateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricEstimateResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.EstimateMetricsOutputSeries")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/estimate"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filterGroups != nil {
		localVarQueryParams.Add("filter[groups]", datadog.ParameterToString(*r.filterGroups, ""))
	}
	if r.filterHoursAgo != nil {
		localVarQueryParams.Add("filter[hours_ago]", datadog.ParameterToString(*r.filterHoursAgo, ""))
	}
	if r.filterNumAggregations != nil {
		localVarQueryParams.Add("filter[num_aggregations]", datadog.ParameterToString(*r.filterNumAggregations, ""))
	}
	if r.filterPct != nil {
		localVarQueryParams.Add("filter[pct]", datadog.ParameterToString(*r.filterPct, ""))
	}
	if r.filterTimespanH != nil {
		localVarQueryParams.Add("filter[timespan_h]", datadog.ParameterToString(*r.filterTimespanH, ""))
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

type apiListActiveMetricConfigurationsRequest struct {
	ctx           _context.Context
	metricName    string
	windowSeconds *int64
}

// ListActiveMetricConfigurationsOptionalParameters holds optional parameters for ListActiveMetricConfigurations.
type ListActiveMetricConfigurationsOptionalParameters struct {
	WindowSeconds *int64
}

// NewListActiveMetricConfigurationsOptionalParameters creates an empty struct for parameters.
func NewListActiveMetricConfigurationsOptionalParameters() *ListActiveMetricConfigurationsOptionalParameters {
	this := ListActiveMetricConfigurationsOptionalParameters{}
	return &this
}

// WithWindowSeconds sets the corresponding parameter name and returns the struct.
func (r *ListActiveMetricConfigurationsOptionalParameters) WithWindowSeconds(windowSeconds int64) *ListActiveMetricConfigurationsOptionalParameters {
	r.WindowSeconds = &windowSeconds
	return r
}

func (a *MetricsApi) buildListActiveMetricConfigurationsRequest(ctx _context.Context, metricName string, o ...ListActiveMetricConfigurationsOptionalParameters) (apiListActiveMetricConfigurationsRequest, error) {
	req := apiListActiveMetricConfigurationsRequest{
		ctx:        ctx,
		metricName: metricName,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListActiveMetricConfigurationsOptionalParameters is allowed")
	}

	if o != nil {
		req.windowSeconds = o[0].WindowSeconds
	}
	return req, nil
}

// ListActiveMetricConfigurations List active tags and aggregations.
// List tags and aggregations that are actively queried on dashboards and monitors for a given metric name.
func (a *MetricsApi) ListActiveMetricConfigurations(ctx _context.Context, metricName string, o ...ListActiveMetricConfigurationsOptionalParameters) (MetricSuggestedTagsAndAggregationsResponse, *_nethttp.Response, error) {
	req, err := a.buildListActiveMetricConfigurationsRequest(ctx, metricName, o...)
	if err != nil {
		var localVarReturnValue MetricSuggestedTagsAndAggregationsResponse
		return localVarReturnValue, nil, err
	}

	return a.listActiveMetricConfigurationsExecute(req)
}

// listActiveMetricConfigurationsExecute executes the request.
func (a *MetricsApi) listActiveMetricConfigurationsExecute(r apiListActiveMetricConfigurationsRequest) (MetricSuggestedTagsAndAggregationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricSuggestedTagsAndAggregationsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.ListActiveMetricConfigurations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/active-configurations"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(r.metricName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.windowSeconds != nil {
		localVarQueryParams.Add("window[seconds]", datadog.ParameterToString(*r.windowSeconds, ""))
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

type apiListTagConfigurationByNameRequest struct {
	ctx        _context.Context
	metricName string
}

func (a *MetricsApi) buildListTagConfigurationByNameRequest(ctx _context.Context, metricName string) (apiListTagConfigurationByNameRequest, error) {
	req := apiListTagConfigurationByNameRequest{
		ctx:        ctx,
		metricName: metricName,
	}
	return req, nil
}

// ListTagConfigurationByName List tag configuration by name.
// Returns the tag configuration for the given metric name.
func (a *MetricsApi) ListTagConfigurationByName(ctx _context.Context, metricName string) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	req, err := a.buildListTagConfigurationByNameRequest(ctx, metricName)
	if err != nil {
		var localVarReturnValue MetricTagConfigurationResponse
		return localVarReturnValue, nil, err
	}

	return a.listTagConfigurationByNameExecute(req)
}

// listTagConfigurationByNameExecute executes the request.
func (a *MetricsApi) listTagConfigurationByNameExecute(r apiListTagConfigurationByNameRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.ListTagConfigurationByName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
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

type apiListTagConfigurationsRequest struct {
	ctx                      _context.Context
	filterConfigured         *bool
	filterTagsConfigured     *string
	filterMetricType         *MetricTagConfigurationMetricTypes
	filterIncludePercentiles *bool
	filterQueried            *bool
	filterTags               *string
	windowSeconds            *int64
}

// ListTagConfigurationsOptionalParameters holds optional parameters for ListTagConfigurations.
type ListTagConfigurationsOptionalParameters struct {
	FilterConfigured         *bool
	FilterTagsConfigured     *string
	FilterMetricType         *MetricTagConfigurationMetricTypes
	FilterIncludePercentiles *bool
	FilterQueried            *bool
	FilterTags               *string
	WindowSeconds            *int64
}

// NewListTagConfigurationsOptionalParameters creates an empty struct for parameters.
func NewListTagConfigurationsOptionalParameters() *ListTagConfigurationsOptionalParameters {
	this := ListTagConfigurationsOptionalParameters{}
	return &this
}

// WithFilterConfigured sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterConfigured(filterConfigured bool) *ListTagConfigurationsOptionalParameters {
	r.FilterConfigured = &filterConfigured
	return r
}

// WithFilterTagsConfigured sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterTagsConfigured(filterTagsConfigured string) *ListTagConfigurationsOptionalParameters {
	r.FilterTagsConfigured = &filterTagsConfigured
	return r
}

// WithFilterMetricType sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterMetricType(filterMetricType MetricTagConfigurationMetricTypes) *ListTagConfigurationsOptionalParameters {
	r.FilterMetricType = &filterMetricType
	return r
}

// WithFilterIncludePercentiles sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterIncludePercentiles(filterIncludePercentiles bool) *ListTagConfigurationsOptionalParameters {
	r.FilterIncludePercentiles = &filterIncludePercentiles
	return r
}

// WithFilterQueried sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterQueried(filterQueried bool) *ListTagConfigurationsOptionalParameters {
	r.FilterQueried = &filterQueried
	return r
}

// WithFilterTags sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithFilterTags(filterTags string) *ListTagConfigurationsOptionalParameters {
	r.FilterTags = &filterTags
	return r
}

// WithWindowSeconds sets the corresponding parameter name and returns the struct.
func (r *ListTagConfigurationsOptionalParameters) WithWindowSeconds(windowSeconds int64) *ListTagConfigurationsOptionalParameters {
	r.WindowSeconds = &windowSeconds
	return r
}

func (a *MetricsApi) buildListTagConfigurationsRequest(ctx _context.Context, o ...ListTagConfigurationsOptionalParameters) (apiListTagConfigurationsRequest, error) {
	req := apiListTagConfigurationsRequest{
		ctx: ctx,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type ListTagConfigurationsOptionalParameters is allowed")
	}

	if o != nil {
		req.filterConfigured = o[0].FilterConfigured
		req.filterTagsConfigured = o[0].FilterTagsConfigured
		req.filterMetricType = o[0].FilterMetricType
		req.filterIncludePercentiles = o[0].FilterIncludePercentiles
		req.filterQueried = o[0].FilterQueried
		req.filterTags = o[0].FilterTags
		req.windowSeconds = o[0].WindowSeconds
	}
	return req, nil
}

// ListTagConfigurations Get a list of metrics.
// Returns all metrics that can be configured in the Metrics Summary page or with Metrics without Limits™ (matching additional filters if specified).
func (a *MetricsApi) ListTagConfigurations(ctx _context.Context, o ...ListTagConfigurationsOptionalParameters) (MetricsAndMetricTagConfigurationsResponse, *_nethttp.Response, error) {
	req, err := a.buildListTagConfigurationsRequest(ctx, o...)
	if err != nil {
		var localVarReturnValue MetricsAndMetricTagConfigurationsResponse
		return localVarReturnValue, nil, err
	}

	return a.listTagConfigurationsExecute(req)
}

// listTagConfigurationsExecute executes the request.
func (a *MetricsApi) listTagConfigurationsExecute(r apiListTagConfigurationsRequest) (MetricsAndMetricTagConfigurationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricsAndMetricTagConfigurationsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.ListTagConfigurations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.filterConfigured != nil {
		localVarQueryParams.Add("filter[configured]", datadog.ParameterToString(*r.filterConfigured, ""))
	}
	if r.filterTagsConfigured != nil {
		localVarQueryParams.Add("filter[tags_configured]", datadog.ParameterToString(*r.filterTagsConfigured, ""))
	}
	if r.filterMetricType != nil {
		localVarQueryParams.Add("filter[metric_type]", datadog.ParameterToString(*r.filterMetricType, ""))
	}
	if r.filterIncludePercentiles != nil {
		localVarQueryParams.Add("filter[include_percentiles]", datadog.ParameterToString(*r.filterIncludePercentiles, ""))
	}
	if r.filterQueried != nil {
		localVarQueryParams.Add("filter[queried]", datadog.ParameterToString(*r.filterQueried, ""))
	}
	if r.filterTags != nil {
		localVarQueryParams.Add("filter[tags]", datadog.ParameterToString(*r.filterTags, ""))
	}
	if r.windowSeconds != nil {
		localVarQueryParams.Add("window[seconds]", datadog.ParameterToString(*r.windowSeconds, ""))
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

type apiListTagsByMetricNameRequest struct {
	ctx        _context.Context
	metricName string
}

func (a *MetricsApi) buildListTagsByMetricNameRequest(ctx _context.Context, metricName string) (apiListTagsByMetricNameRequest, error) {
	req := apiListTagsByMetricNameRequest{
		ctx:        ctx,
		metricName: metricName,
	}
	return req, nil
}

// ListTagsByMetricName List tags by metric name.
// View indexed tag key-value pairs for a given metric name.
func (a *MetricsApi) ListTagsByMetricName(ctx _context.Context, metricName string) (MetricAllTagsResponse, *_nethttp.Response, error) {
	req, err := a.buildListTagsByMetricNameRequest(ctx, metricName)
	if err != nil {
		var localVarReturnValue MetricAllTagsResponse
		return localVarReturnValue, nil, err
	}

	return a.listTagsByMetricNameExecute(req)
}

// listTagsByMetricNameExecute executes the request.
func (a *MetricsApi) listTagsByMetricNameExecute(r apiListTagsByMetricNameRequest) (MetricAllTagsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricAllTagsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.ListTagsByMetricName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/all-tags"
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

type apiListVolumesByMetricNameRequest struct {
	ctx        _context.Context
	metricName string
}

func (a *MetricsApi) buildListVolumesByMetricNameRequest(ctx _context.Context, metricName string) (apiListVolumesByMetricNameRequest, error) {
	req := apiListVolumesByMetricNameRequest{
		ctx:        ctx,
		metricName: metricName,
	}
	return req, nil
}

// ListVolumesByMetricName List distinct metric volumes by metric name.
// View distinct metrics volumes for the given metric name.
//
// Custom metrics generated in-app from other products will return `null` for ingested volumes.
func (a *MetricsApi) ListVolumesByMetricName(ctx _context.Context, metricName string) (MetricVolumesResponse, *_nethttp.Response, error) {
	req, err := a.buildListVolumesByMetricNameRequest(ctx, metricName)
	if err != nil {
		var localVarReturnValue MetricVolumesResponse
		return localVarReturnValue, nil, err
	}

	return a.listVolumesByMetricNameExecute(req)
}

// listVolumesByMetricNameExecute executes the request.
func (a *MetricsApi) listVolumesByMetricNameExecute(r apiListVolumesByMetricNameRequest) (MetricVolumesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricVolumesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.ListVolumesByMetricName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/volumes"
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

type apiQueryScalarDataRequest struct {
	ctx  _context.Context
	body *ScalarFormulaQueryRequest
}

func (a *MetricsApi) buildQueryScalarDataRequest(ctx _context.Context, body ScalarFormulaQueryRequest) (apiQueryScalarDataRequest, error) {
	req := apiQueryScalarDataRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// QueryScalarData Query scalar data across multiple products.
// Query scalar values (as seen on Query Value, Table and Toplist widgets).
// Multiple data sources are supported with the ability to
// process the data using formulas and functions.
func (a *MetricsApi) QueryScalarData(ctx _context.Context, body ScalarFormulaQueryRequest) (ScalarFormulaQueryResponse, *_nethttp.Response, error) {
	req, err := a.buildQueryScalarDataRequest(ctx, body)
	if err != nil {
		var localVarReturnValue ScalarFormulaQueryResponse
		return localVarReturnValue, nil, err
	}

	return a.queryScalarDataExecute(req)
}

// queryScalarDataExecute executes the request.
func (a *MetricsApi) queryScalarDataExecute(r apiQueryScalarDataRequest) (ScalarFormulaQueryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue ScalarFormulaQueryResponse
	)

	operationId := "v2.QueryScalarData"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.QueryScalarData")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/query/scalar"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
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

type apiQueryTimeseriesDataRequest struct {
	ctx  _context.Context
	body *TimeseriesFormulaQueryRequest
}

func (a *MetricsApi) buildQueryTimeseriesDataRequest(ctx _context.Context, body TimeseriesFormulaQueryRequest) (apiQueryTimeseriesDataRequest, error) {
	req := apiQueryTimeseriesDataRequest{
		ctx:  ctx,
		body: &body,
	}
	return req, nil
}

// QueryTimeseriesData Query timeseries data across multiple products.
// Query timeseries data across various data sources and
// process the data by applying formulas and functions.
func (a *MetricsApi) QueryTimeseriesData(ctx _context.Context, body TimeseriesFormulaQueryRequest) (TimeseriesFormulaQueryResponse, *_nethttp.Response, error) {
	req, err := a.buildQueryTimeseriesDataRequest(ctx, body)
	if err != nil {
		var localVarReturnValue TimeseriesFormulaQueryResponse
		return localVarReturnValue, nil, err
	}

	return a.queryTimeseriesDataExecute(req)
}

// queryTimeseriesDataExecute executes the request.
func (a *MetricsApi) queryTimeseriesDataExecute(r apiQueryTimeseriesDataRequest) (TimeseriesFormulaQueryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue TimeseriesFormulaQueryResponse
	)

	operationId := "v2.QueryTimeseriesData"
	if a.Client.Cfg.IsUnstableOperationEnabled(operationId) {
		_log.Printf("WARNING: Using unstable operation '%s'", operationId)
	} else {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: _fmt.Sprintf("Unstable operation '%s' is disabled", operationId)}
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.QueryTimeseriesData")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/query/timeseries"

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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 401 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 429 {
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
	body            *MetricPayload
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

func (a *MetricsApi) buildSubmitMetricsRequest(ctx _context.Context, body MetricPayload, o ...SubmitMetricsOptionalParameters) (apiSubmitMetricsRequest, error) {
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
// The maximum payload size is 500 kilobytes (512000 bytes). Compressed payloads must have a decompressed size of less than 5 megabytes (5242880 bytes).
//
// If you’re submitting metrics directly to the Datadog API without using DogStatsD, expect:
//
// - 64 bits for the timestamp
// - 64 bits for the value
// - 20 bytes for the metric names
// - 50 bytes for the timeseries
// - The full payload is approximately 100 bytes.
//
// Host name is one of the resources in the Resources field.
func (a *MetricsApi) SubmitMetrics(ctx _context.Context, body MetricPayload, o ...SubmitMetricsOptionalParameters) (IntakePayloadAccepted, *_nethttp.Response, error) {
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

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.SubmitMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/series"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.body == nil {
		return localVarReturnValue, nil, datadog.ReportError("body is required and must be specified")
	}
	localVarHeaderParams["Content-Type"] = "application/json"
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

type apiUpdateTagConfigurationRequest struct {
	ctx        _context.Context
	metricName string
	body       *MetricTagConfigurationUpdateRequest
}

func (a *MetricsApi) buildUpdateTagConfigurationRequest(ctx _context.Context, metricName string, body MetricTagConfigurationUpdateRequest) (apiUpdateTagConfigurationRequest, error) {
	req := apiUpdateTagConfigurationRequest{
		ctx:        ctx,
		metricName: metricName,
		body:       &body,
	}
	return req, nil
}

// UpdateTagConfiguration Update a tag configuration.
// Update the tag configuration of a metric or percentile aggregations of a distribution metric or custom aggregations
// of a count, rate, or gauge metric.
// Can only be used with application keys from users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) UpdateTagConfiguration(ctx _context.Context, metricName string, body MetricTagConfigurationUpdateRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	req, err := a.buildUpdateTagConfigurationRequest(ctx, metricName, body)
	if err != nil {
		var localVarReturnValue MetricTagConfigurationResponse
		return localVarReturnValue, nil, err
	}

	return a.updateTagConfigurationExecute(req)
}

// updateTagConfigurationExecute executes the request.
func (a *MetricsApi) updateTagConfigurationExecute(r apiUpdateTagConfigurationRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v2.MetricsApi.UpdateTagConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
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
		if localVarHTTPResponse.StatusCode == 400 || localVarHTTPResponse.StatusCode == 403 || localVarHTTPResponse.StatusCode == 422 || localVarHTTPResponse.StatusCode == 429 {
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
