// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

type rawLogsUnmarshaler struct{}

func newRawLogsUnmarshaler() LogsUnmarshaler {
	return rawLogsUnmarshaler{}
}

func (r rawLogsUnmarshaler) Unmarshal(buf []byte) (plog.Logs, error) {
	l := plog.NewLogs()
	l.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetEmptyBytes().FromRaw(buf)
	return l, nil
}

func (r rawLogsUnmarshaler) Encoding() string {
	return "raw"
}
