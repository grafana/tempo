package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	serverless "github.com/grafana/tempo/cmd/tempo-serverless"
	"github.com/grafana/tempo/pkg/tempopb"
)

func HandleLambdaEvent(event events.ALBTargetGroupRequest) (*tempopb.SearchResponse, error) {
	// jpe do some magic to create an http.Requeset{}

	resp, httpErr := serverless.Handler(&http.Request{})
	if httpErr != nil {
		// jpe how do i return a specific status code?
		return nil, httpErr.Err
	}

	return resp, nil
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
