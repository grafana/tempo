// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// UsageMeteringApi service type
type UsageMeteringApi datadog.Service

// GetCostByOrgOptionalParameters holds optional parameters for GetCostByOrg.
type GetCostByOrgOptionalParameters struct {
	EndMonth *time.Time
}

// NewGetCostByOrgOptionalParameters creates an empty struct for parameters.
func NewGetCostByOrgOptionalParameters() *GetCostByOrgOptionalParameters {
	this := GetCostByOrgOptionalParameters{}
	return &this
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetCostByOrgOptionalParameters) WithEndMonth(endMonth time.Time) *GetCostByOrgOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// GetCostByOrg Get cost across multi-org account.
// Get cost across multi-org account.
// Cost by org data for a given month becomes available no later than the 16th of the following month.
// **Note:** This endpoint has been deprecated. Please use the new endpoint
// [`/historical_cost`](https://docs.datadoghq.com/api/latest/usage-metering/#get-historical-cost-across-your-account)
// instead.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetCostByOrg(ctx _context.Context, startMonth time.Time, o ...GetCostByOrgOptionalParameters) (CostByOrgResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CostByOrgResponse
		optionalParams      GetCostByOrgOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetCostByOrgOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetCostByOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/cost_by_org"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_month", datadog.ParameterToString(startMonth, ""))
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetEstimatedCostByOrgOptionalParameters holds optional parameters for GetEstimatedCostByOrg.
type GetEstimatedCostByOrgOptionalParameters struct {
	View       *string
	StartMonth *time.Time
	EndMonth   *time.Time
	StartDate  *time.Time
	EndDate    *time.Time
}

// NewGetEstimatedCostByOrgOptionalParameters creates an empty struct for parameters.
func NewGetEstimatedCostByOrgOptionalParameters() *GetEstimatedCostByOrgOptionalParameters {
	this := GetEstimatedCostByOrgOptionalParameters{}
	return &this
}

// WithView sets the corresponding parameter name and returns the struct.
func (r *GetEstimatedCostByOrgOptionalParameters) WithView(view string) *GetEstimatedCostByOrgOptionalParameters {
	r.View = &view
	return r
}

// WithStartMonth sets the corresponding parameter name and returns the struct.
func (r *GetEstimatedCostByOrgOptionalParameters) WithStartMonth(startMonth time.Time) *GetEstimatedCostByOrgOptionalParameters {
	r.StartMonth = &startMonth
	return r
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetEstimatedCostByOrgOptionalParameters) WithEndMonth(endMonth time.Time) *GetEstimatedCostByOrgOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// WithStartDate sets the corresponding parameter name and returns the struct.
func (r *GetEstimatedCostByOrgOptionalParameters) WithStartDate(startDate time.Time) *GetEstimatedCostByOrgOptionalParameters {
	r.StartDate = &startDate
	return r
}

// WithEndDate sets the corresponding parameter name and returns the struct.
func (r *GetEstimatedCostByOrgOptionalParameters) WithEndDate(endDate time.Time) *GetEstimatedCostByOrgOptionalParameters {
	r.EndDate = &endDate
	return r
}

