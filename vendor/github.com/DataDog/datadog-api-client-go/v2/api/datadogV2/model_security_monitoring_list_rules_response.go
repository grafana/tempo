// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringListRulesResponse List of rules.
type SecurityMonitoringListRulesResponse struct {
	// Array containing the list of rules.
	Data []SecurityMonitoringRuleResponse `json:"data,omitempty"`
	// Object describing meta attributes of response.
	Meta *ResponseMetaAttributes `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringListRulesResponse instantiates a new SecurityMonitoringListRulesResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringListRulesResponse() *SecurityMonitoringListRulesResponse {
	this := SecurityMonitoringListRulesResponse{}
	return &this
}

// NewSecurityMonitoringListRulesResponseWithDefaults instantiates a new SecurityMonitoringListRulesResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringListRulesResponseWithDefaults() *SecurityMonitoringListRulesResponse {
	this := SecurityMonitoringListRulesResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SecurityMonitoringListRulesResponse) GetData() []SecurityMonitoringRuleResponse {
	if o == nil || o.Data == nil {
		var ret []SecurityMonitoringRuleResponse
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringListRulesResponse) GetDataOk() (*[]SecurityMonitoringRuleResponse, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SecurityMonitoringListRulesResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []SecurityMonitoringRuleResponse and assigns it to the Data field.
func (o *SecurityMonitoringListRulesResponse) SetData(v []SecurityMonitoringRuleResponse) {
	o.Data = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *SecurityMonitoringListRulesResponse) GetMeta() ResponseMetaAttributes {
	if o == nil || o.Meta == nil {
		var ret ResponseMetaAttributes
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringListRulesResponse) GetMetaOk() (*ResponseMetaAttributes, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *SecurityMonitoringListRulesResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given ResponseMetaAttributes and assigns it to the Meta field.
func (o *SecurityMonitoringListRulesResponse) SetMeta(v ResponseMetaAttributes) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringListRulesResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringListRulesResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data []SecurityMonitoringRuleResponse `json:"data,omitempty"`
		Meta *ResponseMetaAttributes          `json:"meta,omitempty"`
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
	o.Data = all.Data
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
