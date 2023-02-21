// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MetricPoint A point object is of the form `{POSIX_timestamp, numeric_value}`.
type MetricPoint struct {
	// The timestamp should be in seconds and current.
	// Current is defined as not more than 10 minutes in the future or more than 1 hour in the past.
	Timestamp *int64 `json:"timestamp,omitempty"`
	// The numeric value format should be a 64bit float gauge-type value.
	Value *float64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricPoint instantiates a new MetricPoint object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricPoint() *MetricPoint {
	this := MetricPoint{}
	return &this
}

// NewMetricPointWithDefaults instantiates a new MetricPoint object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricPointWithDefaults() *MetricPoint {
	this := MetricPoint{}
	return &this
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *MetricPoint) GetTimestamp() int64 {
	if o == nil || o.Timestamp == nil {
		var ret int64
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricPoint) GetTimestampOk() (*int64, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *MetricPoint) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given int64 and assigns it to the Timestamp field.
func (o *MetricPoint) SetTimestamp(v int64) {
	o.Timestamp = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *MetricPoint) GetValue() float64 {
	if o == nil || o.Value == nil {
		var ret float64
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricPoint) GetValueOk() (*float64, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *MetricPoint) HasValue() bool {
	return o != nil && o.Value != nil
}

// SetValue gets a reference to the given float64 and assigns it to the Value field.
func (o *MetricPoint) SetValue(v float64) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricPoint) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Timestamp != nil {
		toSerialize["timestamp"] = o.Timestamp
	}
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricPoint) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Timestamp *int64   `json:"timestamp,omitempty"`
		Value     *float64 `json:"value,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Timestamp = all.Timestamp
	o.Value = all.Value
	return nil
}
