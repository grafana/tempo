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

// CreateBulkTagsMetricsConfiguration Configure tags for multiple metrics.
// Create and define a list of queryable tag keys for a set of existing count, gauge, rate, and distribution metrics.
// Metrics are selected by passing a metric name prefix. Use the Delete method of this API path to remove tag configurations.
// Results can be sent to a set of account email addresses, just like the same operation in the Datadog web app.
// If multiple calls include the same metric, the last configuration applied (not by submit order) is used, do not
// expect deterministic ordering of concurrent calls.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) CreateBulkTagsMetricsConfiguration(ctx _context.Context, body MetricBulkTagConfigCreateRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue MetricBulkTagConfigResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.CreateBulkTagsMetricsConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/config/bulk-tags"

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

// CreateTagConfiguration Create a tag configuration.
// Create and define a list of queryable tag keys for an existing count/gauge/rate/distribution metric.
// Optionally, include percentile aggregations on any distribution metric or configure custom aggregations
// on any count, rate, or gauge metric.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) CreateTagConfiguration(ctx _context.Context, metricName string, body MetricTagConfigurationCreateRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.CreateTagConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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

// DeleteBulkTagsMetricsConfiguration Configure tags for multiple metrics.
// Delete all custom lists of queryable tag keys for a set of existing count, gauge, rate, and distribution metrics.
// Metrics are selected by passing a metric name prefix.
// Results can be sent to a set of account email addresses, just like the same operation in the Datadog web app.
// Can only be used with application keys of users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) DeleteBulkTagsMetricsConfiguration(ctx _context.Context, body MetricBulkTagConfigDeleteRequest) (MetricBulkTagConfigResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodDelete
		localVarPostBody    interface{}
		localVarReturnValue MetricBulkTagConfigResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.DeleteBulkTagsMetricsConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/config/bulk-tags"

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

