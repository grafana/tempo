// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricsAndMetricTagConfigurations - Object for a metrics and metric tag configurations.
type MetricsAndMetricTagConfigurations struct {
	Metric                 *Metric
	MetricTagConfiguration *MetricTagConfiguration

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MetricAsMetricsAndMetricTagConfigurations is a convenience function that returns Metric wrapped in MetricsAndMetricTagConfigurations.
func MetricAsMetricsAndMetricTagConfigurations(v *Metric) MetricsAndMetricTagConfigurations {
	return MetricsAndMetricTagConfigurations{Metric: v}
}

// MetricTagConfigurationAsMetricsAndMetricTagConfigurations is a convenience function that returns MetricTagConfiguration wrapped in MetricsAndMetricTagConfigurations.
func MetricTagConfigurationAsMetricsAndMetricTagConfigurations(v *MetricTagConfiguration) MetricsAndMetricTagConfigurations {
	return MetricsAndMetricTagConfigurations{MetricTagConfiguration: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *MetricsAndMetricTagConfigurations) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into Metric
	err = json.Unmarshal(data, &obj.Metric)
	if err == nil {
		if obj.Metric != nil && obj.Metric.UnparsedObject == nil {
			jsonMetric, _ := json.Marshal(obj.Metric)
			if string(jsonMetric) == "{}" { // empty struct
				obj.Metric = nil
			} else {
				match++
			}
		} else {
			obj.Metric = nil
		}
	} else {
		obj.Metric = nil
	}

	// try to unmarshal data into MetricTagConfiguration
	err = json.Unmarshal(data, &obj.MetricTagConfiguration)
	if err == nil {
		if obj.MetricTagConfiguration != nil && obj.MetricTagConfiguration.UnparsedObject == nil {
			jsonMetricTagConfiguration, _ := json.Marshal(obj.MetricTagConfiguration)
			if string(jsonMetricTagConfiguration) == "{}" { // empty struct
				obj.MetricTagConfiguration = nil
			} else {
				match++
			}
		} else {
			obj.MetricTagConfiguration = nil
		}
	} else {
		obj.MetricTagConfiguration = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.Metric = nil
		obj.MetricTagConfiguration = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj MetricsAndMetricTagConfigurations) MarshalJSON() ([]byte, error) {
	if obj.Metric != nil {
		return json.Marshal(&obj.Metric)
	}

	if obj.MetricTagConfiguration != nil {
		return json.Marshal(&obj.MetricTagConfiguration)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *MetricsAndMetricTagConfigurations) GetActualInstance() interface{} {
	if obj.Metric != nil {
		return obj.Metric
	}

	if obj.MetricTagConfiguration != nil {
		return obj.MetricTagConfiguration
	}

	// all schemas are nil
	return nil
}

// NullableMetricsAndMetricTagConfigurations handles when a null is used for MetricsAndMetricTagConfigurations.
type NullableMetricsAndMetricTagConfigurations struct {
	value *MetricsAndMetricTagConfigurations
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricsAndMetricTagConfigurations) Get() *MetricsAndMetricTagConfigurations {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricsAndMetricTagConfigurations) Set(val *MetricsAndMetricTagConfigurations) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricsAndMetricTagConfigurations) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMetricsAndMetricTagConfigurations) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricsAndMetricTagConfigurations initializes the struct as if Set has been called.
func NewNullableMetricsAndMetricTagConfigurations(val *MetricsAndMetricTagConfigurations) *NullableMetricsAndMetricTagConfigurations {
	return &NullableMetricsAndMetricTagConfigurations{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricsAndMetricTagConfigurations) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricsAndMetricTagConfigurations) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
