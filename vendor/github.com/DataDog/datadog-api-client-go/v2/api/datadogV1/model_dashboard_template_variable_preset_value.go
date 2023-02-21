// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// DashboardTemplateVariablePresetValue Template variables saved views.
type DashboardTemplateVariablePresetValue struct {
	// The name of the variable.
	Name *string `json:"name,omitempty"`
	// (deprecated) The value of the template variable within the saved view. Cannot be used in conjunction with `values`.
	// Deprecated
	Value *string `json:"value,omitempty"`
	// One or many template variable values within the saved view, which will be unioned together using `OR` if more than one is specified. Cannot be used in conjunction with `value`.
	Values []string `json:"values,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewDashboardTemplateVariablePresetValue instantiates a new DashboardTemplateVariablePresetValue object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardTemplateVariablePresetValue() *DashboardTemplateVariablePresetValue {
	this := DashboardTemplateVariablePresetValue{}
	return &this
}

// NewDashboardTemplateVariablePresetValueWithDefaults instantiates a new DashboardTemplateVariablePresetValue object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardTemplateVariablePresetValueWithDefaults() *DashboardTemplateVariablePresetValue {
	this := DashboardTemplateVariablePresetValue{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *DashboardTemplateVariablePresetValue) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardTemplateVariablePresetValue) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *DashboardTemplateVariablePresetValue) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *DashboardTemplateVariablePresetValue) SetName(v string) {
	o.Name = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
// Deprecated
func (o *DashboardTemplateVariablePresetValue) GetValue() string {
	if o == nil || o.Value == nil {
		var ret string
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *DashboardTemplateVariablePresetValue) GetValueOk() (*string, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *DashboardTemplateVariablePresetValue) HasValue() bool {
	return o != nil && o.Value != nil
}

// SetValue gets a reference to the given string and assigns it to the Value field.
// Deprecated
func (o *DashboardTemplateVariablePresetValue) SetValue(v string) {
	o.Value = &v
}

// GetValues returns the Values field value if set, zero value otherwise.
func (o *DashboardTemplateVariablePresetValue) GetValues() []string {
	if o == nil || o.Values == nil {
		var ret []string
		return ret
	}
	return o.Values
}

// GetValuesOk returns a tuple with the Values field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardTemplateVariablePresetValue) GetValuesOk() (*[]string, bool) {
	if o == nil || o.Values == nil {
		return nil, false
	}
	return &o.Values, true
}

// HasValues returns a boolean if a field has been set.
func (o *DashboardTemplateVariablePresetValue) HasValues() bool {
	return o != nil && o.Values != nil
}

// SetValues gets a reference to the given []string and assigns it to the Values field.
func (o *DashboardTemplateVariablePresetValue) SetValues(v []string) {
	o.Values = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardTemplateVariablePresetValue) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}
	if o.Values != nil {
		toSerialize["values"] = o.Values
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardTemplateVariablePresetValue) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Name   *string  `json:"name,omitempty"`
		Value  *string  `json:"value,omitempty"`
		Values []string `json:"values,omitempty"`
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
	o.Name = all.Name
	o.Value = all.Value
	o.Values = all.Values
	return nil
}