// DeleteTagConfiguration Delete a tag configuration.
// Deletes a metric's tag configuration. Can only be used with application
// keys from users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) DeleteTagConfiguration(ctx _context.Context, metricName string) (*_nethttp.Response, error) {
	var (
		localVarHTTPMethod = _nethttp.MethodDelete
		localVarPostBody   interface{}
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.DeleteTagConfiguration")
	if err != nil {
		return nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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

// EstimateMetricsOutputSeries Tag Configuration Cardinality Estimator.
// Returns the estimated cardinality for a metric with a given tag, percentile and number of aggregations configuration using Metrics without Limits&trade;.
func (a *MetricsApi) EstimateMetricsOutputSeries(ctx _context.Context, metricName string, o ...EstimateMetricsOutputSeriesOptionalParameters) (MetricEstimateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricEstimateResponse
		optionalParams      EstimateMetricsOutputSeriesOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type EstimateMetricsOutputSeriesOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.EstimateMetricsOutputSeries")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/estimate"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.FilterGroups != nil {
		localVarQueryParams.Add("filter[groups]", datadog.ParameterToString(*optionalParams.FilterGroups, ""))
	}
	if optionalParams.FilterHoursAgo != nil {
		localVarQueryParams.Add("filter[hours_ago]", datadog.ParameterToString(*optionalParams.FilterHoursAgo, ""))
	}
	if optionalParams.FilterNumAggregations != nil {
		localVarQueryParams.Add("filter[num_aggregations]", datadog.ParameterToString(*optionalParams.FilterNumAggregations, ""))
	}
	if optionalParams.FilterPct != nil {
		localVarQueryParams.Add("filter[pct]", datadog.ParameterToString(*optionalParams.FilterPct, ""))
	}
	if optionalParams.FilterTimespanH != nil {
		localVarQueryParams.Add("filter[timespan_h]", datadog.ParameterToString(*optionalParams.FilterTimespanH, ""))
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

// ListActiveMetricConfigurations List active tags and aggregations.
// List tags and aggregations that are actively queried on dashboards and monitors for a given metric name.
func (a *MetricsApi) ListActiveMetricConfigurations(ctx _context.Context, metricName string, o ...ListActiveMetricConfigurationsOptionalParameters) (MetricSuggestedTagsAndAggregationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricSuggestedTagsAndAggregationsResponse
		optionalParams      ListActiveMetricConfigurationsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListActiveMetricConfigurationsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.ListActiveMetricConfigurations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/active-configurations"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.WindowSeconds != nil {
		localVarQueryParams.Add("window[seconds]", datadog.ParameterToString(*optionalParams.WindowSeconds, ""))
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

// ListTagConfigurationByName List tag configuration by name.
// Returns the tag configuration for the given metric name.
func (a *MetricsApi) ListTagConfigurationByName(ctx _context.Context, metricName string) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.ListTagConfigurationByName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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

// ListTagConfigurations Get a list of metrics.
// Returns all metrics that can be configured in the Metrics Summary page or with Metrics without Limits™ (matching additional filters if specified).
func (a *MetricsApi) ListTagConfigurations(ctx _context.Context, o ...ListTagConfigurationsOptionalParameters) (MetricsAndMetricTagConfigurationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricsAndMetricTagConfigurationsResponse
		optionalParams      ListTagConfigurationsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListTagConfigurationsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.ListTagConfigurations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.FilterConfigured != nil {
		localVarQueryParams.Add("filter[configured]", datadog.ParameterToString(*optionalParams.FilterConfigured, ""))
	}
	if optionalParams.FilterTagsConfigured != nil {
		localVarQueryParams.Add("filter[tags_configured]", datadog.ParameterToString(*optionalParams.FilterTagsConfigured, ""))
	}
	if optionalParams.FilterMetricType != nil {
		localVarQueryParams.Add("filter[metric_type]", datadog.ParameterToString(*optionalParams.FilterMetricType, ""))
	}
	if optionalParams.FilterIncludePercentiles != nil {
		localVarQueryParams.Add("filter[include_percentiles]", datadog.ParameterToString(*optionalParams.FilterIncludePercentiles, ""))
	}
	if optionalParams.FilterQueried != nil {
		localVarQueryParams.Add("filter[queried]", datadog.ParameterToString(*optionalParams.FilterQueried, ""))
	}
	if optionalParams.FilterTags != nil {
		localVarQueryParams.Add("filter[tags]", datadog.ParameterToString(*optionalParams.FilterTags, ""))
	}
	if optionalParams.WindowSeconds != nil {
		localVarQueryParams.Add("window[seconds]", datadog.ParameterToString(*optionalParams.WindowSeconds, ""))
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

// ListTagsByMetricName List tags by metric name.
// View indexed tag key-value pairs for a given metric name.
func (a *MetricsApi) ListTagsByMetricName(ctx _context.Context, metricName string) (MetricAllTagsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricAllTagsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.ListTagsByMetricName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/all-tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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

// ListVolumesByMetricName List distinct metric volumes by metric name.
// View distinct metrics volumes for the given metric name.
//
// Custom metrics generated in-app from other products will return `null` for ingested volumes.
func (a *MetricsApi) ListVolumesByMetricName(ctx _context.Context, metricName string) (MetricVolumesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MetricVolumesResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.ListVolumesByMetricName")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/volumes"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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

// QueryScalarData Query scalar data across multiple products.
// Query scalar values (as seen on Query Value, Table and Toplist widgets).
// Multiple data sources are supported with the ability to
// process the data using formulas and functions.
func (a *MetricsApi) QueryScalarData(ctx _context.Context, body ScalarFormulaQueryRequest) (ScalarFormulaQueryResponse, *_nethttp.Response, error) {
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

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.QueryScalarData")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/query/scalar"

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

// QueryTimeseriesData Query timeseries data across multiple products.
// Query timeseries data across various data sources and
// process the data by applying formulas and functions.
func (a *MetricsApi) QueryTimeseriesData(ctx _context.Context, body TimeseriesFormulaQueryRequest) (TimeseriesFormulaQueryResponse, *_nethttp.Response, error) {
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

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.QueryTimeseriesData")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/query/timeseries"

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
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue IntakePayloadAccepted
		optionalParams      SubmitMetricsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type SubmitMetricsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.SubmitMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/series"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarHeaderParams["Content-Type"] = "application/json"
	localVarHeaderParams["Accept"] = "application/json"

	if optionalParams.ContentEncoding != nil {
		localVarHeaderParams["Content-Encoding"] = datadog.ParameterToString(*optionalParams.ContentEncoding, "")
	}

	// body params
	localVarPostBody = &body
	datadog.SetAuthKeys(
		ctx,
		&localVarHeaderParams,
		[2]string{"apiKeyAuth", "DD-API-KEY"},
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

// UpdateTagConfiguration Update a tag configuration.
// Update the tag configuration of a metric or percentile aggregations of a distribution metric or custom aggregations
// of a count, rate, or gauge metric.
// Can only be used with application keys from users with the `Manage Tags for Metrics` permission.
func (a *MetricsApi) UpdateTagConfiguration(ctx _context.Context, metricName string, body MetricTagConfigurationUpdateRequest) (MetricTagConfigurationResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPatch
		localVarPostBody    interface{}
		localVarReturnValue MetricTagConfigurationResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.MetricsApi.UpdateTagConfiguration")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/metrics/{metric_name}/tags"
	localVarPath = strings.Replace(localVarPath, "{"+"metric_name"+"}", _neturl.PathEscape(datadog.ParameterToString(metricName, "")), -1)

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
