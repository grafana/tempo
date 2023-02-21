// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MonitorConfigPolicyTagPolicyCreateRequest Tag attributes of a monitor configuration policy.
type MonitorConfigPolicyTagPolicyCreateRequest struct {
	// The key of the tag.
	TagKey string `json:"tag_key"`
	// If a tag key is required for monitor creation.
	TagKeyRequired bool `json:"tag_key_required"`
	// Valid values for the tag.
	ValidTagValues []string `json:"valid_tag_values"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorConfigPolicyTagPolicyCreateRequest instantiates a new MonitorConfigPolicyTagPolicyCreateRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorConfigPolicyTagPolicyCreateRequest(tagKey string, tagKeyRequired bool, validTagValues []string) *MonitorConfigPolicyTagPolicyCreateRequest {
	this := MonitorConfigPolicyTagPolicyCreateRequest{}
	this.TagKey = tagKey
	this.TagKeyRequired = tagKeyRequired
	this.ValidTagValues = validTagValues
	return &this
}

// NewMonitorConfigPolicyTagPolicyCreateRequestWithDefaults instantiates a new MonitorConfigPolicyTagPolicyCreateRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorConfigPolicyTagPolicyCreateRequestWithDefaults() *MonitorConfigPolicyTagPolicyCreateRequest {
	this := MonitorConfigPolicyTagPolicyCreateRequest{}
	return &this
}

// GetTagKey returns the TagKey field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetTagKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.TagKey
}

// GetTagKeyOk returns a tuple with the TagKey field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetTagKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.TagKey, true
}

// SetTagKey sets field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) SetTagKey(v string) {
	o.TagKey = v
}

// GetTagKeyRequired returns the TagKeyRequired field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetTagKeyRequired() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.TagKeyRequired
}

// GetTagKeyRequiredOk returns a tuple with the TagKeyRequired field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetTagKeyRequiredOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.TagKeyRequired, true
}

// SetTagKeyRequired sets field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) SetTagKeyRequired(v bool) {
	o.TagKeyRequired = v
}

// GetValidTagValues returns the ValidTagValues field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetValidTagValues() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.ValidTagValues
}

// GetValidTagValuesOk returns a tuple with the ValidTagValues field value
// and a boolean to check if the value has been set.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) GetValidTagValuesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ValidTagValues, true
}

// SetValidTagValues sets field value.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) SetValidTagValues(v []string) {
	o.ValidTagValues = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorConfigPolicyTagPolicyCreateRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["tag_key"] = o.TagKey
	toSerialize["tag_key_required"] = o.TagKeyRequired
	toSerialize["valid_tag_values"] = o.ValidTagValues

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorConfigPolicyTagPolicyCreateRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		TagKey         *string   `json:"tag_key"`
		TagKeyRequired *bool     `json:"tag_key_required"`
		ValidTagValues *[]string `json:"valid_tag_values"`
	}{}
	all := struct {
		TagKey         string   `json:"tag_key"`
		TagKeyRequired bool     `json:"tag_key_required"`
		ValidTagValues []string `json:"valid_tag_values"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.TagKey == nil {
		return fmt.Errorf("required field tag_key missing")
	}
	if required.TagKeyRequired == nil {
		return fmt.Errorf("required field tag_key_required missing")
	}
	if required.ValidTagValues == nil {
		return fmt.Errorf("required field valid_tag_values missing")
	}
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
