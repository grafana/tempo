// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityFilterCreateAttributes Object containing the attributes of the security filter to be created.
type SecurityFilterCreateAttributes struct {
	// Exclusion filters to exclude some logs from the security filter.
	ExclusionFilters []SecurityFilterExclusionFilter `json:"exclusion_filters"`
	// The filtered data type.
	FilteredDataType SecurityFilterFilteredDataType `json:"filtered_data_type"`
	// Whether the security filter is enabled.
	IsEnabled bool `json:"is_enabled"`
	// The name of the security filter.
	Name string `json:"name"`
	// The query of the security filter.
	Query string `json:"query"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilterCreateAttributes instantiates a new SecurityFilterCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilterCreateAttributes(exclusionFilters []SecurityFilterExclusionFilter, filteredDataType SecurityFilterFilteredDataType, isEnabled bool, name string, query string) *SecurityFilterCreateAttributes {
	this := SecurityFilterCreateAttributes{}
	this.ExclusionFilters = exclusionFilters
	this.FilteredDataType = filteredDataType
	this.IsEnabled = isEnabled
	this.Name = name
	this.Query = query
	return &this
}

// NewSecurityFilterCreateAttributesWithDefaults instantiates a new SecurityFilterCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterCreateAttributesWithDefaults() *SecurityFilterCreateAttributes {
	this := SecurityFilterCreateAttributes{}
	return &this
}

// GetExclusionFilters returns the ExclusionFilters field value.
func (o *SecurityFilterCreateAttributes) GetExclusionFilters() []SecurityFilterExclusionFilter {
	if o == nil {
		var ret []SecurityFilterExclusionFilter
		return ret
	}
	return o.ExclusionFilters
}

// GetExclusionFiltersOk returns a tuple with the ExclusionFilters field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterCreateAttributes) GetExclusionFiltersOk() (*[]SecurityFilterExclusionFilter, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ExclusionFilters, true
}

// SetExclusionFilters sets field value.
func (o *SecurityFilterCreateAttributes) SetExclusionFilters(v []SecurityFilterExclusionFilter) {
	o.ExclusionFilters = v
}

// GetFilteredDataType returns the FilteredDataType field value.
func (o *SecurityFilterCreateAttributes) GetFilteredDataType() SecurityFilterFilteredDataType {
	if o == nil {
		var ret SecurityFilterFilteredDataType
		return ret
	}
	return o.FilteredDataType
}

// GetFilteredDataTypeOk returns a tuple with the FilteredDataType field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterCreateAttributes) GetFilteredDataTypeOk() (*SecurityFilterFilteredDataType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.FilteredDataType, true
}

// SetFilteredDataType sets field value.
func (o *SecurityFilterCreateAttributes) SetFilteredDataType(v SecurityFilterFilteredDataType) {
	o.FilteredDataType = v
}

// GetIsEnabled returns the IsEnabled field value.
func (o *SecurityFilterCreateAttributes) GetIsEnabled() bool {
	if o == nil {
		var ret bool
		return ret
	}
	return o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterCreateAttributes) GetIsEnabledOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IsEnabled, true
}

// SetIsEnabled sets field value.
func (o *SecurityFilterCreateAttributes) SetIsEnabled(v bool) {
	o.IsEnabled = v
}

// GetName returns the Name field value.
func (o *SecurityFilterCreateAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterCreateAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SecurityFilterCreateAttributes) SetName(v string) {
	o.Name = v
}

// GetQuery returns the Query field value.
func (o *SecurityFilterCreateAttributes) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *SecurityFilterCreateAttributes) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *SecurityFilterCreateAttributes) SetQuery(v string) {
	o.Query = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilterCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["exclusion_filters"] = o.ExclusionFilters
	toSerialize["filtered_data_type"] = o.FilteredDataType
	toSerialize["is_enabled"] = o.IsEnabled
	toSerialize["name"] = o.Name
	toSerialize["query"] = o.Query

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityFilterCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ExclusionFilters *[]SecurityFilterExclusionFilter `json:"exclusion_filters"`
		FilteredDataType *SecurityFilterFilteredDataType  `json:"filtered_data_type"`
		IsEnabled        *bool                            `json:"is_enabled"`
		Name             *string                          `json:"name"`
		Query            *string                          `json:"query"`
	}{}
	all := struct {
		ExclusionFilters []SecurityFilterExclusionFilter `json:"exclusion_filters"`
		FilteredDataType SecurityFilterFilteredDataType  `json:"filtered_data_type"`
		IsEnabled        bool                            `json:"is_enabled"`
		Name             string                          `json:"name"`
		Query            string                          `json:"query"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ExclusionFilters == nil {
		return fmt.Errorf("required field exclusion_filters missing")
	}
	if required.FilteredDataType == nil {
		return fmt.Errorf("required field filtered_data_type missing")
	}
	if required.IsEnabled == nil {
		return fmt.Errorf("required field is_enabled missing")
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Query == nil {
		return fmt.Errorf("required field query missing")
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
	if v := all.FilteredDataType; !v.IsValid() {
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
	return nil
}
