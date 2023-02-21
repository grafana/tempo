// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricsDataSource A data source that is powered by the Metrics platform.
type MetricsDataSource string

// List of MetricsDataSource.
const (
	METRICSDATASOURCE_METRICS    MetricsDataSource = "metrics"
	METRICSDATASOURCE_CLOUD_COST MetricsDataSource = "cloud_cost"
)

var allowedMetricsDataSourceEnumValues = []MetricsDataSource{
	METRICSDATASOURCE_METRICS,
	METRICSDATASOURCE_CLOUD_COST,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MetricsDataSource) GetAllowedValues() []MetricsDataSource {
	return allowedMetricsDataSourceEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MetricsDataSource) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MetricsDataSource(value)
	return nil
}

// NewMetricsDataSourceFromValue returns a pointer to a valid MetricsDataSource
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMetricsDataSourceFromValue(v string) (*MetricsDataSource, error) {
	ev := MetricsDataSource(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MetricsDataSource: valid values are %v", v, allowedMetricsDataSourceEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MetricsDataSource) IsValid() bool {
	for _, existing := range allowedMetricsDataSourceEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MetricsDataSource value.
func (v MetricsDataSource) Ptr() *MetricsDataSource {
	return &v
}

// NullableMetricsDataSource handles when a null is used for MetricsDataSource.
type NullableMetricsDataSource struct {
	value *MetricsDataSource
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricsDataSource) Get() *MetricsDataSource {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricsDataSource) Set(val *MetricsDataSource) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricsDataSource) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMetricsDataSource) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricsDataSource initializes the struct as if Set has been called.
func NewNullableMetricsDataSource(val *MetricsDataSource) *NullableMetricsDataSource {
	return &NullableMetricsDataSource{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricsDataSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricsDataSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