// GetEstimatedCostByOrg Get estimated cost across your account.
// Get estimated cost across multi-org and single root-org accounts.
// Estimated cost data is only available for the current month and previous month
// and is delayed by up to 72 hours from when it was incurred.
// To access historical costs prior to this, use the `/historical_cost` endpoint.
func (a *UsageMeteringApi) GetEstimatedCostByOrg(ctx _context.Context, o ...GetEstimatedCostByOrgOptionalParameters) (CostByOrgResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CostByOrgResponse
		optionalParams      GetEstimatedCostByOrgOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetEstimatedCostByOrgOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetEstimatedCostByOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/estimated_cost"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.View != nil {
		localVarQueryParams.Add("view", datadog.ParameterToString(*optionalParams.View, ""))
	}
	if optionalParams.StartMonth != nil {
		localVarQueryParams.Add("start_month", datadog.ParameterToString(*optionalParams.StartMonth, ""))
	}
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	if optionalParams.StartDate != nil {
		localVarQueryParams.Add("start_date", datadog.ParameterToString(*optionalParams.StartDate, ""))
	}
	if optionalParams.EndDate != nil {
		localVarQueryParams.Add("end_date", datadog.ParameterToString(*optionalParams.EndDate, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetHistoricalCostByOrgOptionalParameters holds optional parameters for GetHistoricalCostByOrg.
type GetHistoricalCostByOrgOptionalParameters struct {
	View     *string
	EndMonth *time.Time
}

// NewGetHistoricalCostByOrgOptionalParameters creates an empty struct for parameters.
func NewGetHistoricalCostByOrgOptionalParameters() *GetHistoricalCostByOrgOptionalParameters {
	this := GetHistoricalCostByOrgOptionalParameters{}
	return &this
}

// WithView sets the corresponding parameter name and returns the struct.
func (r *GetHistoricalCostByOrgOptionalParameters) WithView(view string) *GetHistoricalCostByOrgOptionalParameters {
	r.View = &view
	return r
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetHistoricalCostByOrgOptionalParameters) WithEndMonth(endMonth time.Time) *GetHistoricalCostByOrgOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// GetHistoricalCostByOrg Get historical cost across your account.
// Get historical cost across multi-org and single root-org accounts.
// Cost data for a given month becomes available no later than the 16th of the following month.
func (a *UsageMeteringApi) GetHistoricalCostByOrg(ctx _context.Context, startMonth time.Time, o ...GetHistoricalCostByOrgOptionalParameters) (CostByOrgResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue CostByOrgResponse
		optionalParams      GetHistoricalCostByOrgOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetHistoricalCostByOrgOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetHistoricalCostByOrg")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/historical_cost"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_month", datadog.ParameterToString(startMonth, ""))
	if optionalParams.View != nil {
		localVarQueryParams.Add("view", datadog.ParameterToString(*optionalParams.View, ""))
	}
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetHourlyUsageOptionalParameters holds optional parameters for GetHourlyUsage.
type GetHourlyUsageOptionalParameters struct {
	FilterTimestampEnd       *time.Time
	FilterIncludeDescendants *bool
	FilterVersions           *string
	PageLimit                *int32
	PageNextRecordId         *string
}

// NewGetHourlyUsageOptionalParameters creates an empty struct for parameters.
func NewGetHourlyUsageOptionalParameters() *GetHourlyUsageOptionalParameters {
	this := GetHourlyUsageOptionalParameters{}
	return &this
}

// WithFilterTimestampEnd sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageOptionalParameters) WithFilterTimestampEnd(filterTimestampEnd time.Time) *GetHourlyUsageOptionalParameters {
	r.FilterTimestampEnd = &filterTimestampEnd
	return r
}

// WithFilterIncludeDescendants sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageOptionalParameters) WithFilterIncludeDescendants(filterIncludeDescendants bool) *GetHourlyUsageOptionalParameters {
	r.FilterIncludeDescendants = &filterIncludeDescendants
	return r
}

// WithFilterVersions sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageOptionalParameters) WithFilterVersions(filterVersions string) *GetHourlyUsageOptionalParameters {
	r.FilterVersions = &filterVersions
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageOptionalParameters) WithPageLimit(pageLimit int32) *GetHourlyUsageOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

// WithPageNextRecordId sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageOptionalParameters) WithPageNextRecordId(pageNextRecordId string) *GetHourlyUsageOptionalParameters {
	r.PageNextRecordId = &pageNextRecordId
	return r
}

