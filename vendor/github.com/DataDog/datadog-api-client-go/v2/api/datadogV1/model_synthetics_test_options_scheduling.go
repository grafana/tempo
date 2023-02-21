// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsTestOptionsScheduling Object containing timeframes and timezone used for advanced scheduling.
type SyntheticsTestOptionsScheduling struct {
	// Array containing objects describing the scheduling pattern to apply to each day.
	Timeframes []SyntheticsTestOptionsSchedulingTimeframe `json:"timeframes,omitempty"`
	// Timezone in which the timeframe is based.
	Timezone *string `json:"timezone,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestOptionsScheduling instantiates a new SyntheticsTestOptionsScheduling object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestOptionsScheduling() *SyntheticsTestOptionsScheduling {
	this := SyntheticsTestOptionsScheduling{}
	return &this
}

// NewSyntheticsTestOptionsSchedulingWithDefaults instantiates a new SyntheticsTestOptionsScheduling object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestOptionsSchedulingWithDefaults() *SyntheticsTestOptionsScheduling {
	this := SyntheticsTestOptionsScheduling{}
	return &this
}

// GetTimeframes returns the Timeframes field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsScheduling) GetTimeframes() []SyntheticsTestOptionsSchedulingTimeframe {
	if o == nil || o.Timeframes == nil {
		var ret []SyntheticsTestOptionsSchedulingTimeframe
		return ret
	}
	return o.Timeframes
}

// GetTimeframesOk returns a tuple with the Timeframes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsScheduling) GetTimeframesOk() (*[]SyntheticsTestOptionsSchedulingTimeframe, bool) {
	if o == nil || o.Timeframes == nil {
		return nil, false
	}
	return &o.Timeframes, true
}

// HasTimeframes returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsScheduling) HasTimeframes() bool {
	return o != nil && o.Timeframes != nil
}

// SetTimeframes gets a reference to the given []SyntheticsTestOptionsSchedulingTimeframe and assigns it to the Timeframes field.
func (o *SyntheticsTestOptionsScheduling) SetTimeframes(v []SyntheticsTestOptionsSchedulingTimeframe) {
	o.Timeframes = v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsScheduling) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsScheduling) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsScheduling) HasTimezone() bool {
	return o != nil && o.Timezone != nil
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *SyntheticsTestOptionsScheduling) SetTimezone(v string) {
	o.Timezone = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestOptionsScheduling) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Timeframes != nil {
		toSerialize["timeframes"] = o.Timeframes
	}
	if o.Timezone != nil {
		toSerialize["timezone"] = o.Timezone
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestOptionsScheduling) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Timeframes []SyntheticsTestOptionsSchedulingTimeframe `json:"timeframes,omitempty"`
		Timezone   *string                                    `json:"timezone,omitempty"`
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
	o.Timeframes = all.Timeframes
	o.Timezone = all.Timezone
	return nil
}
