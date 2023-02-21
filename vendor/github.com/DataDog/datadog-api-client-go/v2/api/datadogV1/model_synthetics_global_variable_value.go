// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// SyntheticsGlobalVariableValue Value of the global variable.
type SyntheticsGlobalVariableValue struct {
	// Options for the Global Variable for MFA.
	Options *SyntheticsGlobalVariableOptions `json:"options,omitempty"`
	// Determines if the value of the variable is hidden.
	Secure *bool `json:"secure,omitempty"`
	// Value of the global variable. When reading a global variable,
	// the value will not be present if the variable is hidden with the `secure` property.
	Value *string `json:"value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsGlobalVariableValue instantiates a new SyntheticsGlobalVariableValue object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsGlobalVariableValue() *SyntheticsGlobalVariableValue {
	this := SyntheticsGlobalVariableValue{}
	return &this
}

// NewSyntheticsGlobalVariableValueWithDefaults instantiates a new SyntheticsGlobalVariableValue object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsGlobalVariableValueWithDefaults() *SyntheticsGlobalVariableValue {
	this := SyntheticsGlobalVariableValue{}
	return &this
}

// GetOptions returns the Options field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableValue) GetOptions() SyntheticsGlobalVariableOptions {
	if o == nil || o.Options == nil {
		var ret SyntheticsGlobalVariableOptions
		return ret
	}
	return *o.Options
}

// GetOptionsOk returns a tuple with the Options field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableValue) GetOptionsOk() (*SyntheticsGlobalVariableOptions, bool) {
	if o == nil || o.Options == nil {
		return nil, false
	}
	return o.Options, true
}

// HasOptions returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableValue) HasOptions() bool {
	return o != nil && o.Options != nil
}

// SetOptions gets a reference to the given SyntheticsGlobalVariableOptions and assigns it to the Options field.
func (o *SyntheticsGlobalVariableValue) SetOptions(v SyntheticsGlobalVariableOptions) {
	o.Options = &v
}

// GetSecure returns the Secure field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableValue) GetSecure() bool {
	if o == nil || o.Secure == nil {
		var ret bool
		return ret
	}
	return *o.Secure
}

// GetSecureOk returns a tuple with the Secure field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableValue) GetSecureOk() (*bool, bool) {
	if o == nil || o.Secure == nil {
		return nil, false
	}
	return o.Secure, true
}

// HasSecure returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableValue) HasSecure() bool {
	return o != nil && o.Secure != nil
}

// SetSecure gets a reference to the given bool and assigns it to the Secure field.
func (o *SyntheticsGlobalVariableValue) SetSecure(v bool) {
	o.Secure = &v
}

// GetValue returns the Value field value if set, zero value otherwise.
func (o *SyntheticsGlobalVariableValue) GetValue() string {
	if o == nil || o.Value == nil {
		var ret string
		return ret
	}
	return *o.Value
}

// GetValueOk returns a tuple with the Value field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsGlobalVariableValue) GetValueOk() (*string, bool) {
	if o == nil || o.Value == nil {
		return nil, false
	}
	return o.Value, true
}

// HasValue returns a boolean if a field has been set.
func (o *SyntheticsGlobalVariableValue) HasValue() bool {
	return o != nil && o.Value != nil
}

// SetValue gets a reference to the given string and assigns it to the Value field.
func (o *SyntheticsGlobalVariableValue) SetValue(v string) {
	o.Value = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsGlobalVariableValue) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Options != nil {
		toSerialize["options"] = o.Options
	}
	if o.Secure != nil {
		toSerialize["secure"] = o.Secure
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
func (o *SyntheticsGlobalVariableValue) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Options *SyntheticsGlobalVariableOptions `json:"options,omitempty"`
		Secure  *bool                            `json:"secure,omitempty"`
		Value   *string                          `json:"value,omitempty"`
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
	if all.Options != nil && all.Options.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Options = all.Options
	o.Secure = all.Secure
	o.Value = all.Value
	return nil
}
