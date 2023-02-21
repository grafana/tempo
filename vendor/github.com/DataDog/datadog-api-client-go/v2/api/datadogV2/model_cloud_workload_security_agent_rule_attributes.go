// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CloudWorkloadSecurityAgentRuleAttributes A Cloud Workload Security Agent rule returned by the API.
type CloudWorkloadSecurityAgentRuleAttributes struct {
	// The category of the Agent rule.
	Category *string `json:"category,omitempty"`
	// When the Agent rule was created, timestamp in milliseconds.
	CreationDate *int64 `json:"creationDate,omitempty"`
	// The attributes of the user who created the Agent rule.
	Creator *CloudWorkloadSecurityAgentRuleCreatorAttributes `json:"creator,omitempty"`
	// Whether the rule is included by default.
	DefaultRule *bool `json:"defaultRule,omitempty"`
	// The description of the Agent rule.
	Description *string `json:"description,omitempty"`
	// Whether the Agent rule is enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// The SECL expression of the Agent rule.
	Expression *string `json:"expression,omitempty"`
	// The name of the Agent rule.
	Name *string `json:"name,omitempty"`
	// When the Agent rule was last updated, timestamp in milliseconds.
	UpdatedAt *int64 `json:"updatedAt,omitempty"`
	// The attributes of the user who last updated the Agent rule.
	Updater *CloudWorkloadSecurityAgentRuleUpdaterAttributes `json:"updater,omitempty"`
	// The version of the Agent rule.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCloudWorkloadSecurityAgentRuleAttributes instantiates a new CloudWorkloadSecurityAgentRuleAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCloudWorkloadSecurityAgentRuleAttributes() *CloudWorkloadSecurityAgentRuleAttributes {
	this := CloudWorkloadSecurityAgentRuleAttributes{}
	return &this
}

// NewCloudWorkloadSecurityAgentRuleAttributesWithDefaults instantiates a new CloudWorkloadSecurityAgentRuleAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCloudWorkloadSecurityAgentRuleAttributesWithDefaults() *CloudWorkloadSecurityAgentRuleAttributes {
	this := CloudWorkloadSecurityAgentRuleAttributes{}
	return &this
}

// GetCategory returns the Category field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCategory() string {
	if o == nil || o.Category == nil {
		var ret string
		return ret
	}
	return *o.Category
}

// GetCategoryOk returns a tuple with the Category field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCategoryOk() (*string, bool) {
	if o == nil || o.Category == nil {
		return nil, false
	}
	return o.Category, true
}

// HasCategory returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasCategory() bool {
	return o != nil && o.Category != nil
}

// SetCategory gets a reference to the given string and assigns it to the Category field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetCategory(v string) {
	o.Category = &v
}

// GetCreationDate returns the CreationDate field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCreationDate() int64 {
	if o == nil || o.CreationDate == nil {
		var ret int64
		return ret
	}
	return *o.CreationDate
}

// GetCreationDateOk returns a tuple with the CreationDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCreationDateOk() (*int64, bool) {
	if o == nil || o.CreationDate == nil {
		return nil, false
	}
	return o.CreationDate, true
}

// HasCreationDate returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasCreationDate() bool {
	return o != nil && o.CreationDate != nil
}

// SetCreationDate gets a reference to the given int64 and assigns it to the CreationDate field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetCreationDate(v int64) {
	o.CreationDate = &v
}

// GetCreator returns the Creator field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCreator() CloudWorkloadSecurityAgentRuleCreatorAttributes {
	if o == nil || o.Creator == nil {
		var ret CloudWorkloadSecurityAgentRuleCreatorAttributes
		return ret
	}
	return *o.Creator
}

// GetCreatorOk returns a tuple with the Creator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetCreatorOk() (*CloudWorkloadSecurityAgentRuleCreatorAttributes, bool) {
	if o == nil || o.Creator == nil {
		return nil, false
	}
	return o.Creator, true
}

// HasCreator returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasCreator() bool {
	return o != nil && o.Creator != nil
}

// SetCreator gets a reference to the given CloudWorkloadSecurityAgentRuleCreatorAttributes and assigns it to the Creator field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetCreator(v CloudWorkloadSecurityAgentRuleCreatorAttributes) {
	o.Creator = &v
}

// GetDefaultRule returns the DefaultRule field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetDefaultRule() bool {
	if o == nil || o.DefaultRule == nil {
		var ret bool
		return ret
	}
	return *o.DefaultRule
}

// GetDefaultRuleOk returns a tuple with the DefaultRule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetDefaultRuleOk() (*bool, bool) {
	if o == nil || o.DefaultRule == nil {
		return nil, false
	}
	return o.DefaultRule, true
}

