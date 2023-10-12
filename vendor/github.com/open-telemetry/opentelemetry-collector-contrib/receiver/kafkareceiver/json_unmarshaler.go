// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"
import (
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type jsonLogsUnmarshaler struct {
}

func newJSONLogsUnmarshaler() LogsUnmarshaler {
	return &jsonLogsUnmarshaler{}
}

func (r *jsonLogsUnmarshaler) Unmarshal(buf []byte) (plog.Logs, error) {
	// create a new Logs struct to be populated with log data and returned
	p := plog.NewLogs()

	// get json logs from the buffer
	jsonVal := map[string]interface{}{}
	if err := jsoniter.Unmarshal(buf, &jsonVal); err != nil {
		return p, err
	}

	// create a new log record
	logRecords := p.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	logRecords.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Set the unmarshaled jsonVal as the body of the log record
	if err := logRecords.Body().SetEmptyMap().FromRaw(jsonVal); err != nil {
		return p, err
	}
	return p, nil
}

func (r *jsonLogsUnmarshaler) Encoding() string {
	return "json"
}
