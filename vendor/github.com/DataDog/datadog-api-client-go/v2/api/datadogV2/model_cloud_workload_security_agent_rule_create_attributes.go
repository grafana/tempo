// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudWorkloadSecurityAgentRuleCreateAttributes Create a new Cloud Workload Security Agent rule.
type CloudWorkloadSecurityAgentRuleCreateAttributes struct {
	// The description of the Agent rule.
	Description *string `json:"description,omitempty"`
	// Whether the Agent rule is enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// The SECL expression of the Agent rule.
	Expression string `json:"expression"`
	// The name of the Agent rule.
	Name string `json:"name"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleCreateAttributes instantiates a new CloudWorkloadSecurityAgentRuleCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleCreateAttributes(expression string, name string) *CloudWorkloadSecurityAgentRuleCreateAttributes {
	this := CloudWorkloadSecurityAgentRuleCreateAttributes{}
	this.Expression = expression
	this.Name = name
	return &this
}

// NewCloudWorkloadSecurityAgentRuleCreateAttributesWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleCreateAttributesWithDefaults() *CloudWorkloadSecurityAgentRuleCreateAttributes {
	this := CloudWorkloadSecurityAgentRuleCreateAttributes{}
	return &this
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) HasEnabled() bool {
	return o != nil && o.Enabled != nil
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) SetEnabled(v bool) {
	o.Enabled = &v
}

// GetExpression returns the Expression field value.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetExpression() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetExpressionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Expression, true
}

// SetExpression sets field value.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) SetExpression(v string) {
	o.Expression = v
}

// GetName returns the Name field value.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) SetName(v string) {
	o.Name = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleCreateAttributes) MarshalJSON() ([]byte, error) {
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
	toSerialize["expression"] = o.Expression
	toSerialize["name"] = o.Name

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudWorkloadSecurityAgentRuleCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Expression *string `json:"expression"`
		Name       *string `json:"name"`
	}{}
	all := struct {
		Description *string `json:"description,omitempty"`
		Enabled     *bool   `json:"enabled,omitempty"`
		Expression  string  `json:"expression"`
		Name        string  `json:"name"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Expression == nil {
		return fmt.Errorf("required field expression missing")
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
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
	o.Description = all.Description
	o.Enabled = all.Enabled
	o.Expression = all.Expression
	o.Name = all.Name
	return nil
}
