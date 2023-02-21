// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalRuleResponse Rule.
type SecurityMonitoringSignalRuleResponse struct {
	// Cases for generating signals.
	Cases []SecurityMonitoringRuleCase `json:"cases,omitempty"`
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
	Queries []SecurityMonitoringSignalRuleResponseQuery `json:"queries,omitempty"`
	// Tags for generated signals.
	Tags []string `json:"tags,omitempty"`
	// The rule type.
	Type *SecurityMonitoringSignalRuleType `json:"type,omitempty"`
	// User ID of the user who updated the rule.
	UpdateAuthorId *int64 `json:"updateAuthorId,omitempty"`
	// The version of the rule.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalRuleResponse instantiates a new SecurityMonitoringSignalRuleResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalRuleResponse() *SecurityMonitoringSignalRuleResponse {
	this := SecurityMonitoringSignalRuleResponse{}
	return &this
}

// NewSecurityMonitoringSignalRuleResponseWithDefaults instantiates a new SecurityMonitoringSignalRuleResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalRuleResponseWithDefaults() *SecurityMonitoringSignalRuleResponse {
	this := SecurityMonitoringSignalRuleResponse{}
	return &this
}

// GetCases returns the Cases field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetCases() []SecurityMonitoringRuleCase {
	if o == nil || o.Cases == nil {
		var ret []SecurityMonitoringRuleCase
		return ret
	}
	return o.Cases
}

// GetCasesOk returns a tuple with the Cases field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetCasesOk() (*[]SecurityMonitoringRuleCase, bool) {
	if o == nil || o.Cases == nil {
		return nil, false
	}
	return &o.Cases, true
}

// HasCases returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasCases() bool {
	return o != nil && o.Cases != nil
}

// SetCases gets a reference to the given []SecurityMonitoringRuleCase and assigns it to the Cases field.
func (o *SecurityMonitoringSignalRuleResponse) SetCases(v []SecurityMonitoringRuleCase) {
	o.Cases = v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetCreatedAt() int64 {
	if o == nil || o.CreatedAt == nil {
		var ret int64
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetCreatedAtOk() (*int64, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given int64 and assigns it to the CreatedAt field.
func (o *SecurityMonitoringSignalRuleResponse) SetCreatedAt(v int64) {
	o.CreatedAt = &v
}

// GetCreationAuthorId returns the CreationAuthorId field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetCreationAuthorId() int64 {
	if o == nil || o.CreationAuthorId == nil {
		var ret int64
		return ret
	}
	return *o.CreationAuthorId
}

// GetCreationAuthorIdOk returns a tuple with the CreationAuthorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetCreationAuthorIdOk() (*int64, bool) {
	if o == nil || o.CreationAuthorId == nil {
		return nil, false
	}
	return o.CreationAuthorId, true
}

// HasCreationAuthorId returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasCreationAuthorId() bool {
	return o != nil && o.CreationAuthorId != nil
}

// SetCreationAuthorId gets a reference to the given int64 and assigns it to the CreationAuthorId field.
func (o *SecurityMonitoringSignalRuleResponse) SetCreationAuthorId(v int64) {
	o.CreationAuthorId = &v
}

// GetDeprecationDate returns the DeprecationDate field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetDeprecationDate() int64 {
	if o == nil || o.DeprecationDate == nil {
		var ret int64
		return ret
	}
	return *o.DeprecationDate
}

// GetDeprecationDateOk returns a tuple with the DeprecationDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetDeprecationDateOk() (*int64, bool) {
	if o == nil || o.DeprecationDate == nil {
		return nil, false
	}
	return o.DeprecationDate, true
}

// HasDeprecationDate returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasDeprecationDate() bool {
	return o != nil && o.DeprecationDate != nil
}

// SetDeprecationDate gets a reference to the given int64 and assigns it to the DeprecationDate field.
func (o *SecurityMonitoringSignalRuleResponse) SetDeprecationDate(v int64) {
	o.DeprecationDate = &v
}

