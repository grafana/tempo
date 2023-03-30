// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	_context "context"
	_nethttp "net/http"
	_neturl "net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// ProcessesApi service type
type ProcessesApi datadog.Service

// ListProcessesOptionalParameters holds optional parameters for ListProcesses.
type ListProcessesOptionalParameters struct {
	Search     *string
	Tags       *string
	From       *int64
	To         *int64
	PageLimit  *int32
	PageCursor *string
}

// NewListProcessesOptionalParameters creates an empty struct for parameters.
func NewListProcessesOptionalParameters() *ListProcessesOptionalParameters {
	this := ListProcessesOptionalParameters{}
	return &this
}

// WithSearch sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithSearch(search string) *ListProcessesOptionalParameters {
	r.Search = &search
	return r
}

// WithTags sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithTags(tags string) *ListProcessesOptionalParameters {
	r.Tags = &tags
	return r
}

// WithFrom sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithFrom(from int64) *ListProcessesOptionalParameters {
	r.From = &from
	return r
}

// WithTo sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithTo(to int64) *ListProcessesOptionalParameters {
	r.To = &to
	return r
}

// WithPageLimit sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithPageLimit(pageLimit int32) *ListProcessesOptionalParameters {
	r.PageLimit = &pageLimit
	return r
}

// WithPageCursor sets the corresponding parameter name and returns the struct.
func (r *ListProcessesOptionalParameters) WithPageCursor(pageCursor string) *ListProcessesOptionalParameters {
	r.PageCursor = &pageCursor
	return r
}

// ListProcesses Get all processes.
// Get all processes for your organization.
func (a *ProcessesApi) ListProcesses(ctx _context.Context, o ...ListProcessesOptionalParameters) (ProcessSummariesResponse, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue ProcessSummariesResponse
		optionalParams      ListProcessesOptionalParameters
	)

	if len(o) > 1 {
		return localVarReturnValue, nil, datadog.ReportError("only one argument of type ListProcessesOptionalParameters is allowed")
	}
	if len(o) == 1 {
		optionalParams = o[0]
	}

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(ctx, "v2.ProcessesApi.ListProcesses")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v2/processes"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if optionalParams.Search != nil {
		localVarQueryParams.Add("search", datadog.ParameterToString(*optionalParams.Search, ""))
	}
	if optionalParams.Tags != nil {
		localVarQueryParams.Add("tags", datadog.ParameterToString(*optionalParams.Tags, ""))
	}
	if optionalParams.From != nil {
		localVarQueryParams.Add("from", datadog.ParameterToString(*optionalParams.From, ""))
	}
	if optionalParams.To != nil {
		localVarQueryParams.Add("to", datadog.ParameterToString(*optionalParams.To, ""))
	}
	if optionalParams.PageLimit != nil {
		localVarQueryParams.Add("page[limit]", datadog.ParameterToString(*optionalParams.PageLimit, ""))
	}
	if optionalParams.PageCursor != nil {
		localVarQueryParams.Add("page[cursor]", datadog.ParameterToString(*optionalParams.PageCursor, ""))
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

// ListProcessesWithPagination provides a paginated version of ListProcesses returning a channel with all items.
func (a *ProcessesApi) ListProcessesWithPagination(ctx _context.Context, o ...ListProcessesOptionalParameters) (<-chan datadog.PaginationResult[ProcessSummary], func()) {
	ctx, cancel := _context.WithCancel(ctx)
	pageSize_ := int32(1000)
	if len(o) == 0 {
		o = append(o, ListProcessesOptionalParameters{})
	}
	if o[0].PageLimit != nil {
		pageSize_ = *o[0].PageLimit
	}
	o[0].PageLimit = &pageSize_

	items := make(chan datadog.PaginationResult[ProcessSummary], pageSize_)
	go func() {
		for {
			resp, _, err := a.ListProcesses(ctx, o...)
			if err != nil {
				var returnItem ProcessSummary
				items <- datadog.PaginationResult[ProcessSummary]{returnItem, err}
				break
			}
			respData, ok := resp.GetDataOk()
			if !ok {
				break
			}
			results := *respData

			for _, item := range results {
				select {
				case items <- datadog.PaginationResult[ProcessSummary]{item, nil}:
				case <-ctx.Done():
					close(items)
					return
				}
			}
			if len(results) < int(pageSize_) {
				break
			}
			cursorMeta, ok := resp.GetMetaOk()
			if !ok {
				break
			}
			cursorMetaPage, ok := cursorMeta.GetPageOk()
			if !ok {
				break
			}
			cursorMetaPageAfter, ok := cursorMetaPage.GetAfterOk()
			if !ok {
				break
			}

			o[0].PageCursor = cursorMetaPageAfter
		}
		close(items)
	}()
	return items, cancel
}

// NewProcessesApi Returns NewProcessesApi.
func NewProcessesApi(client *datadog.APIClient) *ProcessesApi {
	return &ProcessesApi{
		Client: client,
	}
}
