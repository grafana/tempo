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

package exporterhelper

import (
	"errors"
)

var (
	// errNilConfig is returned when an empty name is given.
	errNilConfig = errors.New("nil config")
	// errNilLogger is returned when a logger is nil
	errNilLogger = errors.New("nil logger")
	// errNilPushTraceData is returned when a nil PushTraces is given.
	errNilPushTraceData = errors.New("nil PushTraces")
	// errNilPushMetricsData is returned when a nil PushMetrics is given.
	errNilPushMetricsData = errors.New("nil PushMetrics")
	// errNilPushLogsData is returned when a nil PushLogs is given.
	errNilPushLogsData = errors.New("nil PushLogs")
)