// GetFilters returns the Filters field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetFilters() []SecurityMonitoringFilter {
	if o == nil || o.Filters == nil {
		var ret []SecurityMonitoringFilter
		return ret
	}
	return o.Filters
}

// GetFiltersOk returns a tuple with the Filters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetFiltersOk() (*[]SecurityMonitoringFilter, bool) {
	if o == nil || o.Filters == nil {
		return nil, false
	}
	return &o.Filters, true
}

// HasFilters returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasFilters() bool {
	return o != nil && o.Filters != nil
}

// SetFilters gets a reference to the given []SecurityMonitoringFilter and assigns it to the Filters field.
func (o *SecurityMonitoringSignalRuleResponse) SetFilters(v []SecurityMonitoringFilter) {
	o.Filters = v
}

// GetHasExtendedTitle returns the HasExtendedTitle field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetHasExtendedTitle() bool {
	if o == nil || o.HasExtendedTitle == nil {
		var ret bool
		return ret
	}
	return *o.HasExtendedTitle
}

// GetHasExtendedTitleOk returns a tuple with the HasExtendedTitle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetHasExtendedTitleOk() (*bool, bool) {
	if o == nil || o.HasExtendedTitle == nil {
		return nil, false
	}
	return o.HasExtendedTitle, true
}

// HasHasExtendedTitle returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasHasExtendedTitle() bool {
	return o != nil && o.HasExtendedTitle != nil
}

// SetHasExtendedTitle gets a reference to the given bool and assigns it to the HasExtendedTitle field.
func (o *SecurityMonitoringSignalRuleResponse) SetHasExtendedTitle(v bool) {
	o.HasExtendedTitle = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SecurityMonitoringSignalRuleResponse) SetId(v string) {
	o.Id = &v
}

// GetIsDefault returns the IsDefault field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetIsDefault() bool {
	if o == nil || o.IsDefault == nil {
		var ret bool
		return ret
	}
	return *o.IsDefault
}

// GetIsDefaultOk returns a tuple with the IsDefault field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetIsDefaultOk() (*bool, bool) {
	if o == nil || o.IsDefault == nil {
		return nil, false
	}
	return o.IsDefault, true
}

// HasIsDefault returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasIsDefault() bool {
	return o != nil && o.IsDefault != nil
}

// SetIsDefault gets a reference to the given bool and assigns it to the IsDefault field.
func (o *SecurityMonitoringSignalRuleResponse) SetIsDefault(v bool) {
	o.IsDefault = &v
}

// GetIsDeleted returns the IsDeleted field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetIsDeleted() bool {
	if o == nil || o.IsDeleted == nil {
		var ret bool
		return ret
	}
	return *o.IsDeleted
}

// GetIsDeletedOk returns a tuple with the IsDeleted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetIsDeletedOk() (*bool, bool) {
	if o == nil || o.IsDeleted == nil {
		return nil, false
	}
	return o.IsDeleted, true
}

// HasIsDeleted returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasIsDeleted() bool {
	return o != nil && o.IsDeleted != nil
}

