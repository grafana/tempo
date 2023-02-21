// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SLORawErrorBudgetRemaining Error budget remaining for an SLO.
type SLORawErrorBudgetRemaining struct {
	// Error budget remaining unit.
	Unit *string `json:"unit,omitempty"`
	// Error budget remaining value.
	Value *float64 `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSLORawErrorBudgetRemaining instantiates a new SLORawErrorBudgetRemaining object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLORawErrorBudgetRemaining() *SLORawErrorBudgetRemaining {
	this := SLORawErrorBudgetRemaining{}
	return &this
}

// NewSLORawErrorBudgetRemainingWithDefaults instantiates a new SLORawErrorBudgetRemaining object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLORawErrorBudgetRemainingWithDefaults() *SLORawErrorBudgetRemaining {
	this := SLORawErrorBudgetRemaining{}
	return &this
}

// GetUnit returns the Unit field value if set, zero value otherwise.
func (o *SLORawErrorBudgetRemaining) GetUnit() string {
	if o == nil || o.Unit == nil {
		var ret string
		return ret
	}
	return *o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLORawErrorBudgetRemaining) GetUnitOk() (*string, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *SLORawErrorBudgetRemaining) HasUnit() bool {
	return o != nil && o.Unit != nil
}

// SetUnit gets a reference to the given string and assigns it to the Unit field.
func (o *SLORawErrorBudgetRemaining) SetUnit(v string) {
	o.Unit = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *SLORawErrorBudgetRemaining) GetValue() float64 {
	if o == nil || o.Value == nil {
		var ret float64
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLORawErrorBudgetRemaining) GetValueOk() (*float64, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *SLORawErrorBudgetRemaining) HasValue() bool {
	return o != nil && o.Value != nil
}

// SetValue gets a reference to the given float64 and assigns it to the Value field.
func (o *SLORawErrorBudgetRemaining) SetValue(v float64) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLORawErrorBudgetRemaining) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Unit != nil {
		toSerialize["unit"] = o.Unit
	}
	if o.Value != nil {
		toSerialize["value"] = o.Value
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLORawErrorBudgetRemaining) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Unit  *string  `json:"unit,omitempty"`
		Value *float64 `json:"value,omitempty"`
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
	o.Unit = all.Unit
	o.Value = all.Value
	return nil
}

// NullableSLORawErrorBudgetRemaining handles when a null is used for SLORawErrorBudgetRemaining.
type NullableSLORawErrorBudgetRemaining struct {
	value *SLORawErrorBudgetRemaining
	isSet bool
}

// Get returns the associated value.
func (v NullableSLORawErrorBudgetRemaining) Get() *SLORawErrorBudgetRemaining {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLORawErrorBudgetRemaining) Set(val *SLORawErrorBudgetRemaining) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLORawErrorBudgetRemaining) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSLORawErrorBudgetRemaining) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLORawErrorBudgetRemaining initializes the struct as if Set has been called.
func NewNullableSLORawErrorBudgetRemaining(val *SLORawErrorBudgetRemaining) *NullableSLORawErrorBudgetRemaining {
	return &NullableSLORawErrorBudgetRemaining{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLORawErrorBudgetRemaining) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLORawErrorBudgetRemaining) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
