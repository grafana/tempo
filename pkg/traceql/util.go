package traceql

import "github.com/grafana/tempo/pkg/tempopb"

func MakeCollectTagValueFunc(collect func(tempopb.TagValue) bool) func(v Static) bool {
	return func(v Static) bool {
		tv := tempopb.TagValue{}

		switch v.Type {
		case TypeString:
			tv.Type = "string"
			tv.Value = v.S // avoid formatting

		case TypeBoolean:
			tv.Type = "bool"
			tv.Value = v.String()

		case TypeInt:
			tv.Type = "int"
			tv.Value = v.String()

		case TypeFloat:
			tv.Type = "float"
			tv.Value = v.String()

		case TypeDuration:
			tv.Type = duration
			tv.Value = v.String()

		case TypeStatus:
			tv.Type = "keyword"
			tv.Value = v.String()
		}

		return collect(tv)
	}
}
