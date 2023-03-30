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
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// UsageMeteringApi service type
type UsageMeteringApi datadog.Service

// GetDailyCustomReportsOptionalParameters holds optional parameters for GetDailyCustomReports.
type GetDailyCustomReportsOptionalParameters struct {
	PageSize   *int64
	PageNumber *int64
	SortDir    *UsageSortDirection
	Sort       *UsageSort
}

// NewGetDailyCustomReportsOptionalParameters creates an empty struct for parameters.
func NewGetDailyCustomReportsOptionalParameters() *GetDailyCustomReportsOptionalParameters {
	this := GetDailyCustomReportsOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *GetDailyCustomReportsOptionalParameters) WithPageSize(pageSize int64) *GetDailyCustomReportsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *GetDailyCustomReportsOptionalParameters) WithPageNumber(pageNumber int64) *GetDailyCustomReportsOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *GetDailyCustomReportsOptionalParameters) WithSortDir(sortDir UsageSortDirection) *GetDailyCustomReportsOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *GetDailyCustomReportsOptionalParameters) WithSort(sort UsageSort) *GetDailyCustomReportsOptionalParameters {
	r.Sort = &sort
	return r
}

// GetDailyCustomReports Get the list of available daily custom reports.
// Get daily custom reports.
// **Note:** This endpoint will be fully deprecated on December 1, 2022.
// Refer to [Migrating from v1 to v2 of the Usage Attribution API](https://docs.datadoghq.com/account_management/guide/usage-attribution-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetDailyCustomReports(ctx _context.Context, o ...GetDailyCustomReportsOptionalParameters) (UsageCustomReportsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageCustomReportsResponse
		optionalParams      GetDailyCustomReportsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetDailyCustomReportsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetDailyCustomReports")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/daily_custom_reports"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.PageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*optionalParams.PageSize, ""))
	}
	if optionalParams.PageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*optionalParams.PageNumber, ""))
	}
	if optionalParams.SortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*optionalParams.SortDir, ""))
	}
	if optionalParams.Sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*optionalParams.Sort, ""))
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

// GetHourlyUsageAttributionOptionalParameters holds optional parameters for GetHourlyUsageAttribution.
type GetHourlyUsageAttributionOptionalParameters struct {
	EndHr              *time.Time
	NextRecordId       *string
	TagBreakdownKeys   *string
	IncludeDescendants *bool
}

// NewGetHourlyUsageAttributionOptionalParameters creates an empty struct for parameters.
func NewGetHourlyUsageAttributionOptionalParameters() *GetHourlyUsageAttributionOptionalParameters {
	this := GetHourlyUsageAttributionOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageAttributionOptionalParameters) WithEndHr(endHr time.Time) *GetHourlyUsageAttributionOptionalParameters {
	r.EndHr = &endHr
	return r
}

// WithNextRecordId sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageAttributionOptionalParameters) WithNextRecordId(nextRecordId string) *GetHourlyUsageAttributionOptionalParameters {
	r.NextRecordId = &nextRecordId
	return r
}

// WithTagBreakdownKeys sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageAttributionOptionalParameters) WithTagBreakdownKeys(tagBreakdownKeys string) *GetHourlyUsageAttributionOptionalParameters {
	r.TagBreakdownKeys = &tagBreakdownKeys
	return r
}

// WithIncludeDescendants sets the corresponding parameter name and returns the struct.
func (r *GetHourlyUsageAttributionOptionalParameters) WithIncludeDescendants(includeDescendants bool) *GetHourlyUsageAttributionOptionalParameters {
	r.IncludeDescendants = &includeDescendants
	return r
}

