// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsAggregationFunction An aggregation function
type LogsAggregationFunction string

// List of LogsAggregationFunction.
const (
	LOGSAGGREGATIONFUNCTION_COUNT         LogsAggregationFunction = "count"
	LOGSAGGREGATIONFUNCTION_CARDINALITY   LogsAggregationFunction = "cardinality"
	LOGSAGGREGATIONFUNCTION_PERCENTILE_75 LogsAggregationFunction = "pc75"
	LOGSAGGREGATIONFUNCTION_PERCENTILE_90 LogsAggregationFunction = "pc90"
	LOGSAGGREGATIONFUNCTION_PERCENTILE_95 LogsAggregationFunction = "pc95"
	LOGSAGGREGATIONFUNCTION_PERCENTILE_98 LogsAggregationFunction = "pc98"
	LOGSAGGREGATIONFUNCTION_PERCENTILE_99 LogsAggregationFunction = "pc99"
	LOGSAGGREGATIONFUNCTION_SUM           LogsAggregationFunction = "sum"
	LOGSAGGREGATIONFUNCTION_MIN           LogsAggregationFunction = "min"
	LOGSAGGREGATIONFUNCTION_MAX           LogsAggregationFunction = "max"
	LOGSAGGREGATIONFUNCTION_AVG           LogsAggregationFunction = "avg"
	LOGSAGGREGATIONFUNCTION_MEDIAN        LogsAggregationFunction = "median"
)

var allowedLogsAggregationFunctionEnumValues = []LogsAggregationFunction{
	LOGSAGGREGATIONFUNCTION_COUNT,
	LOGSAGGREGATIONFUNCTION_CARDINALITY,
	LOGSAGGREGATIONFUNCTION_PERCENTILE_75,
	LOGSAGGREGATIONFUNCTION_PERCENTILE_90,
	LOGSAGGREGATIONFUNCTION_PERCENTILE_95,
	LOGSAGGREGATIONFUNCTION_PERCENTILE_98,
	LOGSAGGREGATIONFUNCTION_PERCENTILE_99,
	LOGSAGGREGATIONFUNCTION_SUM,
	LOGSAGGREGATIONFUNCTION_MIN,
	LOGSAGGREGATIONFUNCTION_MAX,
	LOGSAGGREGATIONFUNCTION_AVG,
	LOGSAGGREGATIONFUNCTION_MEDIAN,
}

// GetAllowedValues reeturns the list of possible values.
func (v *LogsAggregationFunction) GetAllowedValues() []LogsAggregationFunction {
	return allowedLogsAggregationFunctionEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *LogsAggregationFunction) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = LogsAggregationFunction(value)
	return nil
}

// NewLogsAggregationFunctionFromValue returns a pointer to a valid LogsAggregationFunction
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewLogsAggregationFunctionFromValue(v string) (*LogsAggregationFunction, error) {
	ev := LogsAggregationFunction(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for LogsAggregationFunction: valid values are %v", v, allowedLogsAggregationFunctionEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v LogsAggregationFunction) IsValid() bool {
	for _, existing := range allowedLogsAggregationFunctionEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to LogsAggregationFunction value.
func (v LogsAggregationFunction) Ptr() *LogsAggregationFunction {
	return &v
}

// NullableLogsAggregationFunction handles when a null is used for LogsAggregationFunction.
type NullableLogsAggregationFunction struct {
	value *LogsAggregationFunction
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsAggregationFunction) Get() *LogsAggregationFunction {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsAggregationFunction) Set(val *LogsAggregationFunction) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsAggregationFunction) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableLogsAggregationFunction) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsAggregationFunction initializes the struct as if Set has been called.
func NewNullableLogsAggregationFunction(val *LogsAggregationFunction) *NullableLogsAggregationFunction {
	return &NullableLogsAggregationFunction{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsAggregationFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsAggregationFunction) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
