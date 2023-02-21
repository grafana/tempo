// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudConfigurationRegoRule Rule details.
type CloudConfigurationRegoRule struct {
	// The policy written in `rego`, see: https://www.openpolicyagent.org/docs/latest/policy-language/
	Policy string `json:"policy"`
	// List of resource types that will be evaluated upon. Must have at least one element.
	ResourceTypes []string `json:"resourceTypes"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudConfigurationRegoRule instantiates a new CloudConfigurationRegoRule object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudConfigurationRegoRule(policy string, resourceTypes []string) *CloudConfigurationRegoRule {
	this := CloudConfigurationRegoRule{}
	this.Policy = policy
	this.ResourceTypes = resourceTypes
	return &this
}

// NewCloudConfigurationRegoRuleWithDefaults instantiates a new CloudConfigurationRegoRule object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudConfigurationRegoRuleWithDefaults() *CloudConfigurationRegoRule {
	this := CloudConfigurationRegoRule{}
	return &this
}

// GetPolicy returns the Policy field value.
func (o *CloudConfigurationRegoRule) GetPolicy() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Policy
}

// GetPolicyOk returns a tuple with the Policy field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRegoRule) GetPolicyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Policy, true
}

// SetPolicy sets field value.
func (o *CloudConfigurationRegoRule) SetPolicy(v string) {
	o.Policy = v
}

// GetResourceTypes returns the ResourceTypes field value.
func (o *CloudConfigurationRegoRule) GetResourceTypes() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.ResourceTypes
}

// GetResourceTypesOk returns a tuple with the ResourceTypes field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRegoRule) GetResourceTypesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ResourceTypes, true
}

// SetResourceTypes sets field value.
func (o *CloudConfigurationRegoRule) SetResourceTypes(v []string) {
	o.ResourceTypes = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudConfigurationRegoRule) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["policy"] = o.Policy
	toSerialize["resourceTypes"] = o.ResourceTypes

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudConfigurationRegoRule) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Policy        *string   `json:"policy"`
		ResourceTypes *[]string `json:"resourceTypes"`
	}{}
	all := struct {
		Policy        string   `json:"policy"`
		ResourceTypes []string `json:"resourceTypes"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Policy == nil {
		return fmt.Errorf("required field policy missing")
	}
	if required.ResourceTypes == nil {
		return fmt.Errorf("required field resourceTypes missing")
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
	o.Policy = all.Policy
	o.ResourceTypes = all.ResourceTypes
	return nil
}
