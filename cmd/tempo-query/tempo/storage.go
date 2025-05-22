package tempo

import (
	"errors"
	"fmt"

	"github.com/grafana/tempo/pkg/jaegerpb/storage/v2"
)

var (
	errCannotConvertValue = errors.New("cannot convert value to string")
)

func valueToString(value *storage.AnyValue) (string, error) {
	switch v := value.Value.(type) {
	case *storage.AnyValue_StringValue:
		return v.StringValue, nil
	case *storage.AnyValue_BoolValue:
		return fmt.Sprintf("%s", v.BoolValue), nil
	case *storage.AnyValue_DoubleValue:
		return fmt.Sprintf("%f", v.DoubleValue), nil
	case *storage.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue), nil
	default:
		return "", errCannotConvertValue
	}
}
