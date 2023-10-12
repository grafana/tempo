// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

func SetValue(value pcommon.Value, val interface{}) error {
	var err error
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
			err = SetValue(pval, a)
		}
	case pcommon.Slice:
		v.CopyTo(value.SetEmptySlice())
	case pcommon.Map:
		v.CopyTo(value.SetEmptyMap())
	case map[string]interface{}:
		value.SetEmptyMap()
		for mk, mv := range v {
			err = SetMapValue(value.Map(), []ottl.Key{{String: &mk}}, mv)
		}
	}
	return err
}

func getIndexableValue(value pcommon.Value, keys []ottl.Key) (any, error) {
	val := value
	var ok bool
	for i := 0; i < len(keys); i++ {
		switch val.Type() {
		case pcommon.ValueTypeMap:
			if keys[i].String == nil {
				return nil, fmt.Errorf("map must be indexed by a string")
			}
			val, ok = val.Map().Get(*keys[i].String)
			if !ok {
				return nil, nil
			}
		case pcommon.ValueTypeSlice:
			if keys[i].Int == nil {
				return nil, fmt.Errorf("slice must be indexed by an int")
			}
			if int(*keys[i].Int) >= val.Slice().Len() || int(*keys[i].Int) < 0 {
				return nil, fmt.Errorf("index %v out of bounds", *keys[i].Int)
			}
			val = val.Slice().At(int(*keys[i].Int))
		default:
			return nil, fmt.Errorf("type %v does not support string indexing", val.Type())
		}
	}
	return ottlcommon.GetValue(val), nil
}

func setIndexableValue(currentValue pcommon.Value, val any, keys []ottl.Key) error {
	var newValue pcommon.Value
	switch val.(type) {
	case []string, []bool, []int64, []float64, [][]byte, []any:
		newValue = pcommon.NewValueSlice()
	default:
		newValue = pcommon.NewValueEmpty()
	}
	err := SetValue(newValue, val)
	if err != nil {
		return err
	}

	for i := 0; i < len(keys); i++ {
		switch currentValue.Type() {
		case pcommon.ValueTypeMap:
			if keys[i].String == nil {
				return errors.New("map must be indexed by a string")
			}
			potentialValue, ok := currentValue.Map().Get(*keys[i].String)
			if !ok {
				currentValue = currentValue.Map().PutEmpty(*keys[i].String)
			} else {
				currentValue = potentialValue
			}
		case pcommon.ValueTypeSlice:
			if keys[i].Int == nil {
				return errors.New("slice must be indexed by an int")
			}
			if int(*keys[i].Int) >= currentValue.Slice().Len() || int(*keys[i].Int) < 0 {
				return fmt.Errorf("index %v out of bounds", *keys[i].Int)
			}
			currentValue = currentValue.Slice().At(int(*keys[i].Int))
		case pcommon.ValueTypeEmpty:
			switch {
			case keys[i].String != nil:
				currentValue = currentValue.SetEmptyMap().PutEmpty(*keys[i].String)
			case keys[i].Int != nil:
				currentValue.SetEmptySlice()
				for k := 0; k < int(*keys[i].Int); k++ {
					currentValue.Slice().AppendEmpty()
				}
				currentValue = currentValue.Slice().AppendEmpty()
			default:
				return errors.New("neither a string nor an int index was given, this is an error in the OTTL")
			}
		default:
			return fmt.Errorf("type %v does not support string indexing", currentValue.Type())
		}
	}
	newValue.CopyTo(currentValue)
	return nil
}
