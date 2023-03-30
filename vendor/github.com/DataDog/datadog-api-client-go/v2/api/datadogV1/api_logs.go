// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// LogsApi service type
type LogsApi datadog.Service

// ListLogs Search logs.
// List endpoint returns logs that match a log search query.
// [Results are paginated][1].
//
// **If you are considering archiving logs for your organization,
// consider use of the Datadog archive capabilities instead of the log list API.
// See [Datadog Logs Archive documentation][2].**
//
// [1]: /logs/guide/collect-multiple-logs-with-pagination
// [2]: https://docs.datadoghq.com/logs/archives
func (a *LogsApi) ListLogs(ctx _context.Context, body LogsListRequest) (LogsListResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue LogsListResponse
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.LogsApi.ListLogs")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/logs-queries/list"

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
		if localVarHTTPResponse.StatusCode == 400 {
			var v LogsAPIErrorResponse
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
			return localVarReturnValue, localVarHTTPResponse, newErr
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

// SubmitLogOptionalParameters holds optional parameters for SubmitLog.
type SubmitLogOptionalParameters struct {
	ContentEncoding *ContentEncoding
	Ddtags          *string
}

// NewSubmitLogOptionalParameters creates an empty struct for parameters.
func NewSubmitLogOptionalParameters() *SubmitLogOptionalParameters {
	this := SubmitLogOptionalParameters{}
	return &this
}

// WithContentEncoding sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithContentEncoding(contentEncoding ContentEncoding) *SubmitLogOptionalParameters {
	r.ContentEncoding = &contentEncoding
	return r
}

// WithDdtags sets the corresponding parameter name and returns the struct.
func (r *SubmitLogOptionalParameters) WithDdtags(ddtags string) *SubmitLogOptionalParameters {
	r.Ddtags = &ddtags
	return r
}

// SubmitLog Send logs.
// Send your logs to your Datadog platform over HTTP. Limits per HTTP request are:
//
// - Maximum content size per payload (uncompressed): 5MB
// - Maximum size for a single log: 1MB
// - Maximum array size if sending multiple logs in an array: 1000 entries
//
// Any log exceeding 1MB is accepted and truncated by Datadog:
// - For a single log request, the API truncates the log at 1MB and returns a 2xx.
// - For a multi-logs request, the API processes all logs, truncates only logs larger than 1MB, and returns a 2xx.
//
// Datadog recommends sending your logs compressed.
// Add the `Content-Encoding: gzip` header to the request when sending compressed logs.
//
// The status codes answered by the HTTP API are:
// - 200: OK
// - 400: Bad request (likely an issue in the payload formatting)
// - 403: Permission issue (likely using an invalid API Key)
// - 413: Payload too large (batch is above 5MB uncompressed)
// - 5xx: Internal error, request should be retried after some time
//
// Deprecated: This API is deprecated.
func (a *LogsApi) SubmitLog(ctx _context.Context, body []HTTPLogItem, o ...SubmitLogOptionalParameters) (interface{}, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodPost
		localVarPostBody    interface{}
		localVarReturnValue interface{}
		optionalParams      SubmitLogOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type SubmitLogOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v1.LogsApi.SubmitLog")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/v1/input"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Ddtags != nil {
		localVarQueryParams.Add("ddtags", datadog.ParameterToString(*optionalParams.Ddtags, ""))
	}
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
		if localVarHTTPResponse.StatusCode == 400 {
			var v HTTPLogError
			err = a.Client.Decode(&v, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
			if err != nil {
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.ErrorModel = v
			return localVarReturnValue, localVarHTTPResponse, newErr
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

// NewLogsApi Returns NewLogsApi.
func NewLogsApi(client *datadog.APIClient) *LogsApi {
	return &LogsApi{
		Client: client,
	}
}
