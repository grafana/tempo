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

// SnapshotsApi service type
type SnapshotsApi datadog.Service

type apiGetGraphSnapshotRequest struct {
	ctx         _context.Context
	start       *int64
	end         *int64
	metricQuery *string
	eventQuery  *string
	graphDef    *string
	title       *string
	height      *int64
	width       *int64
}

// GetGraphSnapshotOptionalParameters holds optional parameters for GetGraphSnapshot.
type GetGraphSnapshotOptionalParameters struct {
	MetricQuery *string
	EventQuery  *string
	GraphDef    *string
	Title       *string
	Height      *int64
	Width       *int64
}

// NewGetGraphSnapshotOptionalParameters creates an empty struct for parameters.
func NewGetGraphSnapshotOptionalParameters() *GetGraphSnapshotOptionalParameters {
	this := GetGraphSnapshotOptionalParameters{}
	return &this
}

// WithMetricQuery sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithMetricQuery(metricQuery string) *GetGraphSnapshotOptionalParameters {
	r.MetricQuery = &metricQuery
	return r
}

// WithEventQuery sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithEventQuery(eventQuery string) *GetGraphSnapshotOptionalParameters {
	r.EventQuery = &eventQuery
	return r
}

// WithGraphDef sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithGraphDef(graphDef string) *GetGraphSnapshotOptionalParameters {
	r.GraphDef = &graphDef
	return r
}

// WithTitle sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithTitle(title string) *GetGraphSnapshotOptionalParameters {
	r.Title = &title
	return r
}

// WithHeight sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithHeight(height int64) *GetGraphSnapshotOptionalParameters {
	r.Height = &height
	return r
}

// WithWidth sets the corresponding parameter name and returns the struct.
func (r *GetGraphSnapshotOptionalParameters) WithWidth(width int64) *GetGraphSnapshotOptionalParameters {
	r.Width = &width
	return r
}

func (a *SnapshotsApi) buildGetGraphSnapshotRequest(ctx _context.Context, start int64, end int64, o ...GetGraphSnapshotOptionalParameters) (apiGetGraphSnapshotRequest, error) {
	req := apiGetGraphSnapshotRequest{
		ctx:   ctx,
		start: &start,
		end:   &end,
	}

	if len(o) > 1 {
		return req, datadog.ReportError("only one argument of type GetGraphSnapshotOptionalParameters is allowed")
	}

	if o != nil {
		req.metricQuery = o[0].MetricQuery
		req.eventQuery = o[0].EventQuery
		req.graphDef = o[0].GraphDef
		req.title = o[0].Title
		req.height = o[0].Height
		req.width = o[0].Width
	}
	return req, nil
}

// GetGraphSnapshot Take graph snapshots.
// Take graph snapshots.
// **Note**: When a snapshot is created, there is some delay before it is available.
func (a *SnapshotsApi) GetGraphSnapshot(ctx _context.Context, start int64, end int64, o ...GetGraphSnapshotOptionalParameters) (GraphSnapshot, *_nethttp.Response, error) {
	req, err := a.buildGetGraphSnapshotRequest(ctx, start, end, o...)
	if err != nil {
		var localVarReturnValue GraphSnapshot
		return localVarReturnValue, nil, err
	}

	return a.getGraphSnapshotExecute(req)
}

// getGraphSnapshotExecute executes the request.
func (a *SnapshotsApi) getGraphSnapshotExecute(r apiGetGraphSnapshotRequest) (GraphSnapshot, *_nethttp.Response, error) {
	var (
		localVarHTTPMethod  = _nethttp.MethodGet
		localVarPostBody    interface{}
		localVarReturnValue GraphSnapshot
	)

	localBasePath, err := a.Client.Cfg.ServerURLWithContext(r.ctx, "v1.SnapshotsApi.GetGraphSnapshot")
	if err != nil {
		return localVarReturnValue, nil, datadog.GenericOpenAPIError{ErrorMessage: err.Error()}
	}

	localVarPath := localBasePath + "/api/v1/graph/snapshot"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := _neturl.Values{}
	localVarFormParams := _neturl.Values{}
	if r.start == nil {
		return localVarReturnValue, nil, datadog.ReportError("start is required and must be specified")
	}
	if r.end == nil {
		return localVarReturnValue, nil, datadog.ReportError("end is required and must be specified")
	}
	localVarQueryParams.Add("start", datadog.ParameterToString(*r.start, ""))
	localVarQueryParams.Add("end", datadog.ParameterToString(*r.end, ""))
	if r.metricQuery != nil {
		localVarQueryParams.Add("metric_query", datadog.ParameterToString(*r.metricQuery, ""))
	}
	if r.eventQuery != nil {
		localVarQueryParams.Add("event_query", datadog.ParameterToString(*r.eventQuery, ""))
	}
	if r.graphDef != nil {
		localVarQueryParams.Add("graph_def", datadog.ParameterToString(*r.graphDef, ""))
	}
	if r.title != nil {
		localVarQueryParams.Add("title", datadog.ParameterToString(*r.title, ""))
	}
	if r.height != nil {
		localVarQueryParams.Add("height", datadog.ParameterToString(*r.height, ""))
	}
	if r.width != nil {
		localVarQueryParams.Add("width", datadog.ParameterToString(*r.width, ""))
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

// NewSnapshotsApi Returns NewSnapshotsApi.
func NewSnapshotsApi(client *datadog.APIClient) *SnapshotsApi {
	return &SnapshotsApi{
		Client: client,
	}
}
