// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricVolumes - Possible response objects for a metric's volume.
type MetricVolumes struct {
	MetricDistinctVolume        *MetricDistinctVolume
	MetricIngestedIndexedVolume *MetricIngestedIndexedVolume

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// MetricDistinctVolumeAsMetricVolumes is a convenience function that returns MetricDistinctVolume wrapped in MetricVolumes.
func MetricDistinctVolumeAsMetricVolumes(v *MetricDistinctVolume) MetricVolumes {
	return MetricVolumes{MetricDistinctVolume: v}
}

// MetricIngestedIndexedVolumeAsMetricVolumes is a convenience function that returns MetricIngestedIndexedVolume wrapped in MetricVolumes.
func MetricIngestedIndexedVolumeAsMetricVolumes(v *MetricIngestedIndexedVolume) MetricVolumes {
	return MetricVolumes{MetricIngestedIndexedVolume: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *MetricVolumes) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into MetricDistinctVolume
	err = json.Unmarshal(data, &obj.MetricDistinctVolume)
	if err == nil {
		if obj.MetricDistinctVolume != nil && obj.MetricDistinctVolume.UnparsedObject == nil {
			jsonMetricDistinctVolume, _ := json.Marshal(obj.MetricDistinctVolume)
			if string(jsonMetricDistinctVolume) == "{}" { // empty struct
				obj.MetricDistinctVolume = nil
			} else {
				match++
			}
		} else {
			obj.MetricDistinctVolume = nil
		}
	} else {
		obj.MetricDistinctVolume = nil
	}

	// try to unmarshal data into MetricIngestedIndexedVolume
	err = json.Unmarshal(data, &obj.MetricIngestedIndexedVolume)
	if err == nil {
		if obj.MetricIngestedIndexedVolume != nil && obj.MetricIngestedIndexedVolume.UnparsedObject == nil {
			jsonMetricIngestedIndexedVolume, _ := json.Marshal(obj.MetricIngestedIndexedVolume)
			if string(jsonMetricIngestedIndexedVolume) == "{}" { // empty struct
				obj.MetricIngestedIndexedVolume = nil
			} else {
				match++
			}
		} else {
			obj.MetricIngestedIndexedVolume = nil
		}
	} else {
		obj.MetricIngestedIndexedVolume = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.MetricDistinctVolume = nil
		obj.MetricIngestedIndexedVolume = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj MetricVolumes) MarshalJSON() ([]byte, error) {
	if obj.MetricDistinctVolume != nil {
		return json.Marshal(&obj.MetricDistinctVolume)
	}

	if obj.MetricIngestedIndexedVolume != nil {
		return json.Marshal(&obj.MetricIngestedIndexedVolume)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *MetricVolumes) GetActualInstance() interface{} {
	if obj.MetricDistinctVolume != nil {
		return obj.MetricDistinctVolume
	}

	if obj.MetricIngestedIndexedVolume != nil {
		return obj.MetricIngestedIndexedVolume
	}

	// all schemas are nil
	return nil
}

// NullableMetricVolumes handles when a null is used for MetricVolumes.
type NullableMetricVolumes struct {
	value *MetricVolumes
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricVolumes) Get() *MetricVolumes {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricVolumes) Set(val *MetricVolumes) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricVolumes) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMetricVolumes) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricVolumes initializes the struct as if Set has been called.
func NewNullableMetricVolumes(val *MetricVolumes) *NullableMetricVolumes {
	return &NullableMetricVolumes{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricVolumes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricVolumes) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
