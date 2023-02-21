// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// HourlyUsageMeasurement Usage amount for a given usage type.
type HourlyUsageMeasurement struct {
	// Type of usage.
	UsageType *string `json:"usage_type,omitempty"`
	// Contains the number measured for the given usage_type during the hour.
	Value datadog.NullableInt64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageMeasurement instantiates a new HourlyUsageMeasurement object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageMeasurement() *HourlyUsageMeasurement {
	this := HourlyUsageMeasurement{}
	return &this
}

// NewHourlyUsageMeasurementWithDefaults instantiates a new HourlyUsageMeasurement object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageMeasurementWithDefaults() *HourlyUsageMeasurement {
	this := HourlyUsageMeasurement{}
	return &this
}

// GetUsageType returns the UsageType field value if set, zero value otherwise.
func (o *HourlyUsageMeasurement) GetUsageType() string {
	if o == nil || o.UsageType == nil {
		var ret string
		return ret
	}
	return *o.UsageType
}

// GetUsageTypeOk returns a tuple with the UsageType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageMeasurement) GetUsageTypeOk() (*string, bool) {
	if o == nil || o.UsageType == nil {
		return nil, false
	}
	return o.UsageType, true
}

// HasUsageType returns a boolean if a field has been set.
func (o *HourlyUsageMeasurement) HasUsageType() bool {
	return o != nil && o.UsageType != nil
}

// SetUsageType gets a reference to the given string and assigns it to the UsageType field.
func (o *HourlyUsageMeasurement) SetUsageType(v string) {
	o.UsageType = &v
}

// GetValue returns the Value field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *HourlyUsageMeasurement) GetValue() int64 {
	if o == nil || o.Value.Get() == nil {
		var ret int64
		return ret
	}
	return *o.Value.Get()
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *HourlyUsageMeasurement) GetValueOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Value.Get(), o.Value.IsSet()
}

// HasValue returns a boolean if a field has been set.
func (o *HourlyUsageMeasurement) HasValue() bool {
	return o != nil && o.Value.IsSet()
}

// SetValue gets a reference to the given datadog.NullableInt64 and assigns it to the Value field.
func (o *HourlyUsageMeasurement) SetValue(v int64) {
	o.Value.Set(&v)
}

// SetValueNil sets the value for Value to be an explicit nil.
func (o *HourlyUsageMeasurement) SetValueNil() {
	o.Value.Set(nil)
}

// UnsetValue ensures that no value is present for Value, not even an explicit nil.
func (o *HourlyUsageMeasurement) UnsetValue() {
	o.Value.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageMeasurement) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.UsageType != nil {
		toSerialize["usage_type"] = o.UsageType
	}
	if o.Value.IsSet() {
		toSerialize["value"] = o.Value.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageMeasurement) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		UsageType *string               `json:"usage_type,omitempty"`
		Value     datadog.NullableInt64 `json:"value,omitempty"`
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
	o.UsageType = all.UsageType
	o.Value = all.Value
	return nil
}
