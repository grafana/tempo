// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// SecurityMonitoringSignalAttributes The object containing all signal attributes and their
// associated values.
type SecurityMonitoringSignalAttributes struct {
	// A JSON object of attributes in the security signal.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	// The message in the security signal defined by the rule that generated the signal.
	Message *string `json:"message,omitempty"`
	// An array of tags associated with the security signal.
	Tags []string `json:"tags,omitempty"`
	// The timestamp of the security signal.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalAttributes instantiates a new SecurityMonitoringSignalAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalAttributes() *SecurityMonitoringSignalAttributes {
	this := SecurityMonitoringSignalAttributes{}
	return &this
}

// NewSecurityMonitoringSignalAttributesWithDefaults instantiates a new SecurityMonitoringSignalAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalAttributesWithDefaults() *SecurityMonitoringSignalAttributes {
	this := SecurityMonitoringSignalAttributes{}
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalAttributes) GetAttributes() map[string]interface{} {
	if o == nil || o.Attributes == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAttributes) GetAttributesOk() (*map[string]interface{}, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalAttributes) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given map[string]interface{} and assigns it to the Attributes field.
func (o *SecurityMonitoringSignalAttributes) SetAttributes(v map[string]interface{}) {
	o.Attributes = v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalAttributes) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAttributes) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalAttributes) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SecurityMonitoringSignalAttributes) SetMessage(v string) {
	o.Message = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SecurityMonitoringSignalAttributes) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalAttributes) GetTimestamp() time.Time {
	if o == nil || o.Timestamp == nil {
		var ret time.Time
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalAttributes) GetTimestampOk() (*time.Time, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given time.Time and assigns it to the Timestamp field.
func (o *SecurityMonitoringSignalAttributes) SetTimestamp(v time.Time) {
	o.Timestamp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Timestamp != nil {
		if o.Timestamp.Nanosecond() == 0 {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes map[string]interface{} `json:"attributes,omitempty"`
		Message    *string                `json:"message,omitempty"`
		Tags       []string               `json:"tags,omitempty"`
		Timestamp  *time.Time             `json:"timestamp,omitempty"`
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
	o.Attributes = all.Attributes
	o.Message = all.Message
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	return nil
}
