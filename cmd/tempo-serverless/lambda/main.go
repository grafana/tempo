package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gogo/protobuf/jsonpb"

	serverless "github.com/grafana/tempo/v2/cmd/tempo-serverless"
)

func main() {
	lambda.Start(HandleLambdaEvent)
}

func HandleLambdaEvent(ctx context.Context, event events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	req, err := httpRequest(event)
	if err != nil {
		return events.ALBTargetGroupResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{}, // alb will 502 if we don't set this: https://github.com/awslabs/aws-lambda-go-api-proxy/issues/79
		}, nil
	}

	resp, httpErr := serverless.Handler(req.WithContext(ctx))
	if httpErr != nil {
		return events.ALBTargetGroupResponse{
			Body:       httpErr.Err.Error(),
			StatusCode: httpErr.Status,
			Headers:    map[string]string{},
		}, nil
	}

	marshaller := &jsonpb.Marshaler{}
	body, err := marshaller.MarshalToString(resp)
	if err != nil {
		return events.ALBTargetGroupResponse{
			Body:       err.Error(),
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{},
		}, nil
	}

	return events.ALBTargetGroupResponse{
		Body:       body,
		StatusCode: http.StatusOK,
		Headers:    map[string]string{},
	}, nil
}

// adapted with love from: https://github.com/akrylysov/algnhsa/blob/4c6f78589c506c0f060512adf96b97e8285ac80c/request.go#L65
func httpRequest(event events.ALBTargetGroupRequest) (*http.Request, error) {
	// params
	params := url.Values{}
	for k, v := range event.QueryStringParameters {
		unescaped, err := url.QueryUnescape(v)
		if err != nil {
			return nil, fmt.Errorf("failed to unescape query string parameter %s: %s: %w", k, v, err)
		}
		params.Set(k, unescaped)
	}
	for k, vals := range event.MultiValueQueryStringParameters {
		for i, v := range vals {
			var err error
			vals[i], err = url.QueryUnescape(v)
			if err != nil {
				return nil, fmt.Errorf("failed to unescape multi val query string parameter %s: %s: %w", k, v, err)
			}
		}
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
