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

// HostsApi service type
type HostsApi datadog.Service

// GetHostTotalsOptionalParameters holds optional parameters for GetHostTotals.
type GetHostTotalsOptionalParameters struct {
	From *int64
}

// NewGetHostTotalsOptionalParameters creates an empty struct for parameters.
func NewGetHostTotalsOptionalParameters() *GetHostTotalsOptionalParameters {
	this := GetHostTotalsOptionalParameters{}
	return &this
}

// WithFrom sets the corresponding parameter name and returns the struct.
func (r *GetHostTotalsOptionalParameters) WithFrom(from int64) *GetHostTotalsOptionalParameters {
	r.From = &from
	return r
}

// GetHostTotals Get the total number of active hosts.
// This endpoint returns the total number of active and up hosts in your Datadog account.
// Active means the host has reported in the past hour, and up means it has reported in the past two hours.
func (a *HostsApi) GetHostTotals(ctx _context.Context, o ...GetHostTotalsOptionalParameters) (HostTotals, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HostTotals
		optionalParams      GetHostTotalsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type GetHostTotalsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.HostsApi.GetHostTotals")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/hosts/totals"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.From != nil {
		localVarQueryParams.Add("from", datadog.ParameterToString(*optionalParams.From, ""))
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

// ListHostsOptionalParameters holds optional parameters for ListHosts.
type ListHostsOptionalParameters struct {
	Filter                *string
	SortField             *string
	SortDir               *string
	Start                 *int64
	Count                 *int64
	From                  *int64
	IncludeMutedHostsData *bool
	IncludeHostsMetadata  *bool
}

// NewListHostsOptionalParameters creates an empty struct for parameters.
func NewListHostsOptionalParameters() *ListHostsOptionalParameters {
	this := ListHostsOptionalParameters{}
	return &this
}

// WithFilter sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithFilter(filter string) *ListHostsOptionalParameters {
	r.Filter = &filter
	return r
}

// WithSortField sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithSortField(sortField string) *ListHostsOptionalParameters {
	r.SortField = &sortField
	return r
}

// WithSortDir sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithSortDir(sortDir string) *ListHostsOptionalParameters {
	r.SortDir = &sortDir
	return r
}

// WithStart sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithStart(start int64) *ListHostsOptionalParameters {
	r.Start = &start
	return r
}

// WithCount sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithCount(count int64) *ListHostsOptionalParameters {
	r.Count = &count
	return r
}

// WithFrom sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithFrom(from int64) *ListHostsOptionalParameters {
	r.From = &from
	return r
}

// WithIncludeMutedHostsData sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithIncludeMutedHostsData(includeMutedHostsData bool) *ListHostsOptionalParameters {
	r.IncludeMutedHostsData = &includeMutedHostsData
	return r
}

// WithIncludeHostsMetadata sets the corresponding parameter name and returns the struct.
func (r *ListHostsOptionalParameters) WithIncludeHostsMetadata(includeHostsMetadata bool) *ListHostsOptionalParameters {
	r.IncludeHostsMetadata = &includeHostsMetadata
	return r
}

// ListHosts Get all hosts for your organization.
// This endpoint allows searching for hosts by name, alias, or tag.
// Hosts live within the past 3 hours are included by default.
// Retention is 7 days.
// Results are paginated with a max of 1000 results at a time.
func (a *HostsApi) ListHosts(ctx _context.Context, o ...ListHostsOptionalParameters) (HostListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue HostListResponse
		optionalParams      ListHostsOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListHostsOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.HostsApi.ListHosts")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/hosts"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Filter != nil {
		localVarQueryParams.Add("filter", datadog.ParameterToString(*optionalParams.Filter, ""))
	}
	if optionalParams.SortField != nil {
		localVarQueryParams.Add("sort_field", datadog.ParameterToString(*optionalParams.SortField, ""))
	}
	if optionalParams.SortDir != nil {
		localVarQueryParams.Add("sort_dir", datadog.ParameterToString(*optionalParams.SortDir, ""))
	}
	if optionalParams.Start != nil {
		localVarQueryParams.Add("start", datadog.ParameterToString(*optionalParams.Start, ""))
	}
	if optionalParams.Count != nil {
		localVarQueryParams.Add("count", datadog.ParameterToString(*optionalParams.Count, ""))
	}
	if optionalParams.From != nil {
		localVarQueryParams.Add("from", datadog.ParameterToString(*optionalParams.From, ""))
	}
	if optionalParams.IncludeMutedHostsData != nil {
		localVarQueryParams.Add("include_muted_hosts_data", datadog.ParameterToString(*optionalParams.IncludeMutedHostsData, ""))
	}
	if optionalParams.IncludeHostsMetadata != nil {
		localVarQueryParams.Add("include_hosts_metadata", datadog.ParameterToString(*optionalParams.IncludeHostsMetadata, ""))
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

// MuteHost Mute a host.
// Mute a host.
func (a *HostsApi) MuteHost(ctx _context.Context, hostName string, body HostMuteSettings) (HostMuteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue HostMuteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.HostsApi.MuteHost")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/host/{host_name}/mute"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(hostName, "")), -1)

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

// UnmuteHost Unmute a host.
// Unmutes a host. This endpoint takes no JSON arguments.
func (a *HostsApi) UnmuteHost(ctx _context.Context, hostName string) (HostMuteResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue HostMuteResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.HostsApi.UnmuteHost")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/host/{host_name}/unmute"
	localVarPath = strings.Replace(localVarPath, "{"+"host_name"+"}", _neturl.PathEscape(datadog.ParameterToString(hostName, "")), -1)

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

// NewHostsApi Returns NewHostsApi.
func NewHostsApi(client *datadog.APIClient) *HostsApi {
	return &HostsApi{
		Client: client,
	}
}
