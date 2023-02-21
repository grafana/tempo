// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// CIAppAggregateBucketValueTimeseriesPoint A timeseries point.
type CIAppAggregateBucketValueTimeseriesPoint struct {
	// The time value for this point.
	Time *time.Time `json:"time,omitempty"`
	// The value for this point.
	Value *float64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppAggregateBucketValueTimeseriesPoint instantiates a new CIAppAggregateBucketValueTimeseriesPoint object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppAggregateBucketValueTimeseriesPoint() *CIAppAggregateBucketValueTimeseriesPoint {
	this := CIAppAggregateBucketValueTimeseriesPoint{}
	return &this
}

// NewCIAppAggregateBucketValueTimeseriesPointWithDefaults instantiates a new CIAppAggregateBucketValueTimeseriesPoint object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppAggregateBucketValueTimeseriesPointWithDefaults() *CIAppAggregateBucketValueTimeseriesPoint {
	this := CIAppAggregateBucketValueTimeseriesPoint{}
	return &this
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *CIAppAggregateBucketValueTimeseriesPoint) GetTime() time.Time {
	if o == nil || o.Time == nil {
		var ret time.Time
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppAggregateBucketValueTimeseriesPoint) GetTimeOk() (*time.Time, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *CIAppAggregateBucketValueTimeseriesPoint) HasTime() bool {
	return o != nil && o.Time != nil
}

// SetTime gets a reference to the given time.Time and assigns it to the Time field.
func (o *CIAppAggregateBucketValueTimeseriesPoint) SetTime(v time.Time) {
	o.Time = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *CIAppAggregateBucketValueTimeseriesPoint) GetValue() float64 {
	if o == nil || o.Value == nil {
		var ret float64
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppAggregateBucketValueTimeseriesPoint) GetValueOk() (*float64, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *CIAppAggregateBucketValueTimeseriesPoint) HasValue() bool {
	return o != nil && o.Value != nil
}

// SetValue gets a reference to the given float64 and assigns it to the Value field.
func (o *CIAppAggregateBucketValueTimeseriesPoint) SetValue(v float64) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppAggregateBucketValueTimeseriesPoint) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Time != nil {
		if o.Time.Nanosecond() == 0 {
			toSerialize["time"] = o.Time.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["time"] = o.Time.Format("2006-01-02T15:04:05.000Z07:00")
		}
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
func (o *CIAppAggregateBucketValueTimeseriesPoint) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Time  *time.Time `json:"time,omitempty"`
		Value *float64   `json:"value,omitempty"`
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
	o.Time = all.Time
	o.Value = all.Value
	return nil
}
