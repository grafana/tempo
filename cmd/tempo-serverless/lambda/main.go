package main

import (
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gogo/protobuf/jsonpb"

	serverless "github.com/grafana/tempo/cmd/tempo-serverless"
)

func main() {
	startE2EProxy()
	lambda.Start(HandleLambdaEvent)
}

func HandleLambdaEvent(event events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	req, err := httpRequest(event)
	if err != nil {
		return events.ALBTargetGroupResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	resp, httpErr := serverless.Handler(req)
	if httpErr != nil {
		return events.ALBTargetGroupResponse{
			Body:       httpErr.Err.Error(),
			StatusCode: httpErr.Status,
		}, nil
	}

	marshaller := &jsonpb.Marshaler{}
	body, err := marshaller.MarshalToString(resp)
	if err != nil {
		return events.ALBTargetGroupResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return events.ALBTargetGroupResponse{
		Body:       body,
		StatusCode: http.StatusOK,
	}, nil
}

// adapted with love from: https://github.com/akrylysov/algnhsa/blob/4c6f78589c506c0f060512adf96b97e8285ac80c/request.go#L65
func httpRequest(event events.ALBTargetGroupRequest) (*http.Request, error) {
	// params
	params := url.Values{}
	for k, v := range event.QueryStringParameters {
		params.Set(k, v)
	}
	for k, vals := range event.MultiValueQueryStringParameters {
		params[k] = vals
	}

	// headers
	headers := make(http.Header)
	for k, v := range event.Headers {
		headers.Set(k, v)
	}
	for k, vals := range event.MultiValueHeaders {
		headers[http.CanonicalHeaderKey(k)] = vals
	}

	// url
	u := url.URL{
		Host:     headers.Get("host"),
		RawPath:  event.Path,
		RawQuery: params.Encode(),
	}

	// Unescape request path
	p, err := url.PathUnescape(u.RawPath)
	if err != nil {
		return nil, err
	}
	u.Path = p

	if u.Path == u.RawPath {
		u.RawPath = ""
	}

	// we don't use the body. ignore

	req, err := http.NewRequest(event.HTTPMethod, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.RequestURI = u.RequestURI()
	req.Header = headers

	return req, nil
}
