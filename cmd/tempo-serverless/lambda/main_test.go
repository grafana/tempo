package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

// Test that httpRequest() works as expected
func Test_httpRequest(t *testing.T) {
	tests := []struct {
		event   events.ALBTargetGroupRequest
		want    *http.Request
		wantErr string
	}{
		// empty
		{
			event: events.ALBTargetGroupRequest{},
			want: &http.Request{
				Method:     "GET",
				RequestURI: "/",
				URL:        &url.URL{},
				Header:     map[string][]string{},
			},
		},
		// rando
		{
			event: events.ALBTargetGroupRequest{
				HTTPMethod: "GET",
				Path:       "/",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				QueryStringParameters: map[string]string{
					"query": "test",
				},
				Body: "test",
			},
			want: &http.Request{
				Method:     "GET",
				RequestURI: "/?query=test",
				URL: &url.URL{
					Path:     "/",
					RawQuery: "query=test",
				},
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			},
		},
		// unescape params
		{
			event: events.ALBTargetGroupRequest{
				QueryStringParameters: map[string]string{
					"q": "%7B%20.foo%20%3D%20bar%20%7D",
				},
			},
			want: &http.Request{
				Method:     "GET",
				RequestURI: "/?q=%7B+.foo+%3D+bar+%7D",
				URL: &url.URL{
					RawQuery: "q=%7B+.foo+%3D+bar+%7D",
				},
				Header: map[string][]string{},
			},
		},
		// unescape multivalue params
		{
			event: events.ALBTargetGroupRequest{
				MultiValueQueryStringParameters: map[string][]string{
					"q2": {"%7B%20.foo%20%3D%20bar%20%7D", "%7B%20.foo%20%3D%20bar%20%7D"},
				},
			},
			want: &http.Request{
				Method:     "GET",
				RequestURI: "/?q2=%7B+.foo+%3D+bar+%7D&q2=%7B+.foo+%3D+bar+%7D",
				URL: &url.URL{
					RawQuery: "q2=%7B+.foo+%3D+bar+%7D&q2=%7B+.foo+%3D+bar+%7D",
				},
				Header: map[string][]string{},
			},
		},
		// unescape error
		{
			event: events.ALBTargetGroupRequest{
				QueryStringParameters: map[string]string{
					"q": "%7B%20.foo%20%3D%20bar%20%7D%ZZ",
				},
			},
			wantErr: "failed to unescape query string parameter q: %7B%20.foo%20%3D%20bar%20%7D%ZZ: invalid URL escape \"%ZZ\"",
		},
	}

	for _, tt := range tests {
		got, err := httpRequest(tt.event)
		if len(tt.wantErr) > 0 {
			require.EqualError(t, err, tt.wantErr)
			continue
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, tt.want.Method, got.Method)
		require.Equal(t, tt.want.RequestURI, got.RequestURI)
		require.Equal(t, tt.want.URL, got.URL)
		require.Equal(t, tt.want.Header, got.Header)
	}
}
