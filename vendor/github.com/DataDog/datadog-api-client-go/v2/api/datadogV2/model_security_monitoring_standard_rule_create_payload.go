// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringStandardRuleCreatePayload Create a new rule.
type SecurityMonitoringStandardRuleCreatePayload struct {
	// Cases for generating signals.
	Cases []SecurityMonitoringRuleCaseCreate `json:"cases"`
	// Additional queries to filter matched events before they are processed.
	Filters []SecurityMonitoringFilter `json:"filters,omitempty"`
	// Whether the notifications include the triggering group-by values in their title.
	HasExtendedTitle *bool `json:"hasExtendedTitle,omitempty"`
	// Whether the rule is enabled.
	IsEnabled bool `json:"isEnabled"`
	// Message for generated signals.
	Message string `json:"message"`
	// The name of the rule.
	Name string `json:"name"`
	// Options on rules.
	Options SecurityMonitoringRuleOptions `json:"options"`
	// Queries for selecting logs which are part of the rule.
	Queries []SecurityMonitoringStandardRuleQuery `json:"queries"`
	// Tags for generated signals.
	Tags []string `json:"tags,omitempty"`
	// The rule type.
	Type *SecurityMonitoringRuleTypeCreate `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringStandardRuleCreatePayload instantiates a new SecurityMonitoringStandardRuleCreatePayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringStandardRuleCreatePayload(cases []SecurityMonitoringRuleCaseCreate, isEnabled bool, message string, name string, options SecurityMonitoringRuleOptions, queries []SecurityMonitoringStandardRuleQuery) *SecurityMonitoringStandardRuleCreatePayload {
	this := SecurityMonitoringStandardRuleCreatePayload{}
	this.Cases = cases
	this.IsEnabled = isEnabled
	this.Message = message
	this.Name = name
	this.Options = options
	this.Queries = queries
	return &this
}

// NewSecurityMonitoringStandardRuleCreatePayloadWithDefaults instantiates a new SecurityMonitoringStandardRuleCreatePayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringStandardRuleCreatePayloadWithDefaults() *SecurityMonitoringStandardRuleCreatePayload {
	this := SecurityMonitoringStandardRuleCreatePayload{}
	return &this
}

// GetCases returns the Cases field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetCases() []SecurityMonitoringRuleCaseCreate {
	if o == nil {
		var ret []SecurityMonitoringRuleCaseCreate
		return ret
	}
	return o.Cases
}

// GetCasesOk returns a tuple with the Cases field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetCasesOk() (*[]SecurityMonitoringRuleCaseCreate, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Cases, true
}

// SetCases sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetCases(v []SecurityMonitoringRuleCaseCreate) {
	o.Cases = v
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetFilters() []SecurityMonitoringFilter {
	if o == nil || o.Filters == nil {
		var ret []SecurityMonitoringFilter
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetFiltersOk() (*[]SecurityMonitoringFilter, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) HasFilters() bool {
	return o != nil && o.Filters != nil
}

// SetFilters gets a reference to the given []SecurityMonitoringFilter and assigns it to the Filters field.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetFilters(v []SecurityMonitoringFilter) {
	o.Filters = v
}

// GetHasExtendedTitle returns the HasExtendedTitle field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetHasExtendedTitle() bool {
	if o == nil || o.HasExtendedTitle == nil {
		var ret bool
		return ret
	}
	return *o.HasExtendedTitle
}

// GetHasExtendedTitleOk returns a tuple with the HasExtendedTitle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetHasExtendedTitleOk() (*bool, bool) {
	if o == nil || o.HasExtendedTitle == nil {
		return nil, false
	}
	return o.HasExtendedTitle, true
}

// HasHasExtendedTitle returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) HasHasExtendedTitle() bool {
	return o != nil && o.HasExtendedTitle != nil
}

// SetHasExtendedTitle gets a reference to the given bool and assigns it to the HasExtendedTitle field.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetHasExtendedTitle(v bool) {
	o.HasExtendedTitle = &v
}

// GetIsEnabled returns the IsEnabled field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetIsEnabled() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetIsEnabledOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsEnabled, true
}

// SetIsEnabled sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetIsEnabled(v bool) {
	o.IsEnabled = v
}

// GetMessage returns the Message field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetMessage() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Message, true
}

// SetMessage sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetMessage(v string) {
	o.Message = v
}

// GetName returns the Name field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetName(v string) {
	o.Name = v
}

// GetOptions returns the Options field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetOptions() SecurityMonitoringRuleOptions {
	if o == nil {
		var ret SecurityMonitoringRuleOptions
		return ret
	}
	return o.Options
}

// GetOptionsOk returns a tuple with the Options field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetOptionsOk() (*SecurityMonitoringRuleOptions, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Options, true
}

// SetOptions sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetOptions(v SecurityMonitoringRuleOptions) {
	o.Options = v
}

// GetQueries returns the Queries field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetQueries() []SecurityMonitoringStandardRuleQuery {
	if o == nil {
		var ret []SecurityMonitoringStandardRuleQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetQueriesOk() (*[]SecurityMonitoringStandardRuleQuery, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Queries, true
}

// SetQueries sets field value.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetQueries(v []SecurityMonitoringStandardRuleQuery) {
	o.Queries = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetType() SecurityMonitoringRuleTypeCreate {
	if o == nil || o.Type == nil {
		var ret SecurityMonitoringRuleTypeCreate
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) GetTypeOk() (*SecurityMonitoringRuleTypeCreate, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleCreatePayload) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SecurityMonitoringRuleTypeCreate and assigns it to the Type field.
func (o *SecurityMonitoringStandardRuleCreatePayload) SetType(v SecurityMonitoringRuleTypeCreate) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringStandardRuleCreatePayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["cases"] = o.Cases
	if o.Filters != nil {
		toSerialize["filters"] = o.Filters
	}
	if o.HasExtendedTitle != nil {
		toSerialize["hasExtendedTitle"] = o.HasExtendedTitle
	}
	toSerialize["isEnabled"] = o.IsEnabled
	toSerialize["message"] = o.Message
	toSerialize["name"] = o.Name
	toSerialize["options"] = o.Options
	toSerialize["queries"] = o.Queries
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
func (o *SecurityMonitoringStandardRuleCreatePayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Cases     *[]SecurityMonitoringRuleCaseCreate    `json:"cases"`
		IsEnabled *bool                                  `json:"isEnabled"`
		Message   *string                                `json:"message"`
		Name      *string                                `json:"name"`
		Options   *SecurityMonitoringRuleOptions         `json:"options"`
		Queries   *[]SecurityMonitoringStandardRuleQuery `json:"queries"`
	}{}
	all := struct {
		Cases            []SecurityMonitoringRuleCaseCreate    `json:"cases"`
		Filters          []SecurityMonitoringFilter            `json:"filters,omitempty"`
		HasExtendedTitle *bool                                 `json:"hasExtendedTitle,omitempty"`
		IsEnabled        bool                                  `json:"isEnabled"`
		Message          string                                `json:"message"`
		Name             string                                `json:"name"`
		Options          SecurityMonitoringRuleOptions         `json:"options"`
		Queries          []SecurityMonitoringStandardRuleQuery `json:"queries"`
		Tags             []string                              `json:"tags,omitempty"`
		Type             *SecurityMonitoringRuleTypeCreate     `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Cases == nil {
		return fmt.Errorf("required field cases missing")
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
	if required.Queries == nil {
		return fmt.Errorf("required field queries missing")
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
	o.Filters = all.Filters
	o.HasExtendedTitle = all.HasExtendedTitle
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
	o.Queries = all.Queries
	o.Tags = all.Tags
	o.Type = all.Type
	return nil
}
