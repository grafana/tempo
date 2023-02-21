// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringRuleUpdatePayload Update an existing rule.
type SecurityMonitoringRuleUpdatePayload struct {
	// Cases for generating signals.
	Cases []SecurityMonitoringRuleCase `json:"cases,omitempty"`
	// How to generate compliance signals. Useful for cloud_configuration rules only.
	ComplianceSignalOptions *CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions,omitempty"`
	// Additional queries to filter matched events before they are processed.
	Filters []SecurityMonitoringFilter `json:"filters,omitempty"`
	// Whether the notifications include the triggering group-by values in their title.
	HasExtendedTitle *bool `json:"hasExtendedTitle,omitempty"`
	// Whether the rule is enabled.
	IsEnabled *bool `json:"isEnabled,omitempty"`
	// Message for generated signals.
	Message *string `json:"message,omitempty"`
	// Name of the rule.
	Name *string `json:"name,omitempty"`
	// Options on rules.
	Options *SecurityMonitoringRuleOptions `json:"options,omitempty"`
	// Queries for selecting logs which are part of the rule.
	Queries []SecurityMonitoringRuleQuery `json:"queries,omitempty"`
	// Tags for generated signals.
	Tags []string `json:"tags,omitempty"`
	// The version of the rule being updated.
	Version *int32 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringRuleUpdatePayload instantiates a new SecurityMonitoringRuleUpdatePayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringRuleUpdatePayload() *SecurityMonitoringRuleUpdatePayload {
	this := SecurityMonitoringRuleUpdatePayload{}
	return &this
}

// NewSecurityMonitoringRuleUpdatePayloadWithDefaults instantiates a new SecurityMonitoringRuleUpdatePayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringRuleUpdatePayloadWithDefaults() *SecurityMonitoringRuleUpdatePayload {
	this := SecurityMonitoringRuleUpdatePayload{}
	return &this
}

// GetCases returns the Cases field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetCases() []SecurityMonitoringRuleCase {
	if o == nil || o.Cases == nil {
		var ret []SecurityMonitoringRuleCase
		return ret
	}
	return o.Cases
}

// GetCasesOk returns a tuple with the Cases field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetCasesOk() (*[]SecurityMonitoringRuleCase, bool) {
	if o == nil || o.Cases == nil {
		return nil, false
	}
	return &o.Cases, true
}

// HasCases returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasCases() bool {
	return o != nil && o.Cases != nil
}

// SetCases gets a reference to the given []SecurityMonitoringRuleCase and assigns it to the Cases field.
func (o *SecurityMonitoringRuleUpdatePayload) SetCases(v []SecurityMonitoringRuleCase) {
	o.Cases = v
}

// GetComplianceSignalOptions returns the ComplianceSignalOptions field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetComplianceSignalOptions() CloudConfigurationRuleComplianceSignalOptions {
	if o == nil || o.ComplianceSignalOptions == nil {
		var ret CloudConfigurationRuleComplianceSignalOptions
		return ret
	}
	return *o.ComplianceSignalOptions
}

// GetComplianceSignalOptionsOk returns a tuple with the ComplianceSignalOptions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetComplianceSignalOptionsOk() (*CloudConfigurationRuleComplianceSignalOptions, bool) {
	if o == nil || o.ComplianceSignalOptions == nil {
		return nil, false
	}
	return o.ComplianceSignalOptions, true
}

// HasComplianceSignalOptions returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasComplianceSignalOptions() bool {
	return o != nil && o.ComplianceSignalOptions != nil
}