// GetHourlyUsageAttribution Get hourly usage attribution.
// Get hourly usage attribution.
//
// This API endpoint is paginated. To make sure you receive all records, check if the value of `next_record_id` is
// set in the response. If it is, make another request and pass `next_record_id` as a parameter.
// Pseudo code example:
//
// ```
// response := GetHourlyUsageAttribution(start_month)
// cursor := response.metadata.pagination.next_record_id
// WHILE cursor != null BEGIN
//   sleep(5 seconds)  # Avoid running into rate limit
//   response := GetHourlyUsageAttribution(start_month, next_record_id=cursor)
//   cursor := response.metadata.pagination.next_record_id
// END
// ```
func (a *UsageMeteringApi) GetHourlyUsageAttribution(ctx _context.Context, startHr time.Time, usageType HourlyUsageAttributionUsageType, o ...GetHourlyUsageAttributionOptionalParameters) (HourlyUsageAttributionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HourlyUsageAttributionResponse
		optionalParams      GetHourlyUsageAttributionOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetHourlyUsageAttributionOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetHourlyUsageAttribution")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/hourly-attribution"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	localVarQueryParams.Add("usage_type", datadog.ParameterToString(usageType, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	if optionalParams.NextRecordId != nil {
		localVarQueryParams.Add("next_record_id", datadog.ParameterToString(*optionalParams.NextRecordId, ""))
	}
	if optionalParams.TagBreakdownKeys != nil {
		localVarQueryParams.Add("tag_breakdown_keys", datadog.ParameterToString(*optionalParams.TagBreakdownKeys, ""))
	}
	if optionalParams.IncludeDescendants != nil {
		localVarQueryParams.Add("include_descendants", datadog.ParameterToString(*optionalParams.IncludeDescendants, ""))
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

// GetIncidentManagementOptionalParameters holds optional parameters for GetIncidentManagement.
type GetIncidentManagementOptionalParameters struct {
	EndHr *time.Time
}

// NewGetIncidentManagementOptionalParameters creates an empty struct for parameters.
func NewGetIncidentManagementOptionalParameters() *GetIncidentManagementOptionalParameters {
	this := GetIncidentManagementOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetIncidentManagementOptionalParameters) WithEndHr(endHr time.Time) *GetIncidentManagementOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetIncidentManagement Get hourly usage for incident management.
// Get hourly usage for incident management.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetIncidentManagement(ctx _context.Context, startHr time.Time, o ...GetIncidentManagementOptionalParameters) (UsageIncidentManagementResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageIncidentManagementResponse
		optionalParams      GetIncidentManagementOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetIncidentManagementOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetIncidentManagement")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/incident-management"

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

// GetIngestedSpansOptionalParameters holds optional parameters for GetIngestedSpans.
type GetIngestedSpansOptionalParameters struct {
	EndHr *time.Time
}

// NewGetIngestedSpansOptionalParameters creates an empty struct for parameters.
func NewGetIngestedSpansOptionalParameters() *GetIngestedSpansOptionalParameters {
	this := GetIngestedSpansOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetIngestedSpansOptionalParameters) WithEndHr(endHr time.Time) *GetIngestedSpansOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetIngestedSpans Get hourly usage for ingested spans.
// Get hourly usage for ingested spans.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetIngestedSpans(ctx _context.Context, startHr time.Time, o ...GetIngestedSpansOptionalParameters) (UsageIngestedSpansResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageIngestedSpansResponse
		optionalParams      GetIngestedSpansOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetIngestedSpansOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetIngestedSpans")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/ingested-spans"

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

// GetMonthlyCustomReportsOptionalParameters holds optional parameters for GetMonthlyCustomReports.
type GetMonthlyCustomReportsOptionalParameters struct {
	PageSize   *int64
	PageNumber *int64
	SortDir    *UsageSortDirection
	Sort       *UsageSort
}

// NewGetMonthlyCustomReportsOptionalParameters creates an empty struct for parameters.
func NewGetMonthlyCustomReportsOptionalParameters() *GetMonthlyCustomReportsOptionalParameters {
	this := GetMonthlyCustomReportsOptionalParameters{}
	return &this
}

// WithPageSize sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyCustomReportsOptionalParameters) WithPageSize(pageSize int64) *GetMonthlyCustomReportsOptionalParameters {
	r.PageSize = &pageSize
	return r
}

// WithPageNumber sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyCustomReportsOptionalParameters) WithPageNumber(pageNumber int64) *GetMonthlyCustomReportsOptionalParameters {
	r.PageNumber = &pageNumber
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyCustomReportsOptionalParameters) WithSortDir(sortDir UsageSortDirection) *GetMonthlyCustomReportsOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithSort sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyCustomReportsOptionalParameters) WithSort(sort UsageSort) *GetMonthlyCustomReportsOptionalParameters {
	r.Sort = &sort
	return r
}

// GetMonthlyCustomReports Get the list of available monthly custom reports.
// Get monthly custom reports.
// **Note:** This endpoint will be fully deprecated on December 1, 2022.
// Refer to [Migrating from v1 to v2 of the Usage Attribution API](https://docs.datadoghq.com/account_management/guide/usage-attribution-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetMonthlyCustomReports(ctx _context.Context, o ...GetMonthlyCustomReportsOptionalParameters) (UsageCustomReportsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageCustomReportsResponse
		optionalParams      GetMonthlyCustomReportsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetMonthlyCustomReportsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetMonthlyCustomReports")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/monthly_custom_reports"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.PageSize != nil {
		localVarQueryParams.Add("page[size]", datadog.ParameterToString(*optionalParams.PageSize, ""))
	}
	if optionalParams.PageNumber != nil {
		localVarQueryParams.Add("page[number]", datadog.ParameterToString(*optionalParams.PageNumber, ""))
	}
	if optionalParams.SortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*optionalParams.SortDir, ""))
	}
	if optionalParams.Sort != nil {
		localVarQueryParams.Add("sort", datadog.ParameterToString(*optionalParams.Sort, ""))
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

// GetMonthlyUsageAttributionOptionalParameters holds optional parameters for GetMonthlyUsageAttribution.
type GetMonthlyUsageAttributionOptionalParameters struct {
	EndMonth           *time.Time
	SortDirection      *UsageSortDirection
	SortName           *MonthlyUsageAttributionSupportedMetrics
	TagBreakdownKeys   *string
	NextRecordId       *string
	IncludeDescendants *bool
}

// NewGetMonthlyUsageAttributionOptionalParameters creates an empty struct for parameters.
func NewGetMonthlyUsageAttributionOptionalParameters() *GetMonthlyUsageAttributionOptionalParameters {
	this := GetMonthlyUsageAttributionOptionalParameters{}
	return &this
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithEndMonth(endMonth time.Time) *GetMonthlyUsageAttributionOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// WithSortDirection sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithSortDirection(sortDirection UsageSortDirection) *GetMonthlyUsageAttributionOptionalParameters {
	r.SortDirection = &sortDirection
	return r
}

// WithSortName sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithSortName(sortName MonthlyUsageAttributionSupportedMetrics) *GetMonthlyUsageAttributionOptionalParameters {
	r.SortName = &sortName
	return r
}

// WithTagBreakdownKeys sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithTagBreakdownKeys(tagBreakdownKeys string) *GetMonthlyUsageAttributionOptionalParameters {
	r.TagBreakdownKeys = &tagBreakdownKeys
	return r
}

// WithNextRecordId sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithNextRecordId(nextRecordId string) *GetMonthlyUsageAttributionOptionalParameters {
	r.NextRecordId = &nextRecordId
	return r
}

// WithIncludeDescendants sets the corresponding parameter name and returns the struct.
func (r *GetMonthlyUsageAttributionOptionalParameters) WithIncludeDescendants(includeDescendants bool) *GetMonthlyUsageAttributionOptionalParameters {
	r.IncludeDescendants = &includeDescendants
	return r
}

