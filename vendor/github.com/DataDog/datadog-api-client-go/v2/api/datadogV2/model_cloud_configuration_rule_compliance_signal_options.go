// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudConfigurationRuleComplianceSignalOptions How to generate compliance signals. Useful for cloud_configuration rules only.
type CloudConfigurationRuleComplianceSignalOptions struct {
	// Whether signals will be sent.
	UserActivationStatus *bool `json:"userActivationStatus,omitempty"`
	// Fields to use to group findings by when sending signals.
	UserGroupByFields []string `json:"userGroupByFields,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudConfigurationRuleComplianceSignalOptions instantiates a new CloudConfigurationRuleComplianceSignalOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudConfigurationRuleComplianceSignalOptions() *CloudConfigurationRuleComplianceSignalOptions {
	this := CloudConfigurationRuleComplianceSignalOptions{}
	return &this
}

// NewCloudConfigurationRuleComplianceSignalOptionsWithDefaults instantiates a new CloudConfigurationRuleComplianceSignalOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudConfigurationRuleComplianceSignalOptionsWithDefaults() *CloudConfigurationRuleComplianceSignalOptions {
	this := CloudConfigurationRuleComplianceSignalOptions{}
	return &this
}

// GetUserActivationStatus returns the UserActivationStatus field value if set, zero value otherwise.
func (o *CloudConfigurationRuleComplianceSignalOptions) GetUserActivationStatus() bool {
	if o == nil || o.UserActivationStatus == nil {
		var ret bool
		return ret
	}
	return *o.UserActivationStatus
}

// GetUserActivationStatusOk returns a tuple with the UserActivationStatus field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleComplianceSignalOptions) GetUserActivationStatusOk() (*bool, bool) {
	if o == nil || o.UserActivationStatus == nil {
		return nil, false
	}
	return o.UserActivationStatus, true
}

// HasUserActivationStatus returns a boolean if a field has been set.
func (o *CloudConfigurationRuleComplianceSignalOptions) HasUserActivationStatus() bool {
	return o != nil && o.UserActivationStatus != nil
}

// SetUserActivationStatus gets a reference to the given bool and assigns it to the UserActivationStatus field.
func (o *CloudConfigurationRuleComplianceSignalOptions) SetUserActivationStatus(v bool) {
	o.UserActivationStatus = &v
}

// GetUserGroupByFields returns the UserGroupByFields field value if set, zero value otherwise.
func (o *CloudConfigurationRuleComplianceSignalOptions) GetUserGroupByFields() []string {
	if o == nil || o.UserGroupByFields == nil {
		var ret []string
		return ret
	}
	return o.UserGroupByFields
}

// GetUserGroupByFieldsOk returns a tuple with the UserGroupByFields field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleComplianceSignalOptions) GetUserGroupByFieldsOk() (*[]string, bool) {
	if o == nil || o.UserGroupByFields == nil {
		return nil, false
	}
	return &o.UserGroupByFields, true
}

// HasUserGroupByFields returns a boolean if a field has been set.
func (o *CloudConfigurationRuleComplianceSignalOptions) HasUserGroupByFields() bool {
	return o != nil && o.UserGroupByFields != nil
}

// SetUserGroupByFields gets a reference to the given []string and assigns it to the UserGroupByFields field.
func (o *CloudConfigurationRuleComplianceSignalOptions) SetUserGroupByFields(v []string) {
	o.UserGroupByFields = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudConfigurationRuleComplianceSignalOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.UserActivationStatus != nil {
		toSerialize["userActivationStatus"] = o.UserActivationStatus
	}
	if o.UserGroupByFields != nil {
		toSerialize["userGroupByFields"] = o.UserGroupByFields
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CloudConfigurationRuleComplianceSignalOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		UserActivationStatus *bool    `json:"userActivationStatus,omitempty"`
		UserGroupByFields    []string `json:"userGroupByFields,omitempty"`
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
	o.UserActivationStatus = all.UserActivationStatus
	o.UserGroupByFields = all.UserGroupByFields
	return nil
}
