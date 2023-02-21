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

package instrumentationlibrary

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/internal/utils"
)

const (
	instrumentationLibraryTag        = "instrumentation_library"
	instrumentationLibraryVersionTag = "instrumentation_library_version"
)

// TagsFromInstrumentationLibraryMetadata takes the name and version of
// the instrumentation library and converts them to Datadog tags.
func TagsFromInstrumentationLibraryMetadata(il pcommon.InstrumentationScope) []string {
	return []string{
		utils.FormatKeyValueTag(instrumentationLibraryTag, il.Name()),
		utils.FormatKeyValueTag(instrumentationLibraryVersionTag, il.Version()),
	}
}
