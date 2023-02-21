// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringStandardRuleResponse Rule.
type SecurityMonitoringStandardRuleResponse struct {
	// Cases for generating signals.
	Cases []SecurityMonitoringRuleCase `json:"cases,omitempty"`
	// How to generate compliance signals. Useful for cloud_configuration rules only.
	ComplianceSignalOptions *CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions,omitempty"`
	// When the rule was created, timestamp in milliseconds.
	CreatedAt *int64 `json:"createdAt,omitempty"`
	// User ID of the user who created the rule.
	CreationAuthorId *int64 `json:"creationAuthorId,omitempty"`
	// When the rule will be deprecated, timestamp in milliseconds.
	DeprecationDate *int64 `json:"deprecationDate,omitempty"`
	// Additional queries to filter matched events before they are processed.
	Filters []SecurityMonitoringFilter `json:"filters,omitempty"`
	// Whether the notifications include the triggering group-by values in their title.
	HasExtendedTitle *bool `json:"hasExtendedTitle,omitempty"`
	// The ID of the rule.
	Id *string `json:"id,omitempty"`
	// Whether the rule is included by default.
	IsDefault *bool `json:"isDefault,omitempty"`
	// Whether the rule has been deleted.
	IsDeleted *bool `json:"isDeleted,omitempty"`
	// Whether the rule is enabled.
	IsEnabled *bool `json:"isEnabled,omitempty"`
	// Message for generated signals.
	Message *string `json:"message,omitempty"`
	// The name of the rule.
	Name *string `json:"name,omitempty"`
	// Options on rules.
	Options *SecurityMonitoringRuleOptions `json:"options,omitempty"`
	// Queries for selecting logs which are part of the rule.
	Queries []SecurityMonitoringStandardRuleQuery `json:"queries,omitempty"`
	// Tags for generated signals.
	Tags []string `json:"tags,omitempty"`
	// The rule type.
	Type *SecurityMonitoringRuleTypeRead `json:"type,omitempty"`
	// User ID of the user who updated the rule.
	UpdateAuthorId *int64 `json:"updateAuthorId,omitempty"`
	// The version of the rule.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringStandardRuleResponse instantiates a new SecurityMonitoringStandardRuleResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringStandardRuleResponse() *SecurityMonitoringStandardRuleResponse {
	this := SecurityMonitoringStandardRuleResponse{}
	return &this
}

// NewSecurityMonitoringStandardRuleResponseWithDefaults instantiates a new SecurityMonitoringStandardRuleResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringStandardRuleResponseWithDefaults() *SecurityMonitoringStandardRuleResponse {
	this := SecurityMonitoringStandardRuleResponse{}
	return &this
}

// GetCases returns the Cases field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetCases() []SecurityMonitoringRuleCase {
	if o == nil || o.Cases == nil {
		var ret []SecurityMonitoringRuleCase
		return ret
	}
	return o.Cases
}

// GetCasesOk returns a tuple with the Cases field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetCasesOk() (*[]SecurityMonitoringRuleCase, bool) {
	if o == nil || o.Cases == nil {
		return nil, false
	}
	return &o.Cases, true
}

// HasCases returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasCases() bool {
	return o != nil && o.Cases != nil
}

// SetCases gets a reference to the given []SecurityMonitoringRuleCase and assigns it to the Cases field.
func (o *SecurityMonitoringStandardRuleResponse) SetCases(v []SecurityMonitoringRuleCase) {
	o.Cases = v
}

// GetComplianceSignalOptions returns the ComplianceSignalOptions field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetComplianceSignalOptions() CloudConfigurationRuleComplianceSignalOptions {
	if o == nil || o.ComplianceSignalOptions == nil {
		var ret CloudConfigurationRuleComplianceSignalOptions
		return ret
	}
	return *o.ComplianceSignalOptions
}

