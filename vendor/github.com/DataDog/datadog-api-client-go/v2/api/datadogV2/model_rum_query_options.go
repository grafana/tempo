// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RUMQueryOptions Global query options that are used during the query.
// Note: Only supply timezone or time offset, not both. Otherwise, the query fails.
type RUMQueryOptions struct {
	// The time offset (in seconds) to apply to the query.
	TimeOffset *int64 `json:"time_offset,omitempty"`
	// The timezone can be specified as GMT, UTC, an offset from UTC (like UTC+1), or as a Timezone Database identifier (like America/New_York).
	Timezone *string `json:"timezone,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRUMQueryOptions instantiates a new RUMQueryOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRUMQueryOptions() *RUMQueryOptions {
	this := RUMQueryOptions{}
	var timezone string = "UTC"
	this.Timezone = &timezone
	return &this
}

// NewRUMQueryOptionsWithDefaults instantiates a new RUMQueryOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRUMQueryOptionsWithDefaults() *RUMQueryOptions {
	this := RUMQueryOptions{}
	var timezone string = "UTC"
	this.Timezone = &timezone
	return &this
}

// GetTimeOffset returns the TimeOffset field value if set, zero value otherwise.
func (o *RUMQueryOptions) GetTimeOffset() int64 {
	if o == nil || o.TimeOffset == nil {
		var ret int64
		return ret
	}
	return *o.TimeOffset
}

// GetTimeOffsetOk returns a tuple with the TimeOffset field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMQueryOptions) GetTimeOffsetOk() (*int64, bool) {
	if o == nil || o.TimeOffset == nil {
		return nil, false
	}
	return o.TimeOffset, true
}

// HasTimeOffset returns a boolean if a field has been set.
func (o *RUMQueryOptions) HasTimeOffset() bool {
	return o != nil && o.TimeOffset != nil
}

// SetTimeOffset gets a reference to the given int64 and assigns it to the TimeOffset field.
func (o *RUMQueryOptions) SetTimeOffset(v int64) {
	o.TimeOffset = &v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *RUMQueryOptions) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RUMQueryOptions) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *RUMQueryOptions) HasTimezone() bool {
	return o != nil && o.Timezone != nil
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *RUMQueryOptions) SetTimezone(v string) {
	o.Timezone = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RUMQueryOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.TimeOffset != nil {
		toSerialize["time_offset"] = o.TimeOffset
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
func (o *RUMQueryOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		TimeOffset *int64  `json:"time_offset,omitempty"`
		Timezone   *string `json:"timezone,omitempty"`
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
	o.TimeOffset = all.TimeOffset
	o.Timezone = all.Timezone
	return nil
}
