// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CIAppAggregationFunction An aggregation function.
type CIAppAggregationFunction string

// List of CIAppAggregationFunction.
const (
	CIAPPAGGREGATIONFUNCTION_COUNT         CIAppAggregationFunction = "count"
	CIAPPAGGREGATIONFUNCTION_CARDINALITY   CIAppAggregationFunction = "cardinality"
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_75 CIAppAggregationFunction = "pc75"
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_90 CIAppAggregationFunction = "pc90"
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_95 CIAppAggregationFunction = "pc95"
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_98 CIAppAggregationFunction = "pc98"
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_99 CIAppAggregationFunction = "pc99"
	CIAPPAGGREGATIONFUNCTION_SUM           CIAppAggregationFunction = "sum"
	CIAPPAGGREGATIONFUNCTION_MIN           CIAppAggregationFunction = "min"
	CIAPPAGGREGATIONFUNCTION_MAX           CIAppAggregationFunction = "max"
	CIAPPAGGREGATIONFUNCTION_AVG           CIAppAggregationFunction = "avg"
	CIAPPAGGREGATIONFUNCTION_MEDIAN        CIAppAggregationFunction = "median"
	CIAPPAGGREGATIONFUNCTION_LATEST        CIAppAggregationFunction = "latest"
	CIAPPAGGREGATIONFUNCTION_EARLIEST      CIAppAggregationFunction = "earliest"
	CIAPPAGGREGATIONFUNCTION_MOST_FREQUENT CIAppAggregationFunction = "most_frequent"
	CIAPPAGGREGATIONFUNCTION_DELTA         CIAppAggregationFunction = "delta"
)

var allowedCIAppAggregationFunctionEnumValues = []CIAppAggregationFunction{
	CIAPPAGGREGATIONFUNCTION_COUNT,
	CIAPPAGGREGATIONFUNCTION_CARDINALITY,
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_75,
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_90,
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_95,
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_98,
	CIAPPAGGREGATIONFUNCTION_PERCENTILE_99,
	CIAPPAGGREGATIONFUNCTION_SUM,
	CIAPPAGGREGATIONFUNCTION_MIN,
	CIAPPAGGREGATIONFUNCTION_MAX,
	CIAPPAGGREGATIONFUNCTION_AVG,
	CIAPPAGGREGATIONFUNCTION_MEDIAN,
	CIAPPAGGREGATIONFUNCTION_LATEST,
	CIAPPAGGREGATIONFUNCTION_EARLIEST,
	CIAPPAGGREGATIONFUNCTION_MOST_FREQUENT,
	CIAPPAGGREGATIONFUNCTION_DELTA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *CIAppAggregationFunction) GetAllowedValues() []CIAppAggregationFunction {
	return allowedCIAppAggregationFunctionEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *CIAppAggregationFunction) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = CIAppAggregationFunction(value)
	return nil
}

// NewCIAppAggregationFunctionFromValue returns a pointer to a valid CIAppAggregationFunction
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewCIAppAggregationFunctionFromValue(v string) (*CIAppAggregationFunction, error) {
	ev := CIAppAggregationFunction(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for CIAppAggregationFunction: valid values are %v", v, allowedCIAppAggregationFunctionEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v CIAppAggregationFunction) IsValid() bool {
	for _, existing := range allowedCIAppAggregationFunctionEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to CIAppAggregationFunction value.
func (v CIAppAggregationFunction) Ptr() *CIAppAggregationFunction {
	return &v
}

// NullableCIAppAggregationFunction handles when a null is used for CIAppAggregationFunction.
type NullableCIAppAggregationFunction struct {
	value *CIAppAggregationFunction
	isSet bool
}

// Get returns the associated value.
func (v NullableCIAppAggregationFunction) Get() *CIAppAggregationFunction {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableCIAppAggregationFunction) Set(val *CIAppAggregationFunction) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableCIAppAggregationFunction) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableCIAppAggregationFunction) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableCIAppAggregationFunction initializes the struct as if Set has been called.
func NewNullableCIAppAggregationFunction(val *CIAppAggregationFunction) *NullableCIAppAggregationFunction {
	return &NullableCIAppAggregationFunction{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableCIAppAggregationFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableCIAppAggregationFunction) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