// GetComplianceSignalOptionsOk returns a tuple with the ComplianceSignalOptions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetComplianceSignalOptionsOk() (*CloudConfigurationRuleComplianceSignalOptions, bool) {
	if o == nil || o.ComplianceSignalOptions == nil {
		return nil, false
	}
	return o.ComplianceSignalOptions, true
}

// HasComplianceSignalOptions returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasComplianceSignalOptions() bool {
	return o != nil && o.ComplianceSignalOptions != nil
}

// SetComplianceSignalOptions gets a reference to the given CloudConfigurationRuleComplianceSignalOptions and assigns it to the ComplianceSignalOptions field.
func (o *SecurityMonitoringStandardRuleResponse) SetComplianceSignalOptions(v CloudConfigurationRuleComplianceSignalOptions) {
	o.ComplianceSignalOptions = &v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetCreatedAt() int64 {
	if o == nil || o.CreatedAt == nil {
		var ret int64
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetCreatedAtOk() (*int64, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given int64 and assigns it to the CreatedAt field.
func (o *SecurityMonitoringStandardRuleResponse) SetCreatedAt(v int64) {
	o.CreatedAt = &v
}

// GetCreationAuthorId returns the CreationAuthorId field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetCreationAuthorId() int64 {
	if o == nil || o.CreationAuthorId == nil {
		var ret int64
		return ret
	}
	return *o.CreationAuthorId
}

// GetCreationAuthorIdOk returns a tuple with the CreationAuthorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetCreationAuthorIdOk() (*int64, bool) {
	if o == nil || o.CreationAuthorId == nil {
		return nil, false
	}
	return o.CreationAuthorId, true
}

// HasCreationAuthorId returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasCreationAuthorId() bool {
	return o != nil && o.CreationAuthorId != nil
}

// SetCreationAuthorId gets a reference to the given int64 and assigns it to the CreationAuthorId field.
func (o *SecurityMonitoringStandardRuleResponse) SetCreationAuthorId(v int64) {
	o.CreationAuthorId = &v
}

// GetDeprecationDate returns the DeprecationDate field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetDeprecationDate() int64 {
	if o == nil || o.DeprecationDate == nil {
		var ret int64
		return ret
	}
	return *o.DeprecationDate
}

// GetDeprecationDateOk returns a tuple with the DeprecationDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetDeprecationDateOk() (*int64, bool) {
	if o == nil || o.DeprecationDate == nil {
		return nil, false
	}
	return o.DeprecationDate, true
}

// HasDeprecationDate returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasDeprecationDate() bool {
	return o != nil && o.DeprecationDate != nil
}

// SetDeprecationDate gets a reference to the given int64 and assigns it to the DeprecationDate field.
func (o *SecurityMonitoringStandardRuleResponse) SetDeprecationDate(v int64) {
	o.DeprecationDate = &v
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetFilters() []SecurityMonitoringFilter {
	if o == nil || o.Filters == nil {
		var ret []SecurityMonitoringFilter
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetFiltersOk() (*[]SecurityMonitoringFilter, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasFilters() bool {
	return o != nil && o.Filters != nil
}

// SetFilters gets a reference to the given []SecurityMonitoringFilter and assigns it to the Filters field.
func (o *SecurityMonitoringStandardRuleResponse) SetFilters(v []SecurityMonitoringFilter) {
	o.Filters = v
}

// GetHasExtendedTitle returns the HasExtendedTitle field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetHasExtendedTitle() bool {
	if o == nil || o.HasExtendedTitle == nil {
		var ret bool
		return ret
	}
	return *o.HasExtendedTitle
}

// GetHasExtendedTitleOk returns a tuple with the HasExtendedTitle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetHasExtendedTitleOk() (*bool, bool) {
	if o == nil || o.HasExtendedTitle == nil {
		return nil, false
	}
	return o.HasExtendedTitle, true
}

// HasHasExtendedTitle returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasHasExtendedTitle() bool {
	return o != nil && o.HasExtendedTitle != nil
}

// SetHasExtendedTitle gets a reference to the given bool and assigns it to the HasExtendedTitle field.
func (o *SecurityMonitoringStandardRuleResponse) SetHasExtendedTitle(v bool) {
	o.HasExtendedTitle = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SecurityMonitoringStandardRuleResponse) SetId(v string) {
	o.Id = &v
}

// GetIsDefault returns the IsDefault field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetIsDefault() bool {
	if o == nil || o.IsDefault == nil {
		var ret bool
		return ret
	}
	return *o.IsDefault
}

