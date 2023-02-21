// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// CloudConfigurationRuleCreatePayload Create a new cloud configuration rule.
type CloudConfigurationRuleCreatePayload struct {
	// Description of generated findings and signals (severity and channels to be notified in case of a signal). Must contain exactly one item.
	//
	Cases []CloudConfigurationRuleCaseCreate `json:"cases"`
	// How to generate compliance signals. Useful for cloud_configuration rules only.
	ComplianceSignalOptions CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions"`
	// Whether the rule is enabled.
	IsEnabled bool `json:"isEnabled"`
	// Message in markdown format for generated findings and signals.
	Message string `json:"message"`
	// The name of the rule.
	Name string `json:"name"`
	// Options on cloud configuration rules.
	Options CloudConfigurationRuleOptions `json:"options"`
	// Tags for generated findings and signals.
	Tags []string `json:"tags,omitempty"`
	// The rule type.
	Type *CloudConfigurationRuleType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudConfigurationRuleCreatePayload instantiates a new CloudConfigurationRuleCreatePayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudConfigurationRuleCreatePayload(cases []CloudConfigurationRuleCaseCreate, complianceSignalOptions CloudConfigurationRuleComplianceSignalOptions, isEnabled bool, message string, name string, options CloudConfigurationRuleOptions) *CloudConfigurationRuleCreatePayload {
	this := CloudConfigurationRuleCreatePayload{}
	this.Cases = cases
	this.ComplianceSignalOptions = complianceSignalOptions
	this.IsEnabled = isEnabled
	this.Message = message
	this.Name = name
	this.Options = options
	return &this
}

// NewCloudConfigurationRuleCreatePayloadWithDefaults instantiates a new CloudConfigurationRuleCreatePayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudConfigurationRuleCreatePayloadWithDefaults() *CloudConfigurationRuleCreatePayload {
	this := CloudConfigurationRuleCreatePayload{}
	return &this
}

// GetCases returns the Cases field value.
func (o *CloudConfigurationRuleCreatePayload) GetCases() []CloudConfigurationRuleCaseCreate {
	if o == nil {
		var ret []CloudConfigurationRuleCaseCreate
		return ret
	}
	return o.Cases
}

// GetCasesOk returns a tuple with the Cases field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetCasesOk() (*[]CloudConfigurationRuleCaseCreate, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Cases, true
}

// SetCases sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetCases(v []CloudConfigurationRuleCaseCreate) {
	o.Cases = v
}

// GetComplianceSignalOptions returns the ComplianceSignalOptions field value.
func (o *CloudConfigurationRuleCreatePayload) GetComplianceSignalOptions() CloudConfigurationRuleComplianceSignalOptions {
	if o == nil {
		var ret CloudConfigurationRuleComplianceSignalOptions
		return ret
	}
	return o.ComplianceSignalOptions
}

// GetComplianceSignalOptionsOk returns a tuple with the ComplianceSignalOptions field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetComplianceSignalOptionsOk() (*CloudConfigurationRuleComplianceSignalOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ComplianceSignalOptions, true
}

// SetComplianceSignalOptions sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetComplianceSignalOptions(v CloudConfigurationRuleComplianceSignalOptions) {
	o.ComplianceSignalOptions = v
}

// GetIsEnabled returns the IsEnabled field value.
func (o *CloudConfigurationRuleCreatePayload) GetIsEnabled() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetIsEnabledOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsEnabled, true
}

// SetIsEnabled sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetIsEnabled(v bool) {
	o.IsEnabled = v
}

// GetMessage returns the Message field value.
func (o *CloudConfigurationRuleCreatePayload) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetMessage(v string) {
	o.Message = v
}

// GetName returns the Name field value.
func (o *CloudConfigurationRuleCreatePayload) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetName(v string) {
	o.Name = v
}

// GetOptions returns the Options field value.
func (o *CloudConfigurationRuleCreatePayload) GetOptions() CloudConfigurationRuleOptions {
	if o == nil {
		var ret CloudConfigurationRuleOptions
		return ret
	}
	return o.Options
}

// GetOptionsOk returns a tuple with the Options field value
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetOptionsOk() (*CloudConfigurationRuleOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Options, true
}

// SetOptions sets field value.
func (o *CloudConfigurationRuleCreatePayload) SetOptions(v CloudConfigurationRuleOptions) {
	o.Options = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *CloudConfigurationRuleCreatePayload) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *CloudConfigurationRuleCreatePayload) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *CloudConfigurationRuleCreatePayload) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *CloudConfigurationRuleCreatePayload) GetType() CloudConfigurationRuleType {
	if o == nil || o.Type == nil {
		var ret CloudConfigurationRuleType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudConfigurationRuleCreatePayload) GetTypeOk() (*CloudConfigurationRuleType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *CloudConfigurationRuleCreatePayload) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given CloudConfigurationRuleType and assigns it to the Type field.
func (o *CloudConfigurationRuleCreatePayload) SetType(v CloudConfigurationRuleType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudConfigurationRuleCreatePayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["cases"] = o.Cases
	toSerialize["complianceSignalOptions"] = o.ComplianceSignalOptions
	toSerialize["isEnabled"] = o.IsEnabled
	toSerialize["message"] = o.Message
	toSerialize["name"] = o.Name
	toSerialize["options"] = o.Options
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
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
func (o *CloudConfigurationRuleCreatePayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Cases                   *[]CloudConfigurationRuleCaseCreate            `json:"cases"`
		ComplianceSignalOptions *CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions"`
		IsEnabled               *bool                                          `json:"isEnabled"`
		Message                 *string                                        `json:"message"`
		Name                    *string                                        `json:"name"`
		Options                 *CloudConfigurationRuleOptions                 `json:"options"`
	}{}
	all := struct {
		Cases                   []CloudConfigurationRuleCaseCreate            `json:"cases"`
		ComplianceSignalOptions CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions"`
		IsEnabled               bool                                          `json:"isEnabled"`
		Message                 string                                        `json:"message"`
		Name                    string                                        `json:"name"`
		Options                 CloudConfigurationRuleOptions                 `json:"options"`
		Tags                    []string                                      `json:"tags,omitempty"`
		Type                    *CloudConfigurationRuleType                   `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Cases == nil {
		return fmt.Errorf("required field cases missing")
	}
	if required.ComplianceSignalOptions == nil {
		return fmt.Errorf("required field complianceSignalOptions missing")
	}
	if required.IsEnabled == nil {
		return fmt.Errorf("required field isEnabled missing")
	}
	if required.Message == nil {
		return fmt.Errorf("required field message missing")
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Options == nil {
		return fmt.Errorf("required field options missing")
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Cases = all.Cases
	if all.ComplianceSignalOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ComplianceSignalOptions = all.ComplianceSignalOptions
	o.IsEnabled = all.IsEnabled
	o.Message = all.Message
	o.Name = all.Name
	if all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}
