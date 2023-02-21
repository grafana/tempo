// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalsListResponseLinks Links attributes.
type SecurityMonitoringSignalsListResponseLinks struct {
	// The link for the next set of results. **Note**: The request can also be made using the
	// POST endpoint.
	Next *string `json:"next,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalsListResponseLinks instantiates a new SecurityMonitoringSignalsListResponseLinks object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalsListResponseLinks() *SecurityMonitoringSignalsListResponseLinks {
	this := SecurityMonitoringSignalsListResponseLinks{}
	return &this
}

// NewSecurityMonitoringSignalsListResponseLinksWithDefaults instantiates a new SecurityMonitoringSignalsListResponseLinks object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalsListResponseLinksWithDefaults() *SecurityMonitoringSignalsListResponseLinks {
	this := SecurityMonitoringSignalsListResponseLinks{}
	return &this
}

// GetNext returns the Next field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalsListResponseLinks) GetNext() string {
	if o == nil || o.Next == nil {
		var ret string
		return ret
	}
	return *o.Next
}

// GetNextOk returns a tuple with the Next field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalsListResponseLinks) GetNextOk() (*string, bool) {
	if o == nil || o.Next == nil {
		return nil, false
	}
	return o.Next, true
}

// HasNext returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalsListResponseLinks) HasNext() bool {
	return o != nil && o.Next != nil
}

// SetNext gets a reference to the given string and assigns it to the Next field.
func (o *SecurityMonitoringSignalsListResponseLinks) SetNext(v string) {
	o.Next = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalsListResponseLinks) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Next != nil {
		toSerialize["next"] = o.Next
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalsListResponseLinks) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Next *string `json:"next,omitempty"`
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
	o.Next = all.Next
	return nil
}