// SetComplianceSignalOptions gets a reference to the given CloudConfigurationRuleComplianceSignalOptions and assigns it to the ComplianceSignalOptions field.
func (o *SecurityMonitoringRuleUpdatePayload) SetComplianceSignalOptions(v CloudConfigurationRuleComplianceSignalOptions) {
	o.ComplianceSignalOptions = &v
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetFilters() []SecurityMonitoringFilter {
	if o == nil || o.Filters == nil {
		var ret []SecurityMonitoringFilter
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetFiltersOk() (*[]SecurityMonitoringFilter, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasFilters() bool {
	return o != nil && o.Filters != nil
}

// SetFilters gets a reference to the given []SecurityMonitoringFilter and assigns it to the Filters field.
func (o *SecurityMonitoringRuleUpdatePayload) SetFilters(v []SecurityMonitoringFilter) {
	o.Filters = v
}

// GetHasExtendedTitle returns the HasExtendedTitle field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetHasExtendedTitle() bool {
	if o == nil || o.HasExtendedTitle == nil {
		var ret bool
		return ret
	}
	return *o.HasExtendedTitle
}

// GetHasExtendedTitleOk returns a tuple with the HasExtendedTitle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetHasExtendedTitleOk() (*bool, bool) {
	if o == nil || o.HasExtendedTitle == nil {
		return nil, false
	}
	return o.HasExtendedTitle, true
}

// HasHasExtendedTitle returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasHasExtendedTitle() bool {
	return o != nil && o.HasExtendedTitle != nil
}

// SetHasExtendedTitle gets a reference to the given bool and assigns it to the HasExtendedTitle field.
func (o *SecurityMonitoringRuleUpdatePayload) SetHasExtendedTitle(v bool) {
	o.HasExtendedTitle = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasIsEnabled() bool {
	return o != nil && o.IsEnabled != nil
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *SecurityMonitoringRuleUpdatePayload) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SecurityMonitoringRuleUpdatePayload) SetMessage(v string) {
	o.Message = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringRuleUpdatePayload) SetName(v string) {
	o.Name = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetOptions() SecurityMonitoringRuleOptions {
	if o == nil || o.Options == nil {
		var ret SecurityMonitoringRuleOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetOptionsOk() (*SecurityMonitoringRuleOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given SecurityMonitoringRuleOptions and assigns it to the Options field.
func (o *SecurityMonitoringRuleUpdatePayload) SetOptions(v SecurityMonitoringRuleOptions) {
	o.Options = &v
}

// GetQueries returns the Queries field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetQueries() []SecurityMonitoringRuleQuery {
	if o == nil || o.Queries == nil {
		var ret []SecurityMonitoringRuleQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetQueriesOk() (*[]SecurityMonitoringRuleQuery, bool) {
	if o == nil || o.Queries == nil {
		return nil, false
	}
	return &o.Queries, true
}

// HasQueries returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasQueries() bool {
	return o != nil && o.Queries != nil
}

// SetQueries gets a reference to the given []SecurityMonitoringRuleQuery and assigns it to the Queries field.
func (o *SecurityMonitoringRuleUpdatePayload) SetQueries(v []SecurityMonitoringRuleQuery) {
	o.Queries = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SecurityMonitoringRuleUpdatePayload) SetTags(v []string) {
	o.Tags = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringRuleUpdatePayload) GetVersion() int32 {
	if o == nil || o.Version == nil {
		var ret int32
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringRuleUpdatePayload) GetVersionOk() (*int32, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringRuleUpdatePayload) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int32 and assigns it to the Version field.
func (o *SecurityMonitoringRuleUpdatePayload) SetVersion(v int32) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringRuleUpdatePayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cases != nil {
		toSerialize["cases"] = o.Cases
	}
	if o.ComplianceSignalOptions != nil {
		toSerialize["complianceSignalOptions"] = o.ComplianceSignalOptions
	}
	if o.Filters != nil {
		toSerialize["filters"] = o.Filters
	}
	if o.HasExtendedTitle != nil {
		toSerialize["hasExtendedTitle"] = o.HasExtendedTitle
	}
	if o.IsEnabled != nil {
		toSerialize["isEnabled"] = o.IsEnabled
	}
	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Options != nil {
		toSerialize["options"] = o.Options
	}
	if o.Queries != nil {
		toSerialize["queries"] = o.Queries
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringRuleUpdatePayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cases                   []SecurityMonitoringRuleCase                   `json:"cases,omitempty"`
		ComplianceSignalOptions *CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions,omitempty"`
		Filters                 []SecurityMonitoringFilter                     `json:"filters,omitempty"`
		HasExtendedTitle        *bool                                          `json:"hasExtendedTitle,omitempty"`
		IsEnabled               *bool                                          `json:"isEnabled,omitempty"`
		Message                 *string                                        `json:"message,omitempty"`
		Name                    *string                                        `json:"name,omitempty"`
		Options                 *SecurityMonitoringRuleOptions                 `json:"options,omitempty"`
		Queries                 []SecurityMonitoringRuleQuery                  `json:"queries,omitempty"`
		Tags                    []string                                       `json:"tags,omitempty"`
		Version                 *int32                                         `json:"version,omitempty"`
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
	o.Cases = all.Cases
	if all.ComplianceSignalOptions != nil && all.ComplianceSignalOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ComplianceSignalOptions = all.ComplianceSignalOptions
	o.Filters = all.Filters
	o.HasExtendedTitle = all.HasExtendedTitle
	o.IsEnabled = all.IsEnabled
	o.Message = all.Message
	o.Name = all.Name
	if all.Options != nil && all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	o.Queries = all.Queries
	o.Tags = all.Tags
	o.Version = all.Version
	return nil
}
