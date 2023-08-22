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

func GetValue(val pcommon.Value) interface{} {
	switch val.Type() {
	case pcommon.ValueTypeStr:
		return val.Str()
	case pcommon.ValueTypeBool:
		return val.Bool()
	case pcommon.ValueTypeInt:
		return val.Int()
	case pcommon.ValueTypeDouble:
		return val.Double()
	case pcommon.ValueTypeMap:
		return val.Map()
	case pcommon.ValueTypeSlice:
		return val.Slice()
	case pcommon.ValueTypeBytes:
		return val.Bytes().AsRaw()
	}
	return nil
}

func SetValue(value pcommon.Value, val interface{}) {
	switch v := val.(type) {
	case string:
		value.SetStr(v)
	case bool:
		value.SetBool(v)
	case int64:
		value.SetInt(v)
	case float64:
		value.SetDouble(v)
	case []byte:
		value.SetEmptyBytes().FromRaw(v)
	case []string:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, str := range v {
			value.Slice().AppendEmpty().SetStr(str)
		}
	case []bool:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, b := range v {
			value.Slice().AppendEmpty().SetBool(b)
		}
	case []int64:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, i := range v {
			value.Slice().AppendEmpty().SetInt(i)
		}
	case []float64:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, f := range v {
			value.Slice().AppendEmpty().SetDouble(f)
		}
	case [][]byte:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, b := range v {
			value.Slice().AppendEmpty().SetEmptyBytes().FromRaw(b)
		}
	case []any:
		value.SetEmptySlice().EnsureCapacity(len(v))
		for _, a := range v {
			pval := value.Slice().AppendEmpty()
			SetValue(pval, a)
		}
	case pcommon.Map:
		v.CopyTo(value.SetEmptyMap())
	case map[string]interface{}:
		value.SetEmptyMap()
		for mk, mv := range v {
			SetMapValue(value.Map(), mk, mv)
		}
	}
}
