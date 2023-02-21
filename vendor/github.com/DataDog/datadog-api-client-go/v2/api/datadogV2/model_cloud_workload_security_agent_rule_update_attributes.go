// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudWorkloadSecurityAgentRuleUpdateAttributes Update an existing Cloud Workload Security Agent rule.
type CloudWorkloadSecurityAgentRuleUpdateAttributes struct {
	// The description of the Agent rule.
	Description *string `json:"description,omitempty"`
	// Whether the Agent rule is enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// The SECL expression of the Agent rule.
	Expression *string `json:"expression,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleUpdateAttributes instantiates a new CloudWorkloadSecurityAgentRuleUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleUpdateAttributes() *CloudWorkloadSecurityAgentRuleUpdateAttributes {
	this := CloudWorkloadSecurityAgentRuleUpdateAttributes{}
	return &this
}

// NewCloudWorkloadSecurityAgentRuleUpdateAttributesWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleUpdateAttributesWithDefaults() *CloudWorkloadSecurityAgentRuleUpdateAttributes {
	this := CloudWorkloadSecurityAgentRuleUpdateAttributes{}
	return &this
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) HasEnabled() bool {
	return o != nil && o.Enabled != nil
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) SetEnabled(v bool) {
	o.Enabled = &v
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetExpression() string {
	if o == nil || o.Expression == nil {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) GetExpressionOk() (*string, bool) {
	if o == nil || o.Expression == nil {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) HasExpression() bool {
	return o != nil && o.Expression != nil
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) SetExpression(v string) {
	o.Expression = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.Enabled != nil {
		toSerialize["enabled"] = o.Enabled
	}
	if o.Expression != nil {
		toSerialize["expression"] = o.Expression
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudWorkloadSecurityAgentRuleUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Description *string `json:"description,omitempty"`
		Enabled     *bool   `json:"enabled,omitempty"`
		Expression  *string `json:"expression,omitempty"`
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
	o.Description = all.Description
	o.Enabled = all.Enabled
	o.Expression = all.Expression
	return nil
}
