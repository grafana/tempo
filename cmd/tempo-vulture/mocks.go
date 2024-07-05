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
	batches_emited []*thrift.Batch
}

func (r MockReporter) EmitZipkinBatch(_ context.Context, _ []*zipkincore.Span) error {
	return r.err
}

func (r *MockReporter) EmitBatch(_ context.Context, b *thrift.Batch) error {
	r.batches_emited = append(r.batches_emited, b)
	return r.err
}

type MockHttpClient struct {
	err  error
	resp http.Response
}

// DeleteOverrides implements httpclient.IClient.
func (m MockHttpClient) DeleteOverrides(version string) error {
	panic("unimplemented")
}

// Do implements httpclient.IClient.
func (m MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &m.resp, m.err
}

// GetOverrides implements httpclient.IClient.
func (m MockHttpClient) GetOverrides() (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

// MetricsSummary implements httpclient.IClient.
func (m MockHttpClient) MetricsSummary(query string, groupBy string, start int64, end int64) (*tempopb.SpanMetricsSummaryResponse, error) {
	panic("unimplemented")
}

// PatchOverrides implements httpclient.IClient.
func (m MockHttpClient) PatchOverrides(limits *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

// QueryTrace implements httpclient.IClient.
func (m MockHttpClient) QueryTrace(id string) (*tempopb.Trace, error) {
	panic("unimplemented")
}

// Search implements httpclient.IClient.
func (m MockHttpClient) Search(tags string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

// SearchTagValues implements httpclient.IClient.
func (m MockHttpClient) SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error) {
	panic("unimplemented")
}

// SearchTagValuesV2 implements httpclient.IClient.
func (m MockHttpClient) SearchTagValuesV2(key string, query string) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

// SearchTagValuesV2WithRange implements httpclient.IClient.
func (m MockHttpClient) SearchTagValuesV2WithRange(tag string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

// SearchTags implements httpclient.IClient.
func (m MockHttpClient) SearchTags() (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

// SearchTagsV2 implements httpclient.IClient.
func (m MockHttpClient) SearchTagsV2() (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

// SearchTagsV2WithRange implements httpclient.IClient.
func (m MockHttpClient) SearchTagsV2WithRange(start int64, end int64) (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

// SearchTagsWithRange implements httpclient.IClient.
func (m MockHttpClient) SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

// SearchTraceQL implements httpclient.IClient.
func (m MockHttpClient) SearchTraceQL(query string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

// SearchTraceQLWithRange implements httpclient.IClient.
func (m MockHttpClient) SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

// SearchWithRange implements httpclient.IClient.
func (m MockHttpClient) SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

// SetOverrides implements httpclient.IClient.
func (m MockHttpClient) SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error) {
	panic("unimplemented")
}

// WithTransport implements httpclient.IClient.
func (m MockHttpClient) WithTransport(t http.RoundTripper) {
	panic("unimplemented")
}
