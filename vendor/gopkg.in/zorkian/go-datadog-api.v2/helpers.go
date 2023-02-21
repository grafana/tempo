/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2017 by authors and contributors.
 */

package datadog

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool { return &v }

// GetBool is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetBool(v *bool) (bool, bool) {
	if v != nil {
		return *v, true
	}

	return false, false
}

// Int is a helper routine that allocates a new int value
// to store v and returns a pointer to it.
func Int(v int) *int { return &v }

// Int64 is a helper routine that allocates a new int64 value to
// store v and return a pointer to it.
func Int64(v int64) *int64 { return &v }

// GetIntOk is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetIntOk(v *int) (int, bool) {
	if v != nil {
		return *v, true
	}

	return 0, false
}

// Float64 is a helper routine that allocates a new float64 value
// to store v and returns a pointer to it.
func Float64(v float64) *float64 { return &v }

// GetFloat64Ok is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetFloat64Ok(v *float64) (float64, bool) {
	if v != nil {
		return *v, true
	}

	return 0, false
}

// Float64AlmostEqual will return true if two floats are within a certain tolerance of each other
func Float64AlmostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string { return &v }

// GetStringOk is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetStringOk(v *string) (string, bool) {
	if v != nil {
		return *v, true
	}

	return "", false
}

// JsonNumber is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func JsonNumber(v json.Number) *json.Number { return &v }

// GetJsonNumberOk is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetJsonNumberOk(v *json.Number) (json.Number, bool) {
	if v != nil {
		return *v, true
	}

	return "", false
}

// Precision is a helper routine that allocates a new precision value
// to store v and returns a pointer to it.
func Precision(v PrecisionT) *PrecisionT { return &v }

// GetPrecision is a helper routine that returns a boolean representing
// if a value was set, and if so, dereferences the pointer to it.
func GetPrecision(v *PrecisionT) (PrecisionT, bool) {
	if v != nil {
		return *v, true
	}

	return PrecisionT(""), false
}

// GetStringId is a helper routine that allows screenboards and timeboards to be retrieved
// by either the legacy numerical format or the new string format.
// It returns the id as is if it is a string, converts it to a string if it is an integer.
// It return an error if the type is neither string or an integer
func GetStringId(id interface{}) (string, error) {
	switch v := id.(type) {
	case int:
		return strconv.Itoa(v), nil
	case string:
		return v, nil
	default:
		return "", errors.New("unsupported id type")
	}
}

func GetFloatFromInterface(intf *interface{}) (*float64, bool, error) {
	var result *float64
	var auto bool

	if intf != nil {
		val := *intf
		switch tp := val.(type) {
		case float32:
			fv := float64(val.(float32))
			result = &fv
		case float64:
			fv := val.(float64)
			result = &fv
		case int:
			fv := float64(val.(int))
			result = &fv
		case int32:
			fv := float64(val.(int32))
			result = &fv
		case int64:
			fv := float64(val.(int64))
			result = &fv
		case string:
			fv := val.(string)
			if fv == "auto" {
				auto = true
			} else {
				f, err := strconv.ParseFloat(fv, 64)
				if err != nil {
					return nil, false, err
				}
				result = &f
			}
		default:
			return nil, false, fmt.Errorf(`bad type "%v" for Yaxis.min, expected "auto" or a number`, tp)
		}
	}
	return result, auto, nil
}
