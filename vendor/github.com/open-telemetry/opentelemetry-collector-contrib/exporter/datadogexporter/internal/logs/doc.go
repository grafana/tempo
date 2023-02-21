// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logs provides utils for transforming OTLP LogRecord to Datadog format
// it also provides sender for submitting transformed logs to datadog backend
// This uses datadog-api-client-go for submitting logs
package logs // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/logs"
