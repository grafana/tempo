// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricsAggregator The type of aggregation that can be performed on metrics queries.
type MetricsAggregator string

// List of MetricsAggregator.
const (
	METRICSAGGREGATOR_AVG        MetricsAggregator = "avg"
	METRICSAGGREGATOR_MIN        MetricsAggregator = "min"
	METRICSAGGREGATOR_MAX        MetricsAggregator = "max"
	METRICSAGGREGATOR_SUM        MetricsAggregator = "sum"
	METRICSAGGREGATOR_LAST       MetricsAggregator = "last"
	METRICSAGGREGATOR_PERCENTILE MetricsAggregator = "percentile"
	METRICSAGGREGATOR_MEAN       MetricsAggregator = "mean"
	METRICSAGGREGATOR_L2NORM     MetricsAggregator = "l2norm"
	METRICSAGGREGATOR_AREA       MetricsAggregator = "area"
)

var allowedMetricsAggregatorEnumValues = []MetricsAggregator{
	METRICSAGGREGATOR_AVG,
	METRICSAGGREGATOR_MIN,
	METRICSAGGREGATOR_MAX,
	METRICSAGGREGATOR_SUM,
	METRICSAGGREGATOR_LAST,
	METRICSAGGREGATOR_PERCENTILE,
	METRICSAGGREGATOR_MEAN,
	METRICSAGGREGATOR_L2NORM,
	METRICSAGGREGATOR_AREA,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricsAggregator) GetAllowedValues() []MetricsAggregator {
	return allowedMetricsAggregatorEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricsAggregator) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricsAggregator(value)
	return nil
}

// NewMetricsAggregatorFromValue returns a pointer to a valid MetricsAggregator
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricsAggregatorFromValue(v string) (*MetricsAggregator, error) {
	ev := MetricsAggregator(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricsAggregator: valid values are %v", v, allowedMetricsAggregatorEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricsAggregator) IsValid() bool {
	for _, existing := range allowedMetricsAggregatorEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricsAggregator value.
func (v MetricsAggregator) Ptr() *MetricsAggregator {
	return &v
}

// NullableMetricsAggregator handles when a null is used for MetricsAggregator.
type NullableMetricsAggregator struct {
	value *MetricsAggregator
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricsAggregator) Get() *MetricsAggregator {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricsAggregator) Set(val *MetricsAggregator) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricsAggregator) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricsAggregator) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricsAggregator initializes the struct as if Set has been called.
func NewNullableMetricsAggregator(val *MetricsAggregator) *NullableMetricsAggregator {
	return &NullableMetricsAggregator{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricsAggregator) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricsAggregator) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
