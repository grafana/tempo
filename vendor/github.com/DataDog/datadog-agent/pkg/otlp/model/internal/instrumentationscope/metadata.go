// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instrumentationscope

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/internal/utils"
)

const (
	instrumentationScopeTag        = "instrumentation_scope"
	instrumentationScopeVersionTag = "instrumentation_scope_version"
)

// TagsFromInstrumentationScopeMetadata takes the name and version of
// the instrumentation scope and converts them to Datadog tags.
func TagsFromInstrumentationScopeMetadata(il pcommon.InstrumentationScope) []string {
	return []string{
		utils.FormatKeyValueTag(instrumentationScopeTag, il.Name()),
		utils.FormatKeyValueTag(instrumentationScopeVersionTag, il.Version()),
	}
}
