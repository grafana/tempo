package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	userconfigurableoverrides "github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	thrift "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"

	"github.com/grafana/tempo/pkg/tempopb"
)

type MockReporter struct {
	err            error
	batchesEmitted []*thrift.Batch
	// We need the lock to control concurrent accesses to batchesEmitted
	m sync.Mutex
}

func (r *MockReporter) EmitBatch(_ context.Context, b *thrift.Batch) error {
	if r.err == nil {
		r.m.Lock()
		defer r.m.Unlock()
		r.batchesEmitted = append(r.batchesEmitted, b)
	}

	return r.err
}

func (r *MockReporter) GetEmittedBatches() []*thrift.Batch {
	r.m.Lock()
	defer r.m.Unlock()
	return r.batchesEmitted
}

type MockHTTPClient struct {
	err                 error
	resp                http.Response
	traceResp           *tempopb.Trace
	requestsCount       int
	searchResponse      []*tempopb.TraceSearchMetadata
	searchesCount       int
	metricsResp         *tempopb.QueryRangeResponse
	metricsCount        int
	metricsInstantResp  *tempopb.QueryInstantResponse
	metricsInstantCount int
	// We need the lock to control concurrent accesses to shared variables in the tests
	m sync.Mutex
}

//nolint:all
func (m *MockHTTPClient) DeleteOverrides(_ string) error {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &m.resp, m.err
}

//nolint:all
func (m *MockHTTPClient) GetOverrides() (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) MetricsQueryRange(_ string, _, _ int64, _ string, _ int) (*tempopb.QueryRangeResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.m.Lock()
	defer m.m.Unlock()
	m.metricsCount++
	return m.metricsResp, nil
}

//nolint:all
func (m *MockHTTPClient) MetricsQueryInstant(_ string, _, _ int64, _ int) (*tempopb.QueryInstantResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.m.Lock()
	defer m.m.Unlock()
	m.metricsInstantCount++
	return m.metricsInstantResp, nil
}

//nolint:all
func (m *MockHTTPClient) PatchOverrides(_ *userconfigurableoverrides.Limits) (*userconfigurableoverrides.Limits, string, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) QueryTrace(_ string) (*tempopb.Trace, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.m.Lock()
	defer m.m.Unlock()
	m.requestsCount++
	return m.traceResp, m.err
}

//nolint:all
func (m *MockHTTPClient) QueryTraceWithRange(_ context.Context, _ string, _ int64, _ int64) (*tempopb.Trace, error) {
	m.m.Lock()
	defer m.m.Unlock()
	m.requestsCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.traceResp, nil
}

func (m *MockHTTPClient) GetRequestsCount() int {
	m.m.Lock()
	defer m.m.Unlock()
	return m.requestsCount
}

//nolint:all
func (m *MockHTTPClient) Search(tags string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagValues(key string) (*tempopb.SearchTagValuesResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagValuesV2(key string, query string) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagValuesV2WithRange(tag, query string, start int64, end int64) (*tempopb.SearchTagValuesV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTags() (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagsV2() (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagsV2WithRange(scope, query string, start int64, end int64) (*tempopb.SearchTagsV2Response, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTagsWithRange(start int64, end int64) (*tempopb.SearchTagsResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTraceQL(query string) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTraceQLWithRangeAndLimit(query string, start int64, end int64, limit int64, spss int64) (*tempopb.SearchResponse, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) SearchTraceQLWithRange(query string, start int64, end int64) (*tempopb.SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.m.Lock()
	defer m.m.Unlock()
	traceQlSearchResponse := &tempopb.SearchResponse{
		Traces: m.searchResponse,
	}
	m.searchesCount++
	return traceQlSearchResponse, m.err
}

//nolint:all
func (m *MockHTTPClient) SearchWithRange(_ context.Context, _ string, _ int64, _ int64) (*tempopb.SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.m.Lock()
	defer m.m.Unlock()
	traceQlSearchResponse := &tempopb.SearchResponse{
		Traces: m.searchResponse,
	}

	m.searchesCount++
	return traceQlSearchResponse, m.err
}

func (m *MockHTTPClient) GetSearchesCount() int {
	m.m.Lock()
	defer m.m.Unlock()
	return m.searchesCount
}

//nolint:all
func (m *MockHTTPClient) SetOverrides(limits *userconfigurableoverrides.Limits, version string) (string, error) {
	panic("unimplemented")
}

//nolint:all
func (m *MockHTTPClient) WithTransport(t http.RoundTripper) {
	panic("unimplemented")
}

func (m *MockHTTPClient) GetMetricsCount() int {
	m.m.Lock()
	defer m.m.Unlock()
	return m.metricsCount
}

type MockClock struct {
	now    time.Time
	sleeps []time.Duration
}

func (m *MockClock) Now() time.Time {
	return m.now
}

func (m *MockClock) Sleep(d time.Duration) {
	m.sleeps = append(m.sleeps, d)
}
