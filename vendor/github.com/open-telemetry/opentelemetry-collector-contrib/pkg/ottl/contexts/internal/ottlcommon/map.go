// Copyright The OpenTelemetry Authors
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

package ottlcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ottlcommon"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func GetMapValue(attrs pcommon.Map, mapKey string) interface{} {
	val, ok := attrs.Get(mapKey)
	if !ok {
		return nil
	}
	return GetValue(val)
}

func SetMapValue(attrs pcommon.Map, mapKey string, val interface{}) {
	var value pcommon.Value
	switch val.(type) {
	case []string, []bool, []int64, []float64, [][]byte, []any:
		value = pcommon.NewValueSlice()
	default:
		value = pcommon.NewValueEmpty()
	}

	SetValue(value, val)
	value.CopyTo(attrs.PutEmpty(mapKey))
}
