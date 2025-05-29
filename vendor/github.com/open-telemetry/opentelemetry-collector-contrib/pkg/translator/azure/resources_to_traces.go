// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azure // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure"

import (
	"bytes"
	"encoding/hex"
	"net/url"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/otel/semconv/v1.13.0"
	"go.uber.org/zap"
)

const (
	// Constants for OpenTelemetry Specs
	traceAzureResourceID = "azure.resource.id"
)

type azureTracesRecords struct {
	Records []azureTracesRecord `json:"records"`
}

// Azure Trace Records based on Azure AppRequests & AppDependencies table data
// the common record schema reference:
// https://learn.microsoft.com/en-us/azure/azure-monitor/reference/tables/apprequests
// https://learn.microsoft.com/en-us/azure/azure-monitor/reference/tables/appdependencies
type azureTracesRecord struct {
	Time                  string             `json:"time"`
	ResourceID            string             `json:"resourceId"`
	ResourceGUID          string             `json:"ResourceGUID"`
	Type                  string             `json:"Type"`
	AppRoleInstance       string             `json:"AppRoleInstance"`
	AppRoleName           string             `json:"AppRoleName"`
	AppVersion            string             `json:"AppVersion"`
	ClientCity            string             `json:"ClientCity"`
	ClientCountryOrRegion string             `json:"ClientCountryOrRegion"`
	ClientIP              string             `json:"ClientIP"`
	ClientStateOrProvince string             `json:"ClientStateOrProvince"`
	ClientType            string             `json:"ClientType"`
	IKey                  string             `json:"IKey"`
	OperationName         string             `json:"OperationName"`
	OperationID           string             `json:"OperationId"`
	ParentID              string             `json:"ParentId"`
	SDKVersion            string             `json:"SDKVersion"`
	Properties            map[string]string  `json:"Properties"`
	Measurements          map[string]float64 `json:"Measurements"`
	SpanID                string             `json:"Id"`
	Name                  string             `json:"Name"`
	URL                   string             `json:"Url"`
	Source                string             `json:"Source"`
	Success               bool               `json:"Success"`
	ResultCode            string             `json:"ResultCode"`
	DurationMs            float64            `json:"DurationMs"`
	PerformanceBucket     string             `json:"PerformanceBucket"`
	ItemCount             float64            `json:"ItemCount"`
}

var _ ptrace.Unmarshaler = (*TracesUnmarshaler)(nil)

type TracesUnmarshaler struct {
	Version     string
	Logger      *zap.Logger
	TimeFormats []string
}

func (r TracesUnmarshaler) UnmarshalTraces(buf []byte) (ptrace.Traces, error) {
	t := ptrace.NewTraces()

	var azureTraces azureTracesRecords
	decoder := jsoniter.NewDecoder(bytes.NewReader(buf))
	err := decoder.Decode(&azureTraces)
	if err != nil {
		return t, err
	}

	resourceTraces := t.ResourceSpans().AppendEmpty()
	resource := resourceTraces.Resource()
	resource.Attributes().PutStr(string(conventions.TelemetrySDKNameKey), scopeName)
	resource.Attributes().PutStr(string(conventions.TelemetrySDKLanguageKey), conventions.TelemetrySDKLanguageGo.Value.AsString())
	resource.Attributes().PutStr(string(conventions.TelemetrySDKVersionKey), r.Version)
	resource.Attributes().PutStr(string(conventions.CloudProviderKey), conventions.CloudProviderAzure.Value.AsString())

	scopeSpans := resourceTraces.ScopeSpans().AppendEmpty()

	spans := scopeSpans.Spans()

	resourceID := ""
	for _, azureTrace := range azureTraces.Records {
		if resourceID == "" && azureTrace.ResourceID != "" {
			resourceID = azureTrace.ResourceID
		}

		resource.Attributes().PutStr("service.name", azureTrace.AppRoleName)

		nanos, err := asTimestamp(azureTrace.Time, r.TimeFormats...)
		if err != nil {
			r.Logger.Warn("Invalid Timestamp", zap.String("time", azureTrace.Time))
			continue
		}

		traceID, traceErr := TraceIDFromHex(azureTrace.OperationID)
		if traceErr != nil {
			r.Logger.Warn("Invalid TraceID", zap.String("traceID", azureTrace.OperationID))
			return t, err
		}
		spanID, spanErr := SpanIDFromHex(azureTrace.SpanID)
		if spanErr != nil {
			r.Logger.Warn("Invalid SpanID", zap.String("spanID", azureTrace.SpanID))
			return t, err
		}
		parentID, parentErr := SpanIDFromHex(azureTrace.ParentID)
		if parentErr != nil {
			r.Logger.Warn("Invalid ParentID", zap.String("parentID", azureTrace.ParentID))
			return t, err
		}

		span := spans.AppendEmpty()
		span.SetTraceID(traceID)
		span.SetSpanID(spanID)
		span.SetParentSpanID(parentID)

		span.Attributes().PutStr("OperationName", azureTrace.OperationName)
		span.Attributes().PutStr("AppRoleName", azureTrace.AppRoleName)
		span.Attributes().PutStr("AppRoleInstance", azureTrace.AppRoleInstance)
		span.Attributes().PutStr("Type", azureTrace.Type)

		span.Attributes().PutStr("http.url", azureTrace.URL)

		urlObj, _ := url.Parse(azureTrace.URL)
		hostname := urlObj.Host
		hostpath := urlObj.Path
		scheme := urlObj.Scheme

		span.Attributes().PutStr("http.host", hostname)
		span.Attributes().PutStr("http.path", hostpath)
		span.Attributes().PutStr("http.response.status_code", azureTrace.ResultCode)
		span.Attributes().PutStr("http.client_ip", azureTrace.ClientIP)
		span.Attributes().PutStr("http.client_city", azureTrace.ClientCity)
		span.Attributes().PutStr("http.client_type", azureTrace.ClientType)
		span.Attributes().PutStr("http.client_state", azureTrace.ClientStateOrProvince)
		span.Attributes().PutStr("http.client_type", azureTrace.ClientType)
		span.Attributes().PutStr("http.client_country", azureTrace.ClientCountryOrRegion)
		span.Attributes().PutStr("http.scheme", scheme)
		span.Attributes().PutStr("http.method", azureTrace.Properties["HTTP Method"])

		for key, value := range azureTrace.Properties {
			if key != "HTTP Method" { // HTTP Method is already mapped to http.method
				span.Attributes().PutStr(key, value)
			}
		}

		span.SetKind(ptrace.SpanKindServer)
		span.SetName(azureTrace.Name)
		span.SetStartTimestamp(nanos)
		span.SetEndTimestamp(nanos + pcommon.Timestamp(azureTrace.DurationMs*1e6))
	}

	if resourceID != "" {
		resourceTraces.Resource().Attributes().PutStr(traceAzureResourceID, resourceID)
	} else {
		r.Logger.Warn("No ResourceID Set on Traces!")
	}

	return t, nil
}

func TraceIDFromHex(hexStr string) (pcommon.TraceID, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return pcommon.TraceID{}, err
	}
	var id pcommon.TraceID
	copy(id[:], bytes)
	return id, nil
}

func SpanIDFromHex(hexStr string) (pcommon.SpanID, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return pcommon.SpanID{}, err
	}
	var id pcommon.SpanID
	copy(id[:], bytes)
	return id, nil
}