// GetMonthlyUsageAttribution Get monthly usage attribution.
// Get monthly usage attribution.
//
// This API endpoint is paginated. To make sure you receive all records, check if the value of `next_record_id` is
// set in the response. If it is, make another request and pass `next_record_id` as a parameter.
// Pseudo code example:
//
// ```
// response := GetMonthlyUsageAttribution(start_month)
// cursor := response.metadata.pagination.next_record_id
// WHILE cursor != null BEGIN
//   sleep(5 seconds)  # Avoid running into rate limit
//   response := GetMonthlyUsageAttribution(start_month, next_record_id=cursor)
//   cursor := response.metadata.pagination.next_record_id
// END
// ```
func (a *UsageMeteringApi) GetMonthlyUsageAttribution(ctx _context.Context, startMonth time.Time, fields MonthlyUsageAttributionSupportedMetrics, o ...GetMonthlyUsageAttributionOptionalParameters) (MonthlyUsageAttributionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue MonthlyUsageAttributionResponse
		optionalParams      GetMonthlyUsageAttributionOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetMonthlyUsageAttributionOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetMonthlyUsageAttribution")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/monthly-attribution"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_month", datadog.ParameterToString(startMonth, ""))
	localVarQueryParams.Add("fields", datadog.ParameterToString(fields, ""))
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	if optionalParams.SortDirection != nil {
		localVarQueryParams.Add("sort_direction", datadog.ParameterToString(*optionalParams.SortDirection, ""))
	}
	if optionalParams.SortName != nil {
		localVarQueryParams.Add("sort_name", datadog.ParameterToString(*optionalParams.SortName, ""))
	}
	if optionalParams.TagBreakdownKeys != nil {
		localVarQueryParams.Add("tag_breakdown_keys", datadog.ParameterToString(*optionalParams.TagBreakdownKeys, ""))
	}
	if optionalParams.NextRecordId != nil {
		localVarQueryParams.Add("next_record_id", datadog.ParameterToString(*optionalParams.NextRecordId, ""))
	}
	if optionalParams.IncludeDescendants != nil {
		localVarQueryParams.Add("include_descendants", datadog.ParameterToString(*optionalParams.IncludeDescendants, ""))
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

// GetSpecifiedDailyCustomReports Get specified daily custom reports.
// Get specified daily custom reports.
// **Note:** This endpoint will be fully deprecated on December 1, 2022.
// Refer to [Migrating from v1 to v2 of the Usage Attribution API](https://docs.datadoghq.com/account_management/guide/usage-attribution-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetSpecifiedDailyCustomReports(ctx _context.Context, reportId string) (UsageSpecifiedCustomReportsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSpecifiedCustomReportsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetSpecifiedDailyCustomReports")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/daily_custom_reports/{report_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"report_id"+"}", _neturl.PathEscape(datadog.ParameterToString(reportId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
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

// GetSpecifiedMonthlyCustomReports Get specified monthly custom reports.
// Get specified monthly custom reports.
// **Note:** This endpoint will be fully deprecated on December 1, 2022.
// Refer to [Migrating from v1 to v2 of the Usage Attribution API](https://docs.datadoghq.com/account_management/guide/usage-attribution-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetSpecifiedMonthlyCustomReports(ctx _context.Context, reportId string) (UsageSpecifiedCustomReportsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSpecifiedCustomReportsResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetSpecifiedMonthlyCustomReports")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/monthly_custom_reports/{report_id}"
	localVarPath = strings.Replace(localVarPath, "{"+"report_id"+"}", _neturl.PathEscape(datadog.ParameterToString(reportId, "")), -1)

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
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

// GetUsageAnalyzedLogsOptionalParameters holds optional parameters for GetUsageAnalyzedLogs.
type GetUsageAnalyzedLogsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageAnalyzedLogsOptionalParameters creates an empty struct for parameters.
func NewGetUsageAnalyzedLogsOptionalParameters() *GetUsageAnalyzedLogsOptionalParameters {
	this := GetUsageAnalyzedLogsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageAnalyzedLogsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageAnalyzedLogsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageAnalyzedLogs Get hourly usage for analyzed logs.
// Get hourly usage for analyzed logs (Security Monitoring).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageAnalyzedLogs(ctx _context.Context, startHr time.Time, o ...GetUsageAnalyzedLogsOptionalParameters) (UsageAnalyzedLogsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageAnalyzedLogsResponse
		optionalParams      GetUsageAnalyzedLogsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageAnalyzedLogsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageAnalyzedLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/analyzed_logs"

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

// GetUsageAttributionOptionalParameters holds optional parameters for GetUsageAttribution.
type GetUsageAttributionOptionalParameters struct {
	EndMonth           *time.Time
	SortDirection      *UsageSortDirection
	SortName           *UsageAttributionSort
	IncludeDescendants *bool
	Offset             *int64
	Limit              *int64
}

// NewGetUsageAttributionOptionalParameters creates an empty struct for parameters.
func NewGetUsageAttributionOptionalParameters() *GetUsageAttributionOptionalParameters {
	this := GetUsageAttributionOptionalParameters{}
	return &this
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithEndMonth(endMonth time.Time) *GetUsageAttributionOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// WithSortDirection sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithSortDirection(sortDirection UsageSortDirection) *GetUsageAttributionOptionalParameters {
	r.SortDirection = &sortDirection
	return r
}

// WithSortName sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithSortName(sortName UsageAttributionSort) *GetUsageAttributionOptionalParameters {
	r.SortName = &sortName
	return r
}

// WithIncludeDescendants sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithIncludeDescendants(includeDescendants bool) *GetUsageAttributionOptionalParameters {
	r.IncludeDescendants = &includeDescendants
	return r
}

// WithOffset sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithOffset(offset int64) *GetUsageAttributionOptionalParameters {
	r.Offset = &offset
	return r
}

// WithLimit sets the corresponding parameter name and returns the struct.
func (r *GetUsageAttributionOptionalParameters) WithLimit(limit int64) *GetUsageAttributionOptionalParameters {
	r.Limit = &limit
	return r
}

