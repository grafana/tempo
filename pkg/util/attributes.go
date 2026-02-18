package util

import (
	"strconv"
	"strings"

	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
)

func StringifyAnyValue(anyValue *v1common.AnyValue) string {
	switch anyValue.Value.(type) {
	case *v1common.AnyValue_BoolValue:
		return strconv.FormatBool(anyValue.GetBoolValue())
	case *v1common.AnyValue_IntValue:
		return strconv.FormatInt(anyValue.GetIntValue(), 10)
	case *v1common.AnyValue_ArrayValue:
		var arrStr strings.Builder
		arrStr.WriteString("[")
		for _, v := range anyValue.GetArrayValue().Values {
			arrStr.WriteString(StringifyAnyValue(v))
		}
		arrStr.WriteString("]")
		return arrStr.String()
	case *v1common.AnyValue_DoubleValue:
		return strconv.FormatFloat(anyValue.GetDoubleValue(), 'f', -1, 64)
	case *v1common.AnyValue_KvlistValue:
		var mapStr strings.Builder
		mapStr.WriteString("{")
		for _, kv := range anyValue.GetKvlistValue().Values {
			mapStr.WriteString(kv.Key + ":" + StringifyAnyValue(kv.Value))
		}
		mapStr.WriteString("}")
		return mapStr.String()
	case *v1common.AnyValue_StringValue:
		return anyValue.GetStringValue()
	}

	return ""
}
