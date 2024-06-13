package traceql

import "github.com/grafana/tempo/pkg/tempopb"

func MakeCollectTagValueFunc(collect func(tempopb.TagValue) bool) func(v Static) bool {
	return func(v Static) bool {
		var tv tempopb.TagValue

		switch v := v.(type) {
		case StaticString:
			tv.Type = "string"
			tv.Value = v.val // avoid formatting

		case StaticBool:
			tv.Type = "bool"
			tv.Value = v.String()

		case StaticInt:
			tv.Type = "int"
			tv.Value = v.String()

		case StaticFloat:
			tv.Type = "float"
			tv.Value = v.String()

		case StaticDuration:
			tv.Type = duration
			tv.Value = v.String()

		case StaticStatus:
			tv.Type = "keyword"
			tv.Value = v.String()
		}

		return collect(tv)
	}
}
