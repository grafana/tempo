// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
