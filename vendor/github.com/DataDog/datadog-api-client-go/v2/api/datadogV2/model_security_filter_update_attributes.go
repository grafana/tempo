// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityFilterUpdateAttributes The security filters properties to be updated.
type SecurityFilterUpdateAttributes struct {
	// Exclusion filters to exclude some logs from the security filter.
	ExclusionFilters []SecurityFilterExclusionFilter `json:"exclusion_filters,omitempty"`
	// The filtered data type.
	FilteredDataType *SecurityFilterFilteredDataType `json:"filtered_data_type,omitempty"`
	// Whether the security filter is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// The name of the security filter.
	Name *string `json:"name,omitempty"`
	// The query of the security filter.
	Query *string `json:"query,omitempty"`
	// The version of the security filter to update.
	Version *int32 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilterUpdateAttributes instantiates a new SecurityFilterUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilterUpdateAttributes() *SecurityFilterUpdateAttributes {
	this := SecurityFilterUpdateAttributes{}
	return &this
}

// NewSecurityFilterUpdateAttributesWithDefaults instantiates a new SecurityFilterUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterUpdateAttributesWithDefaults() *SecurityFilterUpdateAttributes {
	this := SecurityFilterUpdateAttributes{}
	return &this
}

// GetExclusionFilters returns the ExclusionFilters field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetExclusionFilters() []SecurityFilterExclusionFilter {
	if o == nil || o.ExclusionFilters == nil {
		var ret []SecurityFilterExclusionFilter
		return ret
	}
	return o.ExclusionFilters
}

// GetExclusionFiltersOk returns a tuple with the ExclusionFilters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetExclusionFiltersOk() (*[]SecurityFilterExclusionFilter, bool) {
	if o == nil || o.ExclusionFilters == nil {
		return nil, false
	}
	return &o.ExclusionFilters, true
}

// HasExclusionFilters returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasExclusionFilters() bool {
	return o != nil && o.ExclusionFilters != nil
}

// SetExclusionFilters gets a reference to the given []SecurityFilterExclusionFilter and assigns it to the ExclusionFilters field.
func (o *SecurityFilterUpdateAttributes) SetExclusionFilters(v []SecurityFilterExclusionFilter) {
	o.ExclusionFilters = v
}

// GetFilteredDataType returns the FilteredDataType field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetFilteredDataType() SecurityFilterFilteredDataType {
	if o == nil || o.FilteredDataType == nil {
		var ret SecurityFilterFilteredDataType
		return ret
	}
	return *o.FilteredDataType
}

// GetFilteredDataTypeOk returns a tuple with the FilteredDataType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetFilteredDataTypeOk() (*SecurityFilterFilteredDataType, bool) {
	if o == nil || o.FilteredDataType == nil {
		return nil, false
	}
	return o.FilteredDataType, true
}

// HasFilteredDataType returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasFilteredDataType() bool {
	return o != nil && o.FilteredDataType != nil
}

// SetFilteredDataType gets a reference to the given SecurityFilterFilteredDataType and assigns it to the FilteredDataType field.
func (o *SecurityFilterUpdateAttributes) SetFilteredDataType(v SecurityFilterFilteredDataType) {
	o.FilteredDataType = &v
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasIsEnabled() bool {
	return o != nil && o.IsEnabled != nil
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *SecurityFilterUpdateAttributes) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityFilterUpdateAttributes) SetName(v string) {
	o.Name = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *SecurityFilterUpdateAttributes) SetQuery(v string) {
	o.Query = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityFilterUpdateAttributes) GetVersion() int32 {
	if o == nil || o.Version == nil {
		var ret int32
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterUpdateAttributes) GetVersionOk() (*int32, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityFilterUpdateAttributes) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int32 and assigns it to the Version field.
func (o *SecurityFilterUpdateAttributes) SetVersion(v int32) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilterUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ExclusionFilters != nil {
		toSerialize["exclusion_filters"] = o.ExclusionFilters
	}
	if o.FilteredDataType != nil {
		toSerialize["filtered_data_type"] = o.FilteredDataType
	}
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
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
func (o *SecurityFilterUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ExclusionFilters []SecurityFilterExclusionFilter `json:"exclusion_filters,omitempty"`
		FilteredDataType *SecurityFilterFilteredDataType `json:"filtered_data_type,omitempty"`
		IsEnabled        *bool                           `json:"is_enabled,omitempty"`
		Name             *string                         `json:"name,omitempty"`
		Query            *string                         `json:"query,omitempty"`
		Version          *int32                          `json:"version,omitempty"`
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
	if v := all.FilteredDataType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.ExclusionFilters = all.ExclusionFilters
	o.FilteredDataType = all.FilteredDataType
	o.IsEnabled = all.IsEnabled
	o.Name = all.Name
	o.Query = all.Query
	o.Version = all.Version
	return nil
}
