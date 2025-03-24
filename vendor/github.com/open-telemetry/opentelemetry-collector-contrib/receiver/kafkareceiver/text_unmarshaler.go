// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"
import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"golang.org/x/text/encoding"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"
)

type textLogsUnmarshaler struct {
	decoder *encoding.Decoder
}

func newTextLogsUnmarshaler() LogsUnmarshalerWithEnc {
	return &textLogsUnmarshaler{}
}

func (r *textLogsUnmarshaler) Unmarshal(buf []byte) (plog.Logs, error) {
	if r.decoder == nil {
		return plog.Logs{}, errors.New("encoding not set")
	}
	p := plog.NewLogs()
	decoded, err := textutils.DecodeAsString(r.decoder, buf)
	if err != nil {
		return p, err
	}

	l := p.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	l.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	l.Body().SetStr(decoded)
	return p, nil
}

func (r *textLogsUnmarshaler) Encoding() string {
	return "text"
}

func (r *textLogsUnmarshaler) WithEnc(encodingName string) (LogsUnmarshalerWithEnc, error) {
	var err error
	enc, err := textutils.LookupEncoding(encodingName)
	if err != nil {
		return nil, err
	}
	return &textLogsUnmarshaler{
		decoder: enc.NewDecoder(),
	}, nil
}
