package pipeline

import (
	"context"
	"net/http"
)

// this file exists to consolidate and clearly document all context keys that are valid and recognized by the pipeline package

// contextEchoData is used to echo request specific data through the pipeline. It stores any value.
// see usage for samplingRate in modules/frontend/metrics_query_range_sharder.go
var contextRequestDataForResponse = struct{}{}

// ContextAddResponseDataForResponse adds a value to the request context that will be echoed back in the response.
func ContextAddResponseDataForResponse(val any, req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), contextRequestDataForResponse, val))
}
