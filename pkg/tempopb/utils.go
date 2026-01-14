package tempopb

import v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"

func MakeKeyValueString(key, value string) v1.KeyValue {
	return v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: value,
			},
		},
	}
}

func MakeKeyValueStringPtr(key, value string) *v1.KeyValue {
	kv := MakeKeyValueString(key, value)
	return &kv
}

func MakeKeyValueDouble(key string, value float64) v1.KeyValue {
	return v1.KeyValue{
		Key: key,
		Value: &v1.AnyValue{
			Value: &v1.AnyValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
}

// gogo/protobuf has poor support for optional field
// TODO: remove these functions after migration from gogo/protobuf

func (q *QueryRangeRequest) HasInstant() bool {
	if q.XInstant == nil {
		return false
	}

	val, ok := q.GetXInstant().(*QueryRangeRequest_Instant)
	return ok && val != nil
}

func (q *QueryRangeRequest) SetInstant(value bool) {
	q.XInstant = &QueryRangeRequest_Instant{
		Instant: value,
	}
}