// SetIsDeleted gets a reference to the given bool and assigns it to the IsDeleted field.
func (o *SecurityMonitoringSignalRuleResponse) SetIsDeleted(v bool) {
	o.IsDeleted = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasIsEnabled() bool {
	return o != nil && o.IsEnabled != nil
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *SecurityMonitoringSignalRuleResponse) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetMessage returns the Message field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetMessage() string {
	if o == nil || o.Message == nil {
		var ret string
		return ret
	}
	return *o.Message
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetMessageOk() (*string, bool) {
	if o == nil || o.Message == nil {
		return nil, false
	}
	return o.Message, true
}

// HasMessage returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasMessage() bool {
	return o != nil && o.Message != nil
}

// SetMessage gets a reference to the given string and assigns it to the Message field.
func (o *SecurityMonitoringSignalRuleResponse) SetMessage(v string) {
	o.Message = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringSignalRuleResponse) SetName(v string) {
	o.Name = &v
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetOptions() SecurityMonitoringRuleOptions {
	if o == nil || o.Options == nil {
		var ret SecurityMonitoringRuleOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetOptionsOk() (*SecurityMonitoringRuleOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given SecurityMonitoringRuleOptions and assigns it to the Options field.
func (o *SecurityMonitoringSignalRuleResponse) SetOptions(v SecurityMonitoringRuleOptions) {
	o.Options = &v
}

// GetQueries returns the Queries field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetQueries() []SecurityMonitoringSignalRuleResponseQuery {
	if o == nil || o.Queries == nil {
		var ret []SecurityMonitoringSignalRuleResponseQuery
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetQueriesOk() (*[]SecurityMonitoringSignalRuleResponseQuery, bool) {
	if o == nil || o.Queries == nil {
		return nil, false
	}
	return &o.Queries, true
}

// HasQueries returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasQueries() bool {
	return o != nil && o.Queries != nil
}

// SetQueries gets a reference to the given []SecurityMonitoringSignalRuleResponseQuery and assigns it to the Queries field.
func (o *SecurityMonitoringSignalRuleResponse) SetQueries(v []SecurityMonitoringSignalRuleResponseQuery) {
	o.Queries = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *SecurityMonitoringSignalRuleResponse) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetType() SecurityMonitoringSignalRuleType {
	if o == nil || o.Type == nil {
		var ret SecurityMonitoringSignalRuleType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetTypeOk() (*SecurityMonitoringSignalRuleType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SecurityMonitoringSignalRuleType and assigns it to the Type field.
func (o *SecurityMonitoringSignalRuleResponse) SetType(v SecurityMonitoringSignalRuleType) {
	o.Type = &v
}

// GetUpdateAuthorId returns the UpdateAuthorId field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetUpdateAuthorId() int64 {
	if o == nil || o.UpdateAuthorId == nil {
		var ret int64
		return ret
	}
	return *o.UpdateAuthorId
}

// GetUpdateAuthorIdOk returns a tuple with the UpdateAuthorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetUpdateAuthorIdOk() (*int64, bool) {
	if o == nil || o.UpdateAuthorId == nil {
		return nil, false
	}
	return o.UpdateAuthorId, true
}

// HasUpdateAuthorId returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasUpdateAuthorId() bool {
	return o != nil && o.UpdateAuthorId != nil
}

// SetUpdateAuthorId gets a reference to the given int64 and assigns it to the UpdateAuthorId field.
func (o *SecurityMonitoringSignalRuleResponse) SetUpdateAuthorId(v int64) {
	o.UpdateAuthorId = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponse) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponse) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponse) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SecurityMonitoringSignalRuleResponse) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalRuleResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cases != nil {
		toSerialize["cases"] = o.Cases
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
func (o *SecurityMonitoringSignalRuleResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cases            []SecurityMonitoringRuleCase                `json:"cases,omitempty"`
		CreatedAt        *int64                                      `json:"createdAt,omitempty"`
		CreationAuthorId *int64                                      `json:"creationAuthorId,omitempty"`
		DeprecationDate  *int64                                      `json:"deprecationDate,omitempty"`
		Filters          []SecurityMonitoringFilter                  `json:"filters,omitempty"`
		HasExtendedTitle *bool                                       `json:"hasExtendedTitle,omitempty"`
		Id               *string                                     `json:"id,omitempty"`
		IsDefault        *bool                                       `json:"isDefault,omitempty"`
		IsDeleted        *bool                                       `json:"isDeleted,omitempty"`
		IsEnabled        *bool                                       `json:"isEnabled,omitempty"`
		Message          *string                                     `json:"message,omitempty"`
		Name             *string                                     `json:"name,omitempty"`
		Options          *SecurityMonitoringRuleOptions              `json:"options,omitempty"`
		Queries          []SecurityMonitoringSignalRuleResponseQuery `json:"queries,omitempty"`
		Tags             []string                                    `json:"tags,omitempty"`
		Type             *SecurityMonitoringSignalRuleType           `json:"type,omitempty"`
		UpdateAuthorId   *int64                                      `json:"updateAuthorId,omitempty"`
		Version          *int64                                      `json:"version,omitempty"`
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
