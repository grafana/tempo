package main

import (
	"context"
	"net/http"

	//nolint:all
	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/tempopb"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
)

type MockReporter struct {
	err            error
	batchesEmitted []*thrift.Batch
}

func (r MockReporter) EmitZipkinBatch(_ context.Context, _ []*zipkincore.Span) error {
	return r.err
}

func (r *MockReporter) EmitBatch(_ context.Context, b *thrift.Batch) error {
	r.batchesEmitted = append(r.batchesEmitted, b)
	return r.err
}

type MockHTTPClient struct {
	err  error
	resp http.Response
}

//nolint:all
func (m MockHTTPClient) DeleteOverrides(version string) error {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &m.resp, m.err
}

//nolint:all
func (m MockHTTPClient) GetOverrides() (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) MetricsSummary(query string, groupBy string, start int64, end int64) (*tempopb.SpanMetricsSummaryResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) PatchOverrides(limits *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) QueryTrace(id string) (*tempopb.Trace, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) Search(tags string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagValuesV2(key string, query string) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagValuesV2WithRange(tag string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTags() (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagsV2() (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagsV2WithRange(start int64, end int64) (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTraceQL(query string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error) {
	panic("unimplemented")
}

//nolint:all
func (m MockHTTPClient) WithTransport(t http.RoundTripper) {
	panic("unimplemented")
}
