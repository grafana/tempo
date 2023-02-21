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

package attributes

import (
	"fmt"

	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
)

type systemAttributes struct {
	OSType string
}

func (sattrs *systemAttributes) extractTags() []string {
	tags := make([]string, 0, 1)

	// Add OS type, eg. WINDOWS, LINUX, etc.
	if sattrs.OSType != "" {
		tags = append(tags, fmt.Sprintf("%s:%s", conventions.AttributeOSType, sattrs.OSType))
	}

	return tags
}
