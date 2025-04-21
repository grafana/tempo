// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package unmarshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/unmarshaler"
import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"golang.org/x/text/encoding"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/textutils"
)

var _ plog.Unmarshaler = (*TextLogsUnmarshaler)(nil)

type TextLogsUnmarshaler struct {
	decoder *encoding.Decoder
}

func NewTextLogsUnmarshaler(encodingName string) (*TextLogsUnmarshaler, error) {
	encoding, err := textutils.LookupEncoding(encodingName)
	if err != nil {
		return nil, err
	}
	return &TextLogsUnmarshaler{decoder: encoding.NewDecoder()}, nil
}

func (r *TextLogsUnmarshaler) UnmarshalLogs(buf []byte) (plog.Logs, error) {
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
