// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudConfigurationRuleOptions Options on cloud configuration rules.
type CloudConfigurationRuleOptions struct {
	// Options for cloud_configuration rules.
	// Fields `resourceType` and `regoRule` are mandatory when managing custom `cloud_configuration` rules.
	//
	ComplianceRuleOptions CloudConfigurationComplianceRuleOptions `json:"complianceRuleOptions"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudConfigurationRuleOptions instantiates a new CloudConfigurationRuleOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudConfigurationRuleOptions(complianceRuleOptions CloudConfigurationComplianceRuleOptions) *CloudConfigurationRuleOptions {
	this := CloudConfigurationRuleOptions{}
	this.ComplianceRuleOptions = complianceRuleOptions
	return &this
}

// NewCloudConfigurationRuleOptionsWithDefaults instantiates a new CloudConfigurationRuleOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudConfigurationRuleOptionsWithDefaults() *CloudConfigurationRuleOptions {
	this := CloudConfigurationRuleOptions{}
	return &this
}

// GetComplianceRuleOptions returns the ComplianceRuleOptions field value.
func (o *CloudConfigurationRuleOptions) GetComplianceRuleOptions() CloudConfigurationComplianceRuleOptions {
	if o == nil {
		var ret CloudConfigurationComplianceRuleOptions
		return ret
	}
	return o.ComplianceRuleOptions
}

// GetComplianceRuleOptionsOk returns a tuple with the ComplianceRuleOptions field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleOptions) GetComplianceRuleOptionsOk() (*CloudConfigurationComplianceRuleOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ComplianceRuleOptions, true
}

// SetComplianceRuleOptions sets field value.
func (o *CloudConfigurationRuleOptions) SetComplianceRuleOptions(v CloudConfigurationComplianceRuleOptions) {
	o.ComplianceRuleOptions = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudConfigurationRuleOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["complianceRuleOptions"] = o.ComplianceRuleOptions

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudConfigurationRuleOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ComplianceRuleOptions *CloudConfigurationComplianceRuleOptions `json:"complianceRuleOptions"`
	}{}
	all := struct {
		ComplianceRuleOptions CloudConfigurationComplianceRuleOptions `json:"complianceRuleOptions"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ComplianceRuleOptions == nil {
		return fmt.Errorf("required field complianceRuleOptions missing")
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
	if all.ComplianceRuleOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ComplianceRuleOptions = all.ComplianceRuleOptions
	return nil
}