// HasDefaultRule returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasDefaultRule() bool {
	return o != nil && o.DefaultRule != nil
}

// SetDefaultRule gets a reference to the given bool and assigns it to the DefaultRule field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetDefaultRule(v bool) {
	o.DefaultRule = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasDescription() bool {
	return o != nil && o.Description != nil
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasEnabled() bool {
	return o != nil && o.Enabled != nil
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetEnabled(v bool) {
	o.Enabled = &v
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetExpression() string {
	if o == nil || o.Expression == nil {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetExpressionOk() (*string, bool) {
	if o == nil || o.Expression == nil {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasExpression() bool {
	return o != nil && o.Expression != nil
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetExpression(v string) {
	o.Expression = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetName(v string) {
	o.Name = &v
}

// GetUpdatedAt returns the UpdatedAt field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetUpdatedAt() int64 {
	if o == nil || o.UpdatedAt == nil {
		var ret int64
		return ret
	}
	return *o.UpdatedAt
}

// GetUpdatedAtOk returns a tuple with the UpdatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetUpdatedAtOk() (*int64, bool) {
	if o == nil || o.UpdatedAt == nil {
		return nil, false
	}
	return o.UpdatedAt, true
}

// HasUpdatedAt returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasUpdatedAt() bool {
	return o != nil && o.UpdatedAt != nil
}

// SetUpdatedAt gets a reference to the given int64 and assigns it to the UpdatedAt field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetUpdatedAt(v int64) {
	o.UpdatedAt = &v
}

// GetUpdater returns the Updater field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetUpdater() CloudWorkloadSecurityAgentRuleUpdaterAttributes {
	if o == nil || o.Updater == nil {
		var ret CloudWorkloadSecurityAgentRuleUpdaterAttributes
		return ret
	}
	return *o.Updater
}

// GetUpdaterOk returns a tuple with the Updater field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetUpdaterOk() (*CloudWorkloadSecurityAgentRuleUpdaterAttributes, bool) {
	if o == nil || o.Updater == nil {
		return nil, false
	}
	return o.Updater, true
}

// HasUpdater returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasUpdater() bool {
	return o != nil && o.Updater != nil
}

// SetUpdater gets a reference to the given CloudWorkloadSecurityAgentRuleUpdaterAttributes and assigns it to the Updater field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetUpdater(v CloudWorkloadSecurityAgentRuleUpdaterAttributes) {
	o.Updater = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *CloudWorkloadSecurityAgentRuleAttributes) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *CloudWorkloadSecurityAgentRuleAttributes) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CloudWorkloadSecurityAgentRuleAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Category != nil {
		toSerialize["category"] = o.Category
	}
	if o.CreationDate != nil {
		toSerialize["creationDate"] = o.CreationDate
	}
	if o.Creator != nil {
		toSerialize["creator"] = o.Creator
	}
	if o.DefaultRule != nil {
		toSerialize["defaultRule"] = o.DefaultRule
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
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.UpdatedAt != nil {
		toSerialize["updatedAt"] = o.UpdatedAt
	}
	if o.Updater != nil {
		toSerialize["updater"] = o.Updater
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
func (o *CloudWorkloadSecurityAgentRuleAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Category     *string                                          `json:"category,omitempty"`
		CreationDate *int64                                           `json:"creationDate,omitempty"`
		Creator      *CloudWorkloadSecurityAgentRuleCreatorAttributes `json:"creator,omitempty"`
		DefaultRule  *bool                                            `json:"defaultRule,omitempty"`
		Description  *string                                          `json:"description,omitempty"`
		Enabled      *bool                                            `json:"enabled,omitempty"`
		Expression   *string                                          `json:"expression,omitempty"`
		Name         *string                                          `json:"name,omitempty"`
		UpdatedAt    *int64                                           `json:"updatedAt,omitempty"`
		Updater      *CloudWorkloadSecurityAgentRuleUpdaterAttributes `json:"updater,omitempty"`
		Version      *int64                                           `json:"version,omitempty"`
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
	o.Category = all.Category
	o.CreationDate = all.CreationDate
	if all.Creator != nil && all.Creator.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Creator = all.Creator
	o.DefaultRule = all.DefaultRule
	o.Description = all.Description
	o.Enabled = all.Enabled
	o.Expression = all.Expression
	o.Name = all.Name
	o.UpdatedAt = all.UpdatedAt
	if all.Updater != nil && all.Updater.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Updater = all.Updater
	o.Version = all.Version
	return nil
}
