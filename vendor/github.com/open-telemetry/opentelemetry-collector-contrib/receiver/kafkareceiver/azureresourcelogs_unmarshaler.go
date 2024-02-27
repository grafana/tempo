// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure"
)

type azureResourceLogsUnmarshaler struct {
	unmarshaler *azure.ResourceLogsUnmarshaler
}

func newAzureResourceLogsUnmarshaler(version string, logger *zap.Logger) LogsUnmarshaler {
	return azureResourceLogsUnmarshaler{
		unmarshaler: &azure.ResourceLogsUnmarshaler{
			Version: version,
			Logger:  logger,
		},
	}
}

func (r azureResourceLogsUnmarshaler) Unmarshal(buf []byte) (plog.Logs, error) {
	return r.unmarshaler.UnmarshalLogs(buf)
}

func (r azureResourceLogsUnmarshaler) Encoding() string {
	return "azure_resource_logs"
}
