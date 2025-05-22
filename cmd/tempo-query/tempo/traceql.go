package tempo

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"

	storage_v2 "github.com/grafana/tempo/pkg/jaegerpb/storage/v2"
	"github.com/grafana/tempo/pkg/traceql"
)

func stringToStatus(s string) traceql.Status {
	s = strings.ToLower(s)
	switch s {
	case "error":
		return traceql.StatusError
	case "ok":
		return traceql.StatusUnset
	}
	return traceql.StatusUnset
}

func BuildTQL(attrs map[string]*storage_v2.AnyValue) ([]traceql.Condition, error) {
	var conditions []traceql.Condition
	for k, v := range attrs {
		id, err := traceql.ParseIdentifier(k)
		if err != nil {
			return nil, fmt.Errorf("failed to parse identifier: %w", err)
		}
		if k == "status" {
			// status is a special case, we need to convert it to a number
			conditions = append(conditions, traceql.Condition{
				Attribute: id,
				Op:        traceql.OpEqual,
				Operands:  traceql.Operands{traceql.NewStaticStatus(stringToStatus(v.GetStringValue()))},
			})
		} else {
			conditions = append(conditions, traceql.Condition{
				Attribute: id,
				Op:        traceql.OpEqual,
				Operands:  traceql.Operands{toStatic(v)},
				CallBack:  nil,
			})
		}
	}
	return conditions, nil
}

func toStatic(v *storage_v2.AnyValue) traceql.Static {
	switch v.Value.(type) {
	case *storage_v2.AnyValue_StringValue:
		// Among the string values, there's still the weird case where Jaeger sends the string "400" instead of the
		// number 400. Therefore, we check here if the user provided a number, and in that case we parse it as a float / int
		i, err := strconv.ParseInt(v.GetStringValue(), 10, 32)
		if err == nil {
			return traceql.NewStaticInt(int(i))
		}
		d, err := strconv.ParseFloat(v.GetStringValue(), 64)
		if err == nil {
			return traceql.NewStaticFloat(d)
		}
		return traceql.NewStaticString(v.GetStringValue())
	case *storage_v2.AnyValue_BoolValue:
		return traceql.NewStaticBool(v.GetBoolValue())
	case *storage_v2.AnyValue_DoubleValue:
		return traceql.NewStaticFloat(v.GetDoubleValue())
	case *storage_v2.AnyValue_IntValue:
		return traceql.NewStaticInt(int(v.GetIntValue())) // TODO: Check if casting is OK
	}
	return traceql.NewStaticString("") // TODO: Handle other types
}

func operandsToString(operands traceql.Operands) string {
	if len(operands) != 1 {
		panic("invalid number of operands") // TODO: Find a clever way to handle this
		return ""
	}
	return operands[0].String()
}

func ConditionToString(condition traceql.Condition) string {
	return condition.Attribute.String() + condition.Op.String() + operandsToString(condition.Operands)
}

func ConditionsToString(tql []traceql.Condition, durationMin *types.Duration, durationMax *types.Duration) string {
	var conditions []string
	for _, condition := range tql {
		conditions = append(conditions, ConditionToString(condition))
	}
	if durationMin != nil && !durationMin.Equal(types.Duration{}) {
		minDurationD := time.Duration(durationMin.Seconds*1e9 + int64(durationMin.Nanos))
		conditions = append(conditions, fmt.Sprintf("trace:duration >= %s", minDurationD.String()))
	}
	if durationMax != nil && !durationMax.Equal(types.Duration{}) {
		maxDurationD := time.Duration(durationMax.Seconds*1e9 + int64(durationMax.Nanos))
		conditions = append(conditions, fmt.Sprintf("trace:duration <= %s", maxDurationD.String()))
	}
	return strings.Join(conditions, " && ")
}
