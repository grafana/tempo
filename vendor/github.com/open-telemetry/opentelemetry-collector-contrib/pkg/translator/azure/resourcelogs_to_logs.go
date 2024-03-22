// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azure // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure"

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/relvacode/iso8601"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	conventions "go.opentelemetry.io/collector/semconv/v1.13.0"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

const (
	// Constants for OpenTelemetry Specs
	scopeName = "otelcol/azureresourcelogs"

	// Constants for Azure Log Records
	azureCategory          = "azure.category"
	azureCorrelationID     = "azure.correlation.id"
	azureDuration          = "azure.duration"
	azureIdentity          = "azure.identity"
	azureOperationName     = "azure.operation.name"
	azureOperationVersion  = "azure.operation.version"
	azureProperties        = "azure.properties"
	azureResourceID        = "azure.resource.id"
	azureResultType        = "azure.result.type"
	azureResultSignature   = "azure.result.signature"
	azureResultDescription = "azure.result.description"
	azureTenantID          = "azure.tenant.id"
)

var (
	errMissingTimestamp = errors.New("missing timestamp")
)

// azureRecords represents an array of Azure log records
// as exported via an Azure Event Hub
type azureRecords struct {
	Records []azureLogRecord `json:"records"`
}

// azureLogRecord represents a single Azure log following
// the common schema:
// https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/resource-logs-schema
type azureLogRecord struct {
	Time              string       `json:"time"`
	Timestamp         string       `json:"timeStamp"`
	ResourceID        string       `json:"resourceId"`
	TenantID          *string      `json:"tenantId"`
	OperationName     string       `json:"operationName"`
	OperationVersion  *string      `json:"operationVersion"`
	Category          string       `json:"category"`
	ResultType        *string      `json:"resultType"`
	ResultSignature   *string      `json:"resultSignature"`
	ResultDescription *string      `json:"resultDescription"`
	DurationMs        *json.Number `json:"durationMs"`
	CallerIPAddress   *string      `json:"callerIpAddress"`
	CorrelationID     *string      `json:"correlationId"`
	Identity          *any         `json:"identity"`
	Level             *json.Number `json:"Level"`
	Location          *string      `json:"location"`
	Properties        *any         `json:"properties"`
}

var _ plog.Unmarshaler = (*ResourceLogsUnmarshaler)(nil)

type ResourceLogsUnmarshaler struct {
	Version string
	Logger  *zap.Logger
}

func (r ResourceLogsUnmarshaler) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	l := plog.NewLogs()

	var azureLogs azureRecords
	decoder := jsoniter.NewDecoder(bytes.NewReader(buf))
	if err := decoder.Decode(&azureLogs); err != nil {
		return l, err
	}

	var resourceIds []string
	azureResourceLogs := make(map[string][]azureLogRecord)
	for _, azureLog := range azureLogs.Records {
		azureResourceLogs[azureLog.ResourceID] = append(azureResourceLogs[azureLog.ResourceID], azureLog)
		keyExists := slices.Contains(resourceIds, azureLog.ResourceID)
		if !keyExists {
			resourceIds = append(resourceIds, azureLog.ResourceID)
		}
	}

	for _, resourceID := range resourceIds {
		logs := azureResourceLogs[resourceID]
		resourceLogs := l.ResourceLogs().AppendEmpty()
		resourceLogs.Resource().Attributes().PutStr(azureResourceID, resourceID)
		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		scopeLogs.Scope().SetName(scopeName)
		scopeLogs.Scope().SetVersion(r.Version)
		logRecords := scopeLogs.LogRecords()

		for i := 0; i < len(logs); i++ {
			log := logs[i]
			nanos, err := getTimestamp(log)
			if err != nil {
				r.Logger.Warn("Unable to convert timestamp from log", zap.String("timestamp", log.Time))
				continue
			}

			lr := logRecords.AppendEmpty()
			lr.SetTimestamp(nanos)

			if log.Level != nil {
				severity := asSeverity(*log.Level)
				lr.SetSeverityNumber(severity)
				lr.SetSeverityText(log.Level.String())
			}

			if err := lr.Attributes().FromRaw(extractRawAttributes(log)); err != nil {
				return l, err
			}
		}
	}

	return l, nil
}

func getTimestamp(record azureLogRecord) (pcommon.Timestamp, error) {
	if record.Time != "" {
		return asTimestamp(record.Time)
	} else if record.Timestamp != "" {
		return asTimestamp(record.Timestamp)
	}

	return 0, errMissingTimestamp
}

// asTimestamp will parse an ISO8601 string into an OpenTelemetry
// nanosecond timestamp. If the string cannot be parsed, it will
// return zero and the error.
func asTimestamp(s string) (pcommon.Timestamp, error) {
	t, err := iso8601.ParseString(s)
	if err != nil {
		return 0, err
	}

	return pcommon.Timestamp(t.UnixNano()), nil
}

// asSeverity converts the Azure log level to equivalent
// OpenTelemetry severity numbers. If the log level is not
// valid, then the 'Unspecified' value is returned.
func asSeverity(number json.Number) plog.SeverityNumber {
	switch number.String() {
	case "Informational":
		return plog.SeverityNumberInfo
	case "Warning":
		return plog.SeverityNumberWarn
	case "Error":
		return plog.SeverityNumberError
	case "Critical":
		return plog.SeverityNumberFatal
	default:
		var levelNumber, _ = number.Int64()
		if levelNumber > 0 {
			return plog.SeverityNumber(levelNumber)
		}

		return plog.SeverityNumberUnspecified
	}
}

func extractRawAttributes(log azureLogRecord) map[string]any {
	var attrs = map[string]any{}

	attrs[azureCategory] = log.Category
	setIf(attrs, azureCorrelationID, log.CorrelationID)
	if log.DurationMs != nil {
		duration, err := strconv.ParseInt(log.DurationMs.String(), 10, 64)
		if err == nil {
			attrs[azureDuration] = duration
		}
	}
	if log.Identity != nil {
		attrs[azureIdentity] = *log.Identity
	}
	attrs[azureOperationName] = log.OperationName
	setIf(attrs, azureOperationVersion, log.OperationVersion)
	if log.Properties != nil {
		attrs[azureProperties] = *log.Properties
	}
	setIf(attrs, azureResultDescription, log.ResultDescription)
	setIf(attrs, azureResultSignature, log.ResultSignature)
	setIf(attrs, azureResultType, log.ResultType)
	setIf(attrs, azureTenantID, log.TenantID)

	setIf(attrs, conventions.AttributeCloudRegion, log.Location)
	attrs[conventions.AttributeCloudProvider] = conventions.AttributeCloudProviderAzure

	setIf(attrs, conventions.AttributeNetSockPeerAddr, log.CallerIPAddress)
	return attrs
}

func setIf(attrs map[string]any, key string, value *string) {
	if value != nil && *value != "" {
		attrs[key] = *value
	}
}
