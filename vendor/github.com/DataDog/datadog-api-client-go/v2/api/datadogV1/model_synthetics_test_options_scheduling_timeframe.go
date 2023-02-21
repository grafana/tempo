// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsTestOptionsSchedulingTimeframe Object describing a timeframe.
type SyntheticsTestOptionsSchedulingTimeframe struct {
	// Number representing the day of the week.
	Day *int32 `json:"day,omitempty"`
	// The hour of the day on which scheduling starts.
	From *string `json:"from,omitempty"`
	// The hour of the day on which scheduling ends.
	To *string `json:"to,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsTestOptionsSchedulingTimeframe instantiates a new SyntheticsTestOptionsSchedulingTimeframe object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsTestOptionsSchedulingTimeframe() *SyntheticsTestOptionsSchedulingTimeframe {
	this := SyntheticsTestOptionsSchedulingTimeframe{}
	return &this
}

// NewSyntheticsTestOptionsSchedulingTimeframeWithDefaults instantiates a new SyntheticsTestOptionsSchedulingTimeframe object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsTestOptionsSchedulingTimeframeWithDefaults() *SyntheticsTestOptionsSchedulingTimeframe {
	this := SyntheticsTestOptionsSchedulingTimeframe{}
	return &this
}

// GetDay returns the Day field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetDay() int32 {
	if o == nil || o.Day == nil {
		var ret int32
		return ret
	}
	return *o.Day
}

// GetDayOk returns a tuple with the Day field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetDayOk() (*int32, bool) {
	if o == nil || o.Day == nil {
		return nil, false
	}
	return o.Day, true
}

// HasDay returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) HasDay() bool {
	return o != nil && o.Day != nil
}

// SetDay gets a reference to the given int32 and assigns it to the Day field.
func (o *SyntheticsTestOptionsSchedulingTimeframe) SetDay(v int32) {
	o.Day = &v
}

// GetFrom returns the From field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetFrom() string {
	if o == nil || o.From == nil {
		var ret string
		return ret
	}
	return *o.From
}

// GetFromOk returns a tuple with the From field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetFromOk() (*string, bool) {
	if o == nil || o.From == nil {
		return nil, false
	}
	return o.From, true
}

// HasFrom returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) HasFrom() bool {
	return o != nil && o.From != nil
}

// SetFrom gets a reference to the given string and assigns it to the From field.
func (o *SyntheticsTestOptionsSchedulingTimeframe) SetFrom(v string) {
	o.From = &v
}

// GetTo returns the To field value if set, zero value otherwise.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetTo() string {
	if o == nil || o.To == nil {
		var ret string
		return ret
	}
	return *o.To
}

// GetToOk returns a tuple with the To field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) GetToOk() (*string, bool) {
	if o == nil || o.To == nil {
		return nil, false
	}
	return o.To, true
}

// HasTo returns a boolean if a field has been set.
func (o *SyntheticsTestOptionsSchedulingTimeframe) HasTo() bool {
	return o != nil && o.To != nil
}

// SetTo gets a reference to the given string and assigns it to the To field.
func (o *SyntheticsTestOptionsSchedulingTimeframe) SetTo(v string) {
	o.To = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsTestOptionsSchedulingTimeframe) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Day != nil {
		toSerialize["day"] = o.Day
	}
	if o.From != nil {
		toSerialize["from"] = o.From
	}
	if o.To != nil {
		toSerialize["to"] = o.To
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsTestOptionsSchedulingTimeframe) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Day  *int32  `json:"day,omitempty"`
		From *string `json:"from,omitempty"`
		To   *string `json:"to,omitempty"`
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
	o.Day = all.Day
	o.From = all.From
	o.To = all.To
	return nil
}
