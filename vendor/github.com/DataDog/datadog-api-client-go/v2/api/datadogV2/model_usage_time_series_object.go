// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// UsageTimeSeriesObject Usage timeseries data.
type UsageTimeSeriesObject struct {
	// Datetime in ISO-8601 format, UTC. The hour for the usage.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// Contains the number measured for the given usage_type during the hour.
	Value datadog.NullableInt64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsageTimeSeriesObject instantiates a new UsageTimeSeriesObject object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageTimeSeriesObject() *UsageTimeSeriesObject {
	this := UsageTimeSeriesObject{}
	return &this
}

// NewUsageTimeSeriesObjectWithDefaults instantiates a new UsageTimeSeriesObject object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageTimeSeriesObjectWithDefaults() *UsageTimeSeriesObject {
	this := UsageTimeSeriesObject{}
	return &this
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *UsageTimeSeriesObject) GetTimestamp() time.Time {
	if o == nil || o.Timestamp == nil {
		var ret time.Time
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageTimeSeriesObject) GetTimestampOk() (*time.Time, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *UsageTimeSeriesObject) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given time.Time and assigns it to the Timestamp field.
func (o *UsageTimeSeriesObject) SetTimestamp(v time.Time) {
	o.Timestamp = &v
}

// GetValue returns the Value field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageTimeSeriesObject) GetValue() int64 {
	if o == nil || o.Value.Get() == nil {
		var ret int64
		return ret
	}
	return *o.Value.Get()
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageTimeSeriesObject) GetValueOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Value.Get(), o.Value.IsSet()
}

// HasValue returns a boolean if a field has been set.
func (o *UsageTimeSeriesObject) HasValue() bool {
	return o != nil && o.Value.IsSet()
}

// SetValue gets a reference to the given datadog.NullableInt64 and assigns it to the Value field.
func (o *UsageTimeSeriesObject) SetValue(v int64) {
	o.Value.Set(&v)
}

// SetValueNil sets the value for Value to be an explicit nil.
func (o *UsageTimeSeriesObject) SetValueNil() {
	o.Value.Set(nil)
}

// UnsetValue ensures that no value is present for Value, not even an explicit nil.
func (o *UsageTimeSeriesObject) UnsetValue() {
	o.Value.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageTimeSeriesObject) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Timestamp != nil {
		if o.Timestamp.Nanosecond() == 0 {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
		}
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
func (o *UsageTimeSeriesObject) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Timestamp *time.Time            `json:"timestamp,omitempty"`
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
	o.Timestamp = all.Timestamp
	o.Value = all.Value
	return nil
}
