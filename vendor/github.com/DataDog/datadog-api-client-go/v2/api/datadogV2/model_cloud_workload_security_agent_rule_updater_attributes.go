// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudWorkloadSecurityAgentRuleUpdaterAttributes The attributes of the user who last updated the Agent rule.
type CloudWorkloadSecurityAgentRuleUpdaterAttributes struct {
	// The handle of the user.
	Handle *string `json:"handle,omitempty"`
	// The name of the user.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleUpdaterAttributes instantiates a new CloudWorkloadSecurityAgentRuleUpdaterAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleUpdaterAttributes() *CloudWorkloadSecurityAgentRuleUpdaterAttributes {
	this := CloudWorkloadSecurityAgentRuleUpdaterAttributes{}
	return &this
}

// NewCloudWorkloadSecurityAgentRuleUpdaterAttributesWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleUpdaterAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleUpdaterAttributesWithDefaults() *CloudWorkloadSecurityAgentRuleUpdaterAttributes {
	this := CloudWorkloadSecurityAgentRuleUpdaterAttributes{}
	return &this
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) HasHandle() bool {
	return o != nil && o.Handle != nil
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) SetHandle(v string) {
	o.Handle = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleUpdaterAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudWorkloadSecurityAgentRuleUpdaterAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Handle *string `json:"handle,omitempty"`
		Name   *string `json:"name,omitempty"`
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
	o.Handle = all.Handle
	o.Name = all.Name
	return nil
}