// GetHourlyUsage Get hourly usage by product family.
// Get hourly usage by product family.
func (a *UsageMeteringApi) GetHourlyUsage(ctx _context.Context, filterTimestampStart time.Time, filterProductFamilies string, o ...GetHourlyUsageOptionalParameters) (HourlyUsageResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HourlyUsageResponse
		optionalParams      GetHourlyUsageOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetHourlyUsageOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetHourlyUsage")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/hourly_usage"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("filter[timestamp][start]", datadog.ParameterToString(filterTimestampStart, ""))
	localVarQueryParams.Add("filter[product_families]", datadog.ParameterToString(filterProductFamilies, ""))
	if optionalParams.FilterTimestampEnd != nil {
		localVarQueryParams.Add("filter[timestamp][end]", datadog.ParameterToString(*optionalParams.FilterTimestampEnd, ""))
	}
	if optionalParams.FilterIncludeDescendants != nil {
		localVarQueryParams.Add("filter[include_descendants]", datadog.ParameterToString(*optionalParams.FilterIncludeDescendants, ""))
	}
	if optionalParams.FilterVersions != nil {
		localVarQueryParams.Add("filter[versions]", datadog.ParameterToString(*optionalParams.FilterVersions, ""))
	}
	if optionalParams.PageLimit != nil {
		localVarQueryParams.Add("page[limit]", datadog.ParameterToString(*optionalParams.PageLimit, ""))
	}
	if optionalParams.PageNextRecordId != nil {
		localVarQueryParams.Add("page[next_record_id]", datadog.ParameterToString(*optionalParams.PageNextRecordId, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetUsageApplicationSecurityMonitoringOptionalParameters holds optional parameters for GetUsageApplicationSecurityMonitoring.
type GetUsageApplicationSecurityMonitoringOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageApplicationSecurityMonitoringOptionalParameters creates an empty struct for parameters.
func NewGetUsageApplicationSecurityMonitoringOptionalParameters() *GetUsageApplicationSecurityMonitoringOptionalParameters {
	this := GetUsageApplicationSecurityMonitoringOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageApplicationSecurityMonitoringOptionalParameters) WithEndHr(endHr time.Time) *GetUsageApplicationSecurityMonitoringOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageApplicationSecurityMonitoring Get hourly usage for application security.
// Get hourly usage for application security .
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family)
func (a *UsageMeteringApi) GetUsageApplicationSecurityMonitoring(ctx _context.Context, startHr time.Time, o ...GetUsageApplicationSecurityMonitoringOptionalParameters) (UsageApplicationSecurityMonitoringResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageApplicationSecurityMonitoringResponse
		optionalParams      GetUsageApplicationSecurityMonitoringOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageApplicationSecurityMonitoringOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetUsageApplicationSecurityMonitoring")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/application_security"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetUsageLambdaTracedInvocationsOptionalParameters holds optional parameters for GetUsageLambdaTracedInvocations.
type GetUsageLambdaTracedInvocationsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageLambdaTracedInvocationsOptionalParameters creates an empty struct for parameters.
func NewGetUsageLambdaTracedInvocationsOptionalParameters() *GetUsageLambdaTracedInvocationsOptionalParameters {
	this := GetUsageLambdaTracedInvocationsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageLambdaTracedInvocationsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageLambdaTracedInvocationsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageLambdaTracedInvocations Get hourly usage for lambda traced invocations.
// Get hourly usage for lambda traced invocations.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family)
func (a *UsageMeteringApi) GetUsageLambdaTracedInvocations(ctx _context.Context, startHr time.Time, o ...GetUsageLambdaTracedInvocationsOptionalParameters) (UsageLambdaTracedInvocationsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageLambdaTracedInvocationsResponse
		optionalParams      GetUsageLambdaTracedInvocationsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageLambdaTracedInvocationsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetUsageLambdaTracedInvocations")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/lambda_traced_invocations"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// GetUsageObservabilityPipelinesOptionalParameters holds optional parameters for GetUsageObservabilityPipelines.
type GetUsageObservabilityPipelinesOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageObservabilityPipelinesOptionalParameters creates an empty struct for parameters.
func NewGetUsageObservabilityPipelinesOptionalParameters() *GetUsageObservabilityPipelinesOptionalParameters {
	this := GetUsageObservabilityPipelinesOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageObservabilityPipelinesOptionalParameters) WithEndHr(endHr time.Time) *GetUsageObservabilityPipelinesOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageObservabilityPipelines Get hourly usage for observability pipelines.
// Get hourly usage for observability pipelines.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family)
func (a *UsageMeteringApi) GetUsageObservabilityPipelines(ctx _context.Context, startHr time.Time, o ...GetUsageObservabilityPipelinesOptionalParameters) (UsageObservabilityPipelinesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageObservabilityPipelinesResponse
		optionalParams      GetUsageObservabilityPipelinesOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageObservabilityPipelinesOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.UsageMeteringApi.GetUsageObservabilityPipelines")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/usage/observability_pipelines"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	localVarHeaderParams["Accept"] = "application/json;datetime-format=rfc3339"

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

// NewUsageMeteringApi Returns NewUsageMeteringApi.
func NewUsageMeteringApi(client *datadog.APIClient) *UsageMeteringApi {
	return &UsageMeteringApi{
		Client: client,
	}
}
