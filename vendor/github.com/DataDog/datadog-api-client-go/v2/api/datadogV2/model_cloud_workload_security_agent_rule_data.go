// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudWorkloadSecurityAgentRuleData Object for a single Agent rule.
type CloudWorkloadSecurityAgentRuleData struct {
	// A Cloud Workload Security Agent rule returned by the API.
	Attributes *CloudWorkloadSecurityAgentRuleAttributes `json:"attributes,omitempty"`
	// The ID of the Agent rule.
	Id *string `json:"id,omitempty"`
	// The type of the resource. The value should always be `agent_rule`.
	Type *CloudWorkloadSecurityAgentRuleType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleData instantiates a new CloudWorkloadSecurityAgentRuleData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleData() *CloudWorkloadSecurityAgentRuleData {
	this := CloudWorkloadSecurityAgentRuleData{}
	var typeVar CloudWorkloadSecurityAgentRuleType = CLOUDWORKLOADSECURITYAGENTRULETYPE_AGENT_RULE
	this.Type = &typeVar
	return &this
}

// NewCloudWorkloadSecurityAgentRuleDataWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleDataWithDefaults() *CloudWorkloadSecurityAgentRuleData {
	this := CloudWorkloadSecurityAgentRuleData{}
	var typeVar CloudWorkloadSecurityAgentRuleType = CLOUDWORKLOADSECURITYAGENTRULETYPE_AGENT_RULE
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleData) GetAttributes() CloudWorkloadSecurityAgentRuleAttributes {
	if o == nil || o.Attributes == nil {
		var ret CloudWorkloadSecurityAgentRuleAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleData) GetAttributesOk() (*CloudWorkloadSecurityAgentRuleAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given CloudWorkloadSecurityAgentRuleAttributes and assigns it to the Attributes field.
func (o *CloudWorkloadSecurityAgentRuleData) SetAttributes(v CloudWorkloadSecurityAgentRuleAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *CloudWorkloadSecurityAgentRuleData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleData) GetType() CloudWorkloadSecurityAgentRuleType {
	if o == nil || o.Type == nil {
		var ret CloudWorkloadSecurityAgentRuleType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleData) GetTypeOk() (*CloudWorkloadSecurityAgentRuleType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleData) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given CloudWorkloadSecurityAgentRuleType and assigns it to the Type field.
func (o *CloudWorkloadSecurityAgentRuleData) SetType(v CloudWorkloadSecurityAgentRuleType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudWorkloadSecurityAgentRuleData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *CloudWorkloadSecurityAgentRuleAttributes `json:"attributes,omitempty"`
		Id         *string                                   `json:"id,omitempty"`
		Type       *CloudWorkloadSecurityAgentRuleType       `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
