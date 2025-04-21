// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package unmarshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/unmarshaler"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

var _ plog.Unmarshaler = RawLogsUnmarshaler{}

type RawLogsUnmarshaler struct{}

func (r RawLogsUnmarshaler) UnmarshalLogs(buf []byte) (plog.Logs, error) {
	l := plog.NewLogs()
	l.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetEmptyBytes().FromRaw(buf)
	return l, nil
}
