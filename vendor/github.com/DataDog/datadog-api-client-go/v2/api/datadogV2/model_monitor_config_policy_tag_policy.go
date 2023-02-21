// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// MonitorConfigPolicyTagPolicy Tag attributes of a monitor configuration policy.
type MonitorConfigPolicyTagPolicy struct {
	// The key of the tag.
	TagKey *string `json:"tag_key,omitempty"`
	// If a tag key is required for monitor creation.
	TagKeyRequired *bool `json:"tag_key_required,omitempty"`
	// Valid values for the tag.
	ValidTagValues []string `json:"valid_tag_values,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorConfigPolicyTagPolicy instantiates a new MonitorConfigPolicyTagPolicy object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorConfigPolicyTagPolicy() *MonitorConfigPolicyTagPolicy {
	this := MonitorConfigPolicyTagPolicy{}
	return &this
}

// NewMonitorConfigPolicyTagPolicyWithDefaults instantiates a new MonitorConfigPolicyTagPolicy object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorConfigPolicyTagPolicyWithDefaults() *MonitorConfigPolicyTagPolicy {
	this := MonitorConfigPolicyTagPolicy{}
	return &this
}

// GetTagKey returns the TagKey field value if set, zero value otherwise.
func (o *MonitorConfigPolicyTagPolicy) GetTagKey() string {
	if o == nil || o.TagKey == nil {
		var ret string
		return ret
	}
	return *o.TagKey
}

// GetTagKeyOk returns a tuple with the TagKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicy) GetTagKeyOk() (*string, bool) {
	if o == nil || o.TagKey == nil {
		return nil, false
	}
	return o.TagKey, true
}

// HasTagKey returns a boolean if a field has been set.
func (o *MonitorConfigPolicyTagPolicy) HasTagKey() bool {
	return o != nil && o.TagKey != nil
}

// SetTagKey gets a reference to the given string and assigns it to the TagKey field.
func (o *MonitorConfigPolicyTagPolicy) SetTagKey(v string) {
	o.TagKey = &v
}

// GetTagKeyRequired returns the TagKeyRequired field value if set, zero value otherwise.
func (o *MonitorConfigPolicyTagPolicy) GetTagKeyRequired() bool {
	if o == nil || o.TagKeyRequired == nil {
		var ret bool
		return ret
	}
	return *o.TagKeyRequired
}

// GetTagKeyRequiredOk returns a tuple with the TagKeyRequired field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicy) GetTagKeyRequiredOk() (*bool, bool) {
	if o == nil || o.TagKeyRequired == nil {
		return nil, false
	}
	return o.TagKeyRequired, true
}

// HasTagKeyRequired returns a boolean if a field has been set.
func (o *MonitorConfigPolicyTagPolicy) HasTagKeyRequired() bool {
	return o != nil && o.TagKeyRequired != nil
}

// SetTagKeyRequired gets a reference to the given bool and assigns it to the TagKeyRequired field.
func (o *MonitorConfigPolicyTagPolicy) SetTagKeyRequired(v bool) {
	o.TagKeyRequired = &v
}

// GetValidTagValues returns the ValidTagValues field value if set, zero value otherwise.
func (o *MonitorConfigPolicyTagPolicy) GetValidTagValues() []string {
	if o == nil || o.ValidTagValues == nil {
		var ret []string
		return ret
	}
	return o.ValidTagValues
}

// GetValidTagValuesOk returns a tuple with the ValidTagValues field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicy) GetValidTagValuesOk() (*[]string, bool) {
	if o == nil || o.ValidTagValues == nil {
		return nil, false
	}
	return &o.ValidTagValues, true
}

// HasValidTagValues returns a boolean if a field has been set.
func (o *MonitorConfigPolicyTagPolicy) HasValidTagValues() bool {
	return o != nil && o.ValidTagValues != nil
}

// SetValidTagValues gets a reference to the given []string and assigns it to the ValidTagValues field.
func (o *MonitorConfigPolicyTagPolicy) SetValidTagValues(v []string) {
	o.ValidTagValues = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorConfigPolicyTagPolicy) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.TagKey != nil {
		toSerialize["tag_key"] = o.TagKey
	}
	if o.TagKeyRequired != nil {
		toSerialize["tag_key_required"] = o.TagKeyRequired
	}
	if o.ValidTagValues != nil {
		toSerialize["valid_tag_values"] = o.ValidTagValues
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorConfigPolicyTagPolicy) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		TagKey         *string  `json:"tag_key,omitempty"`
		TagKeyRequired *bool    `json:"tag_key_required,omitempty"`
		ValidTagValues []string `json:"valid_tag_values,omitempty"`
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
	o.TagKey = all.TagKey
	o.TagKeyRequired = all.TagKeyRequired
	o.ValidTagValues = all.ValidTagValues
	return nil
}