// GetUsageAttribution Get usage attribution.
// Get usage attribution.
// **Note:** This endpoint will be fully deprecated on December 1, 2022.
// Refer to [Migrating from v1 to v2 of the Usage Attribution API](https://docs.datadoghq.com/account_management/guide/usage-attribution-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetUsageAttribution(ctx _context.Context, startMonth time.Time, fields UsageAttributionSupportedMetrics, o ...GetUsageAttributionOptionalParameters) (UsageAttributionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageAttributionResponse
		optionalParams      GetUsageAttributionOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageAttributionOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageAttribution")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/attribution"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_month", datadog.ParameterToString(startMonth, ""))
	localVarQueryParams.Add("fields", datadog.ParameterToString(fields, ""))
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	if optionalParams.SortDirection != nil {
		localVarQueryParams.Add("sort_direction", datadog.ParameterToString(*optionalParams.SortDirection, ""))
	}
	if optionalParams.SortName != nil {
		localVarQueryParams.Add("sort_name", datadog.ParameterToString(*optionalParams.SortName, ""))
	}
	if optionalParams.IncludeDescendants != nil {
		localVarQueryParams.Add("include_descendants", datadog.ParameterToString(*optionalParams.IncludeDescendants, ""))
	}
	if optionalParams.Offset != nil {
		localVarQueryParams.Add("offset", datadog.ParameterToString(*optionalParams.Offset, ""))
	}
	if optionalParams.Limit != nil {
		localVarQueryParams.Add("limit", datadog.ParameterToString(*optionalParams.Limit, ""))
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

// GetUsageAuditLogsOptionalParameters holds optional parameters for GetUsageAuditLogs.
type GetUsageAuditLogsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageAuditLogsOptionalParameters creates an empty struct for parameters.
func NewGetUsageAuditLogsOptionalParameters() *GetUsageAuditLogsOptionalParameters {
	this := GetUsageAuditLogsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageAuditLogsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageAuditLogsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageAuditLogs Get hourly usage for audit logs.
// Get hourly usage for audit logs.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageAuditLogs(ctx _context.Context, startHr time.Time, o ...GetUsageAuditLogsOptionalParameters) (UsageAuditLogsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageAuditLogsResponse
		optionalParams      GetUsageAuditLogsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageAuditLogsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageAuditLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/audit_logs"

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

// GetUsageBillableSummaryOptionalParameters holds optional parameters for GetUsageBillableSummary.
type GetUsageBillableSummaryOptionalParameters struct {
	Month *time.Time
}

// NewGetUsageBillableSummaryOptionalParameters creates an empty struct for parameters.
func NewGetUsageBillableSummaryOptionalParameters() *GetUsageBillableSummaryOptionalParameters {
	this := GetUsageBillableSummaryOptionalParameters{}
	return &this
}

// WithMonth sets the corresponding parameter name and returns the struct.
func (r *GetUsageBillableSummaryOptionalParameters) WithMonth(month time.Time) *GetUsageBillableSummaryOptionalParameters {
	r.Month = &month
	return r
}

// GetUsageBillableSummary Get billable usage across your account.
// Get billable usage across your account.
func (a *UsageMeteringApi) GetUsageBillableSummary(ctx _context.Context, o ...GetUsageBillableSummaryOptionalParameters) (UsageBillableSummaryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageBillableSummaryResponse
		optionalParams      GetUsageBillableSummaryOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageBillableSummaryOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageBillableSummary")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/billable-summary"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Month != nil {
		localVarQueryParams.Add("month", datadog.ParameterToString(*optionalParams.Month, ""))
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

// GetUsageCIAppOptionalParameters holds optional parameters for GetUsageCIApp.
type GetUsageCIAppOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageCIAppOptionalParameters creates an empty struct for parameters.
func NewGetUsageCIAppOptionalParameters() *GetUsageCIAppOptionalParameters {
	this := GetUsageCIAppOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageCIAppOptionalParameters) WithEndHr(endHr time.Time) *GetUsageCIAppOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageCIApp Get hourly usage for CI visibility.
// Get hourly usage for CI visibility (tests, pipeline, and spans).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageCIApp(ctx _context.Context, startHr time.Time, o ...GetUsageCIAppOptionalParameters) (UsageCIVisibilityResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageCIVisibilityResponse
		optionalParams      GetUsageCIAppOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageCIAppOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageCIApp")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/ci-app"

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

// GetUsageCWSOptionalParameters holds optional parameters for GetUsageCWS.
type GetUsageCWSOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageCWSOptionalParameters creates an empty struct for parameters.
func NewGetUsageCWSOptionalParameters() *GetUsageCWSOptionalParameters {
	this := GetUsageCWSOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageCWSOptionalParameters) WithEndHr(endHr time.Time) *GetUsageCWSOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageCWS Get hourly usage for cloud workload security.
// Get hourly usage for cloud workload security.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageCWS(ctx _context.Context, startHr time.Time, o ...GetUsageCWSOptionalParameters) (UsageCWSResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageCWSResponse
		optionalParams      GetUsageCWSOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageCWSOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageCWS")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/cws"

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

// GetUsageCloudSecurityPostureManagementOptionalParameters holds optional parameters for GetUsageCloudSecurityPostureManagement.
type GetUsageCloudSecurityPostureManagementOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageCloudSecurityPostureManagementOptionalParameters creates an empty struct for parameters.
func NewGetUsageCloudSecurityPostureManagementOptionalParameters() *GetUsageCloudSecurityPostureManagementOptionalParameters {
	this := GetUsageCloudSecurityPostureManagementOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageCloudSecurityPostureManagementOptionalParameters) WithEndHr(endHr time.Time) *GetUsageCloudSecurityPostureManagementOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageCloudSecurityPostureManagement Get hourly usage for CSPM.
// Get hourly usage for cloud security posture management (CSPM).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageCloudSecurityPostureManagement(ctx _context.Context, startHr time.Time, o ...GetUsageCloudSecurityPostureManagementOptionalParameters) (UsageCloudSecurityPostureManagementResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageCloudSecurityPostureManagementResponse
		optionalParams      GetUsageCloudSecurityPostureManagementOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageCloudSecurityPostureManagementOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageCloudSecurityPostureManagement")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/cspm"

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

// GetUsageDBMOptionalParameters holds optional parameters for GetUsageDBM.
type GetUsageDBMOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageDBMOptionalParameters creates an empty struct for parameters.
func NewGetUsageDBMOptionalParameters() *GetUsageDBMOptionalParameters {
	this := GetUsageDBMOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageDBMOptionalParameters) WithEndHr(endHr time.Time) *GetUsageDBMOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageDBM Get hourly usage for database monitoring.
// Get hourly usage for database monitoring
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageDBM(ctx _context.Context, startHr time.Time, o ...GetUsageDBMOptionalParameters) (UsageDBMResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageDBMResponse
		optionalParams      GetUsageDBMOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageDBMOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageDBM")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/dbm"

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

// GetUsageFargateOptionalParameters holds optional parameters for GetUsageFargate.
type GetUsageFargateOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageFargateOptionalParameters creates an empty struct for parameters.
func NewGetUsageFargateOptionalParameters() *GetUsageFargateOptionalParameters {
	this := GetUsageFargateOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageFargateOptionalParameters) WithEndHr(endHr time.Time) *GetUsageFargateOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageFargate Get hourly usage for Fargate.
// Get hourly usage for [Fargate](https://docs.datadoghq.com/integrations/ecs_fargate/).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageFargate(ctx _context.Context, startHr time.Time, o ...GetUsageFargateOptionalParameters) (UsageFargateResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageFargateResponse
		optionalParams      GetUsageFargateOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageFargateOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageFargate")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/fargate"

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

// GetUsageHostsOptionalParameters holds optional parameters for GetUsageHosts.
type GetUsageHostsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageHostsOptionalParameters creates an empty struct for parameters.
func NewGetUsageHostsOptionalParameters() *GetUsageHostsOptionalParameters {
	this := GetUsageHostsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageHostsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageHostsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageHosts Get hourly usage for hosts and containers.
// Get hourly usage for hosts and containers.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageHosts(ctx _context.Context, startHr time.Time, o ...GetUsageHostsOptionalParameters) (UsageHostsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageHostsResponse
		optionalParams      GetUsageHostsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageHostsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageHosts")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/hosts"

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

// GetUsageIndexedSpansOptionalParameters holds optional parameters for GetUsageIndexedSpans.
type GetUsageIndexedSpansOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageIndexedSpansOptionalParameters creates an empty struct for parameters.
func NewGetUsageIndexedSpansOptionalParameters() *GetUsageIndexedSpansOptionalParameters {
	this := GetUsageIndexedSpansOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageIndexedSpansOptionalParameters) WithEndHr(endHr time.Time) *GetUsageIndexedSpansOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageIndexedSpans Get hourly usage for indexed spans.
// Get hourly usage for indexed spans.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageIndexedSpans(ctx _context.Context, startHr time.Time, o ...GetUsageIndexedSpansOptionalParameters) (UsageIndexedSpansResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageIndexedSpansResponse
		optionalParams      GetUsageIndexedSpansOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageIndexedSpansOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageIndexedSpans")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/indexed-spans"

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

// GetUsageInternetOfThingsOptionalParameters holds optional parameters for GetUsageInternetOfThings.
type GetUsageInternetOfThingsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageInternetOfThingsOptionalParameters creates an empty struct for parameters.
func NewGetUsageInternetOfThingsOptionalParameters() *GetUsageInternetOfThingsOptionalParameters {
	this := GetUsageInternetOfThingsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageInternetOfThingsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageInternetOfThingsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageInternetOfThings Get hourly usage for IoT.
// Get hourly usage for IoT.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageInternetOfThings(ctx _context.Context, startHr time.Time, o ...GetUsageInternetOfThingsOptionalParameters) (UsageIoTResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageIoTResponse
		optionalParams      GetUsageInternetOfThingsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageInternetOfThingsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageInternetOfThings")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/iot"

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

// GetUsageLambdaOptionalParameters holds optional parameters for GetUsageLambda.
type GetUsageLambdaOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageLambdaOptionalParameters creates an empty struct for parameters.
func NewGetUsageLambdaOptionalParameters() *GetUsageLambdaOptionalParameters {
	this := GetUsageLambdaOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageLambdaOptionalParameters) WithEndHr(endHr time.Time) *GetUsageLambdaOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageLambda Get hourly usage for lambda.
// Get hourly usage for lambda.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageLambda(ctx _context.Context, startHr time.Time, o ...GetUsageLambdaOptionalParameters) (UsageLambdaResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageLambdaResponse
		optionalParams      GetUsageLambdaOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageLambdaOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageLambda")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/aws_lambda"

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

// GetUsageLogsOptionalParameters holds optional parameters for GetUsageLogs.
type GetUsageLogsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageLogsOptionalParameters creates an empty struct for parameters.
func NewGetUsageLogsOptionalParameters() *GetUsageLogsOptionalParameters {
	this := GetUsageLogsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageLogsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageLogsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageLogs Get hourly usage for logs.
// Get hourly usage for logs.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageLogs(ctx _context.Context, startHr time.Time, o ...GetUsageLogsOptionalParameters) (UsageLogsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageLogsResponse
		optionalParams      GetUsageLogsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageLogsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/logs"

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

// GetUsageLogsByIndexOptionalParameters holds optional parameters for GetUsageLogsByIndex.
type GetUsageLogsByIndexOptionalParameters struct {
	EndHr     *time.Time
	IndexName *[]string
}

// NewGetUsageLogsByIndexOptionalParameters creates an empty struct for parameters.
func NewGetUsageLogsByIndexOptionalParameters() *GetUsageLogsByIndexOptionalParameters {
	this := GetUsageLogsByIndexOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageLogsByIndexOptionalParameters) WithEndHr(endHr time.Time) *GetUsageLogsByIndexOptionalParameters {
	r.EndHr = &endHr
	return r
}

// WithIndexName sets the corresponding parameter name and returns the struct.
func (r *GetUsageLogsByIndexOptionalParameters) WithIndexName(indexName []string) *GetUsageLogsByIndexOptionalParameters {
	r.IndexName = &indexName
	return r
}

// GetUsageLogsByIndex Get hourly usage for logs by index.
// Get hourly usage for logs by index.
func (a *UsageMeteringApi) GetUsageLogsByIndex(ctx _context.Context, startHr time.Time, o ...GetUsageLogsByIndexOptionalParameters) (UsageLogsByIndexResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageLogsByIndexResponse
		optionalParams      GetUsageLogsByIndexOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageLogsByIndexOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageLogsByIndex")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/logs_by_index"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	if optionalParams.IndexName != nil {
		t := *optionalParams.IndexName
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				localVarQueryParams.Add("index_name", datadog.ParameterToString(s.Index(i), "multi"))
			}
		} else {
			localVarQueryParams.Add("index_name", datadog.ParameterToString(t, "multi"))
		}
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

// GetUsageLogsByRetentionOptionalParameters holds optional parameters for GetUsageLogsByRetention.
type GetUsageLogsByRetentionOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageLogsByRetentionOptionalParameters creates an empty struct for parameters.
func NewGetUsageLogsByRetentionOptionalParameters() *GetUsageLogsByRetentionOptionalParameters {
	this := GetUsageLogsByRetentionOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageLogsByRetentionOptionalParameters) WithEndHr(endHr time.Time) *GetUsageLogsByRetentionOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageLogsByRetention Get hourly logs usage by retention.
// Get hourly usage for indexed logs by retention period.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageLogsByRetention(ctx _context.Context, startHr time.Time, o ...GetUsageLogsByRetentionOptionalParameters) (UsageLogsByRetentionResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageLogsByRetentionResponse
		optionalParams      GetUsageLogsByRetentionOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageLogsByRetentionOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageLogsByRetention")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/logs-by-retention"

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

// GetUsageNetworkFlowsOptionalParameters holds optional parameters for GetUsageNetworkFlows.
type GetUsageNetworkFlowsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageNetworkFlowsOptionalParameters creates an empty struct for parameters.
func NewGetUsageNetworkFlowsOptionalParameters() *GetUsageNetworkFlowsOptionalParameters {
	this := GetUsageNetworkFlowsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageNetworkFlowsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageNetworkFlowsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageNetworkFlows get hourly usage for network flows.
// Get hourly usage for network flows.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageNetworkFlows(ctx _context.Context, startHr time.Time, o ...GetUsageNetworkFlowsOptionalParameters) (UsageNetworkFlowsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageNetworkFlowsResponse
		optionalParams      GetUsageNetworkFlowsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageNetworkFlowsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageNetworkFlows")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/network_flows"

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

// GetUsageNetworkHostsOptionalParameters holds optional parameters for GetUsageNetworkHosts.
type GetUsageNetworkHostsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageNetworkHostsOptionalParameters creates an empty struct for parameters.
func NewGetUsageNetworkHostsOptionalParameters() *GetUsageNetworkHostsOptionalParameters {
	this := GetUsageNetworkHostsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageNetworkHostsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageNetworkHostsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageNetworkHosts Get hourly usage for network hosts.
// Get hourly usage for network hosts.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageNetworkHosts(ctx _context.Context, startHr time.Time, o ...GetUsageNetworkHostsOptionalParameters) (UsageNetworkHostsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageNetworkHostsResponse
		optionalParams      GetUsageNetworkHostsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageNetworkHostsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageNetworkHosts")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/network_hosts"

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

// GetUsageOnlineArchiveOptionalParameters holds optional parameters for GetUsageOnlineArchive.
type GetUsageOnlineArchiveOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageOnlineArchiveOptionalParameters creates an empty struct for parameters.
func NewGetUsageOnlineArchiveOptionalParameters() *GetUsageOnlineArchiveOptionalParameters {
	this := GetUsageOnlineArchiveOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageOnlineArchiveOptionalParameters) WithEndHr(endHr time.Time) *GetUsageOnlineArchiveOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageOnlineArchive Get hourly usage for online archive.
// Get hourly usage for online archive.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageOnlineArchive(ctx _context.Context, startHr time.Time, o ...GetUsageOnlineArchiveOptionalParameters) (UsageOnlineArchiveResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageOnlineArchiveResponse
		optionalParams      GetUsageOnlineArchiveOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageOnlineArchiveOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageOnlineArchive")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/online-archive"

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

// GetUsageProfilingOptionalParameters holds optional parameters for GetUsageProfiling.
type GetUsageProfilingOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageProfilingOptionalParameters creates an empty struct for parameters.
func NewGetUsageProfilingOptionalParameters() *GetUsageProfilingOptionalParameters {
	this := GetUsageProfilingOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageProfilingOptionalParameters) WithEndHr(endHr time.Time) *GetUsageProfilingOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageProfiling Get hourly usage for profiled hosts.
// Get hourly usage for profiled hosts.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageProfiling(ctx _context.Context, startHr time.Time, o ...GetUsageProfilingOptionalParameters) (UsageProfilingResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageProfilingResponse
		optionalParams      GetUsageProfilingOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageProfilingOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageProfiling")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/profiling"

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

// GetUsageRumSessionsOptionalParameters holds optional parameters for GetUsageRumSessions.
type GetUsageRumSessionsOptionalParameters struct {
	EndHr *time.Time
	Type  *string
}

// NewGetUsageRumSessionsOptionalParameters creates an empty struct for parameters.
func NewGetUsageRumSessionsOptionalParameters() *GetUsageRumSessionsOptionalParameters {
	this := GetUsageRumSessionsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageRumSessionsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageRumSessionsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// WithType sets the corresponding parameter name and returns the struct.
func (r *GetUsageRumSessionsOptionalParameters) WithType(typeVar string) *GetUsageRumSessionsOptionalParameters {
	r.Type = &typeVar
	return r
}

// GetUsageRumSessions Get hourly usage for RUM sessions.
// Get hourly usage for [RUM](https://docs.datadoghq.com/real_user_monitoring/) Sessions.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageRumSessions(ctx _context.Context, startHr time.Time, o ...GetUsageRumSessionsOptionalParameters) (UsageRumSessionsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageRumSessionsResponse
		optionalParams      GetUsageRumSessionsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageRumSessionsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageRumSessions")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/rum_sessions"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_hr", datadog.ParameterToString(startHr, ""))
	if optionalParams.EndHr != nil {
		localVarQueryParams.Add("end_hr", datadog.ParameterToString(*optionalParams.EndHr, ""))
	}
	if optionalParams.Type != nil {
		localVarQueryParams.Add("type", datadog.ParameterToString(*optionalParams.Type, ""))
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

// GetUsageRumUnitsOptionalParameters holds optional parameters for GetUsageRumUnits.
type GetUsageRumUnitsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageRumUnitsOptionalParameters creates an empty struct for parameters.
func NewGetUsageRumUnitsOptionalParameters() *GetUsageRumUnitsOptionalParameters {
	this := GetUsageRumUnitsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageRumUnitsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageRumUnitsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageRumUnits Get hourly usage for RUM units.
// Get hourly usage for [RUM](https://docs.datadoghq.com/real_user_monitoring/) Units.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageRumUnits(ctx _context.Context, startHr time.Time, o ...GetUsageRumUnitsOptionalParameters) (UsageRumUnitsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageRumUnitsResponse
		optionalParams      GetUsageRumUnitsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageRumUnitsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageRumUnits")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/rum"

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

// GetUsageSDSOptionalParameters holds optional parameters for GetUsageSDS.
type GetUsageSDSOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageSDSOptionalParameters creates an empty struct for parameters.
func NewGetUsageSDSOptionalParameters() *GetUsageSDSOptionalParameters {
	this := GetUsageSDSOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageSDSOptionalParameters) WithEndHr(endHr time.Time) *GetUsageSDSOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageSDS Get hourly usage for sensitive data scanner.
// Get hourly usage for sensitive data scanner.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageSDS(ctx _context.Context, startHr time.Time, o ...GetUsageSDSOptionalParameters) (UsageSDSResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSDSResponse
		optionalParams      GetUsageSDSOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSDSOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSDS")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/sds"

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

// GetUsageSNMPOptionalParameters holds optional parameters for GetUsageSNMP.
type GetUsageSNMPOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageSNMPOptionalParameters creates an empty struct for parameters.
func NewGetUsageSNMPOptionalParameters() *GetUsageSNMPOptionalParameters {
	this := GetUsageSNMPOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageSNMPOptionalParameters) WithEndHr(endHr time.Time) *GetUsageSNMPOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageSNMP Get hourly usage for SNMP devices.
// Get hourly usage for SNMP devices.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageSNMP(ctx _context.Context, startHr time.Time, o ...GetUsageSNMPOptionalParameters) (UsageSNMPResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSNMPResponse
		optionalParams      GetUsageSNMPOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSNMPOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSNMP")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/snmp"

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

// GetUsageSummaryOptionalParameters holds optional parameters for GetUsageSummary.
type GetUsageSummaryOptionalParameters struct {
	EndMonth          *time.Time
	IncludeOrgDetails *bool
}

// NewGetUsageSummaryOptionalParameters creates an empty struct for parameters.
func NewGetUsageSummaryOptionalParameters() *GetUsageSummaryOptionalParameters {
	this := GetUsageSummaryOptionalParameters{}
	return &this
}

// WithEndMonth sets the corresponding parameter name and returns the struct.
func (r *GetUsageSummaryOptionalParameters) WithEndMonth(endMonth time.Time) *GetUsageSummaryOptionalParameters {
	r.EndMonth = &endMonth
	return r
}

// WithIncludeOrgDetails sets the corresponding parameter name and returns the struct.
func (r *GetUsageSummaryOptionalParameters) WithIncludeOrgDetails(includeOrgDetails bool) *GetUsageSummaryOptionalParameters {
	r.IncludeOrgDetails = &includeOrgDetails
	return r
}

// GetUsageSummary Get usage across your account.
// Get all usage across your account.
func (a *UsageMeteringApi) GetUsageSummary(ctx _context.Context, startMonth time.Time, o ...GetUsageSummaryOptionalParameters) (UsageSummaryResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSummaryResponse
		optionalParams      GetUsageSummaryOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSummaryOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSummary")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/summary"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	localVarQueryParams.Add("start_month", datadog.ParameterToString(startMonth, ""))
	if optionalParams.EndMonth != nil {
		localVarQueryParams.Add("end_month", datadog.ParameterToString(*optionalParams.EndMonth, ""))
	}
	if optionalParams.IncludeOrgDetails != nil {
		localVarQueryParams.Add("include_org_details", datadog.ParameterToString(*optionalParams.IncludeOrgDetails, ""))
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

// GetUsageSyntheticsOptionalParameters holds optional parameters for GetUsageSynthetics.
type GetUsageSyntheticsOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageSyntheticsOptionalParameters creates an empty struct for parameters.
func NewGetUsageSyntheticsOptionalParameters() *GetUsageSyntheticsOptionalParameters {
	this := GetUsageSyntheticsOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageSyntheticsOptionalParameters) WithEndHr(endHr time.Time) *GetUsageSyntheticsOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageSynthetics Get hourly usage for synthetics checks.
// Get hourly usage for [synthetics checks](https://docs.datadoghq.com/synthetics/).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
//
// Deprecated: This API is deprecated.
func (a *UsageMeteringApi) GetUsageSynthetics(ctx _context.Context, startHr time.Time, o ...GetUsageSyntheticsOptionalParameters) (UsageSyntheticsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSyntheticsResponse
		optionalParams      GetUsageSyntheticsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSyntheticsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSynthetics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/synthetics"

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

// GetUsageSyntheticsAPIOptionalParameters holds optional parameters for GetUsageSyntheticsAPI.
type GetUsageSyntheticsAPIOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageSyntheticsAPIOptionalParameters creates an empty struct for parameters.
func NewGetUsageSyntheticsAPIOptionalParameters() *GetUsageSyntheticsAPIOptionalParameters {
	this := GetUsageSyntheticsAPIOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageSyntheticsAPIOptionalParameters) WithEndHr(endHr time.Time) *GetUsageSyntheticsAPIOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageSyntheticsAPI Get hourly usage for synthetics API checks.
// Get hourly usage for [synthetics API checks](https://docs.datadoghq.com/synthetics/).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageSyntheticsAPI(ctx _context.Context, startHr time.Time, o ...GetUsageSyntheticsAPIOptionalParameters) (UsageSyntheticsAPIResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSyntheticsAPIResponse
		optionalParams      GetUsageSyntheticsAPIOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSyntheticsAPIOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSyntheticsAPI")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/synthetics_api"

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

// GetUsageSyntheticsBrowserOptionalParameters holds optional parameters for GetUsageSyntheticsBrowser.
type GetUsageSyntheticsBrowserOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageSyntheticsBrowserOptionalParameters creates an empty struct for parameters.
func NewGetUsageSyntheticsBrowserOptionalParameters() *GetUsageSyntheticsBrowserOptionalParameters {
	this := GetUsageSyntheticsBrowserOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageSyntheticsBrowserOptionalParameters) WithEndHr(endHr time.Time) *GetUsageSyntheticsBrowserOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageSyntheticsBrowser Get hourly usage for synthetics browser checks.
// Get hourly usage for synthetics browser checks.
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageSyntheticsBrowser(ctx _context.Context, startHr time.Time, o ...GetUsageSyntheticsBrowserOptionalParameters) (UsageSyntheticsBrowserResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageSyntheticsBrowserResponse
		optionalParams      GetUsageSyntheticsBrowserOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageSyntheticsBrowserOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageSyntheticsBrowser")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/synthetics_browser"

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

// GetUsageTimeseriesOptionalParameters holds optional parameters for GetUsageTimeseries.
type GetUsageTimeseriesOptionalParameters struct {
	EndHr *time.Time
}

// NewGetUsageTimeseriesOptionalParameters creates an empty struct for parameters.
func NewGetUsageTimeseriesOptionalParameters() *GetUsageTimeseriesOptionalParameters {
	this := GetUsageTimeseriesOptionalParameters{}
	return &this
}

// WithEndHr sets the corresponding parameter name and returns the struct.
func (r *GetUsageTimeseriesOptionalParameters) WithEndHr(endHr time.Time) *GetUsageTimeseriesOptionalParameters {
	r.EndHr = &endHr
	return r
}

// GetUsageTimeseries Get hourly usage for custom metrics.
// Get hourly usage for [custom metrics](https://docs.datadoghq.com/developers/metrics/custom_metrics/).
// **Note:** hourly usage data for all products is now available in the [Get hourly usage by product family API](https://docs.datadoghq.com/api/latest/usage-metering/#get-hourly-usage-by-product-family). Refer to [Migrating from the V1 Hourly Usage APIs to V2](https://docs.datadoghq.com/account_management/guide/hourly-usage-migration/) for the associated migration guide.
func (a *UsageMeteringApi) GetUsageTimeseries(ctx _context.Context, startHr time.Time, o ...GetUsageTimeseriesOptionalParameters) (UsageTimeseriesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageTimeseriesResponse
		optionalParams      GetUsageTimeseriesOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageTimeseriesOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageTimeseries")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/timeseries"

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

// GetUsageTopAvgMetricsOptionalParameters holds optional parameters for GetUsageTopAvgMetrics.
type GetUsageTopAvgMetricsOptionalParameters struct {
	Month        *time.Time
	Day          *time.Time
	Names        *[]string
	Limit        *int32
	NextRecordId *string
}

// NewGetUsageTopAvgMetricsOptionalParameters creates an empty struct for parameters.
func NewGetUsageTopAvgMetricsOptionalParameters() *GetUsageTopAvgMetricsOptionalParameters {
	this := GetUsageTopAvgMetricsOptionalParameters{}
	return &this
}

// WithMonth sets the corresponding parameter name and returns the struct.
func (r *GetUsageTopAvgMetricsOptionalParameters) WithMonth(month time.Time) *GetUsageTopAvgMetricsOptionalParameters {
	r.Month = &month
	return r
}

// WithDay sets the corresponding parameter name and returns the struct.
func (r *GetUsageTopAvgMetricsOptionalParameters) WithDay(day time.Time) *GetUsageTopAvgMetricsOptionalParameters {
	r.Day = &day
	return r
}

// WithNames sets the corresponding parameter name and returns the struct.
func (r *GetUsageTopAvgMetricsOptionalParameters) WithNames(names []string) *GetUsageTopAvgMetricsOptionalParameters {
	r.Names = &names
	return r
}

// WithLimit sets the corresponding parameter name and returns the struct.
func (r *GetUsageTopAvgMetricsOptionalParameters) WithLimit(limit int32) *GetUsageTopAvgMetricsOptionalParameters {
	r.Limit = &limit
	return r
}

// WithNextRecordId sets the corresponding parameter name and returns the struct.
func (r *GetUsageTopAvgMetricsOptionalParameters) WithNextRecordId(nextRecordId string) *GetUsageTopAvgMetricsOptionalParameters {
	r.NextRecordId = &nextRecordId
	return r
}

// GetUsageTopAvgMetrics Get all custom metrics by hourly average.
// Get all [custom metrics](https://docs.datadoghq.com/developers/metrics/custom_metrics/) by hourly average. Use the month parameter to get a month-to-date data resolution or use the day parameter to get a daily resolution. One of the two is required, and only one of the two is allowed.
func (a *UsageMeteringApi) GetUsageTopAvgMetrics(ctx _context.Context, o ...GetUsageTopAvgMetricsOptionalParameters) (UsageTopAvgMetricsResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue UsageTopAvgMetricsResponse
		optionalParams      GetUsageTopAvgMetricsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetUsageTopAvgMetricsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.UsageMeteringApi.GetUsageTopAvgMetrics")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/usage/top_avg_metrics"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Month != nil {
		localVarQueryParams.Add("month", datadog.ParameterToString(*optionalParams.Month, ""))
	}
	if optionalParams.Day != nil {
		localVarQueryParams.Add("day", datadog.ParameterToString(*optionalParams.Day, ""))
	}
	if optionalParams.Names != nil {
		t := *optionalParams.Names
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				localVarQueryParams.Add("names", datadog.ParameterToString(s.Index(i), "multi"))
			}
		} else {
			localVarQueryParams.Add("names", datadog.ParameterToString(t, "multi"))
		}
	}
	if optionalParams.Limit != nil {
		localVarQueryParams.Add("limit", datadog.ParameterToString(*optionalParams.Limit, ""))
	}
	if optionalParams.NextRecordId != nil {
		localVarQueryParams.Add("next_record_id", datadog.ParameterToString(*optionalParams.NextRecordId, ""))
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