// GetIsDefaultOk returns a tuple with the IsDefault field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetIsDefaultOk() (*bool, bool) {
	if o == nil || o.IsDefault == nil {
		return nil, false
	}
	return o.IsDefault, true
}

// HasIsDefault returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasIsDefault() bool {
	return o != nil && o.IsDefault != nil
}

// SetIsDefault gets a reference to the given bool and assigns it to the IsDefault field.
func (o *SecurityMonitoringStandardRuleResponse) SetIsDefault(v bool) {
	o.IsDefault = &v
}

// GetIsDeleted returns the IsDeleted field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetIsDeleted() bool {
	if o == nil || o.IsDeleted == nil {
		var ret bool
		return ret
	}
	return *o.IsDeleted
}

// GetIsDeletedOk returns a tuple with the IsDeleted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetIsDeletedOk() (*bool, bool) {
	if o == nil || o.IsDeleted == nil {
		return nil, false
	}
	return o.IsDeleted, true
}

// HasIsDeleted returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasIsDeleted() bool {
	return o != nil && o.IsDeleted != nil
}

// SetIsDeleted gets a reference to the given bool and assigns it to the IsDeleted field.
func (o *SecurityMonitoringStandardRuleResponse) SetIsDeleted(v bool) {
	o.IsDeleted = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasIsEnabled() bool {
	return o != nil && o.IsEnabled != nil
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *SecurityMonitoringStandardRuleResponse) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SecurityMonitoringStandardRuleResponse) SetMessage(v string) {
	o.Message = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringStandardRuleResponse) SetName(v string) {
	o.Name = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetOptions() SecurityMonitoringRuleOptions {
	if o == nil || o.Options == nil {
		var ret SecurityMonitoringRuleOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetOptionsOk() (*SecurityMonitoringRuleOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given SecurityMonitoringRuleOptions and assigns it to the Options field.
func (o *SecurityMonitoringStandardRuleResponse) SetOptions(v SecurityMonitoringRuleOptions) {
	o.Options = &v
}

// GetQueries returns the Queries field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetQueries() []SecurityMonitoringStandardRuleQuery {
	if o == nil || o.Queries == nil {
		var ret []SecurityMonitoringStandardRuleQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetQueriesOk() (*[]SecurityMonitoringStandardRuleQuery, bool) {
	if o == nil || o.Queries == nil {
		return nil, false
	}
	return &o.Queries, true
}

// HasQueries returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasQueries() bool {
	return o != nil && o.Queries != nil
}

// SetQueries gets a reference to the given []SecurityMonitoringStandardRuleQuery and assigns it to the Queries field.
func (o *SecurityMonitoringStandardRuleResponse) SetQueries(v []SecurityMonitoringStandardRuleQuery) {
	o.Queries = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SecurityMonitoringStandardRuleResponse) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetType() SecurityMonitoringRuleTypeRead {
	if o == nil || o.Type == nil {
		var ret SecurityMonitoringRuleTypeRead
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetTypeOk() (*SecurityMonitoringRuleTypeRead, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SecurityMonitoringRuleTypeRead and assigns it to the Type field.
func (o *SecurityMonitoringStandardRuleResponse) SetType(v SecurityMonitoringRuleTypeRead) {
	o.Type = &v
}

// GetUpdateAuthorId returns the UpdateAuthorId field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetUpdateAuthorId() int64 {
	if o == nil || o.UpdateAuthorId == nil {
		var ret int64
		return ret
	}
	return *o.UpdateAuthorId
}

// GetUpdateAuthorIdOk returns a tuple with the UpdateAuthorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetUpdateAuthorIdOk() (*int64, bool) {
	if o == nil || o.UpdateAuthorId == nil {
		return nil, false
	}
	return o.UpdateAuthorId, true
}

// HasUpdateAuthorId returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasUpdateAuthorId() bool {
	return o != nil && o.UpdateAuthorId != nil
}

// SetUpdateAuthorId gets a reference to the given int64 and assigns it to the UpdateAuthorId field.
func (o *SecurityMonitoringStandardRuleResponse) SetUpdateAuthorId(v int64) {
	o.UpdateAuthorId = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringStandardRuleResponse) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringStandardRuleResponse) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringStandardRuleResponse) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SecurityMonitoringStandardRuleResponse) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringStandardRuleResponse) MarshalJSON() ([]byte, error) {
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
	if o.CreatedAt != nil {
		toSerialize["createdAt"] = o.CreatedAt
	}
	if o.CreationAuthorId != nil {
		toSerialize["creationAuthorId"] = o.CreationAuthorId
	}
	if o.DeprecationDate != nil {
		toSerialize["deprecationDate"] = o.DeprecationDate
	}
	if o.Filters != nil {
		toSerialize["filters"] = o.Filters
	}
	if o.HasExtendedTitle != nil {
		toSerialize["hasExtendedTitle"] = o.HasExtendedTitle
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IsDefault != nil {
		toSerialize["isDefault"] = o.IsDefault
	}
	if o.IsDeleted != nil {
		toSerialize["isDeleted"] = o.IsDeleted
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
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.UpdateAuthorId != nil {
		toSerialize["updateAuthorId"] = o.UpdateAuthorId
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
func (o *SecurityMonitoringStandardRuleResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cases                   []SecurityMonitoringRuleCase                   `json:"cases,omitempty"`
		ComplianceSignalOptions *CloudConfigurationRuleComplianceSignalOptions `json:"complianceSignalOptions,omitempty"`
		CreatedAt               *int64                                         `json:"createdAt,omitempty"`
		CreationAuthorId        *int64                                         `json:"creationAuthorId,omitempty"`
		DeprecationDate         *int64                                         `json:"deprecationDate,omitempty"`
		Filters                 []SecurityMonitoringFilter                     `json:"filters,omitempty"`
		HasExtendedTitle        *bool                                          `json:"hasExtendedTitle,omitempty"`
		Id                      *string                                        `json:"id,omitempty"`
		IsDefault               *bool                                          `json:"isDefault,omitempty"`
		IsDeleted               *bool                                          `json:"isDeleted,omitempty"`
		IsEnabled               *bool                                          `json:"isEnabled,omitempty"`
		Message                 *string                                        `json:"message,omitempty"`
		Name                    *string                                        `json:"name,omitempty"`
		Options                 *SecurityMonitoringRuleOptions                 `json:"options,omitempty"`
		Queries                 []SecurityMonitoringStandardRuleQuery          `json:"queries,omitempty"`
		Tags                    []string                                       `json:"tags,omitempty"`
		Type                    *SecurityMonitoringRuleTypeRead                `json:"type,omitempty"`
		UpdateAuthorId          *int64                                         `json:"updateAuthorId,omitempty"`
		Version                 *int64                                         `json:"version,omitempty"`
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
	o.Cases = all.Cases
	if all.ComplianceSignalOptions != nil && all.ComplianceSignalOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ComplianceSignalOptions = all.ComplianceSignalOptions
	o.CreatedAt = all.CreatedAt
	o.CreationAuthorId = all.CreationAuthorId
	o.DeprecationDate = all.DeprecationDate
	o.Filters = all.Filters
	o.HasExtendedTitle = all.HasExtendedTitle
	o.Id = all.Id
	o.IsDefault = all.IsDefault
	o.IsDeleted = all.IsDeleted
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
	o.Type = all.Type
	o.UpdateAuthorId = all.UpdateAuthorId
	o.Version = all.Version
	return nil
}
