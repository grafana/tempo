package main

import (
	"context"
	"net/http"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"

	"github.com/grafana/tempo/pkg/tempopb"
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
	err            error
	resp           http.Response
	traceResp      *tempopb.Trace
	requestsCount  int
	searchResponse []*tempopb.TraceSearchMetadata
	searchesCount  int
}

func (m MockHttpClient) DeleteOverrides(version string) error {
	panic("unimplemented")
}

func (m MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &m.resp, m.err
}

func (m MockHttpClient) GetOverrides() (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

func (m MockHttpClient) MetricsSummary(query string, groupBy string, start int64, end int64) (*tempopb.SpanMetricsSummaryResponse, error) {
	panic("unimplemented")
}

func (m MockHttpClient) PatchOverrides(limits *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

func (m *MockHttpClient) QueryTrace(id string) (*tempopb.Trace, error) {
	m.requestsCount++
	return m.traceResp, m.err
}

func (m MockHttpClient) Search(tags string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagValuesV2(key string, query string) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagValuesV2WithRange(tag string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTags() (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagsV2() (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagsV2WithRange(start int64, end int64) (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

func (m MockHttpClient) SearchTraceQL(query string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

func (m *MockHttpClient) SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error) {
	traceQlSearchResponse := &tempopb.SearchResponse{
		Traces: m.searchResponse,
	}
	m.searchesCount++
	return traceQlSearchResponse, m.err
}

func (m *MockHttpClient) SearchWithRange(tags string, start int64, end int64) (*tempopb.SearchResponse, error) {
	traceQlSearchResponse := &tempopb.SearchResponse{
		Traces: m.searchResponse,
	}

	m.searchesCount++
	return traceQlSearchResponse, m.err
}

func (m MockHttpClient) SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error) {
	panic("unimplemented")
}

func (m MockHttpClient) WithTransport(t http.RoundTripper) {
	panic("unimplemented")
}
