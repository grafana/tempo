// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityFilterMeta Optional metadata associated to the response.
type SecurityFilterMeta struct {
	// A warning message.
	Warning *string `json:"warning,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilterMeta instantiates a new SecurityFilterMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilterMeta() *SecurityFilterMeta {
	this := SecurityFilterMeta{}
	return &this
}

// NewSecurityFilterMetaWithDefaults instantiates a new SecurityFilterMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterMetaWithDefaults() *SecurityFilterMeta {
	this := SecurityFilterMeta{}
	return &this
}

// GetWarning returns the Warning field value if set, zero value otherwise.
func (o *SecurityFilterMeta) GetWarning() string {
	if o == nil || o.Warning == nil {
		var ret string
		return ret
	}
	return *o.Warning
}

// GetWarningOk returns a tuple with the Warning field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilterMeta) GetWarningOk() (*string, bool) {
	if o == nil || o.Warning == nil {
		return nil, false
	}
	return o.Warning, true
}

// HasWarning returns a boolean if a field has been set.
func (o *SecurityFilterMeta) HasWarning() bool {
	return o != nil && o.Warning != nil
}

// SetWarning gets a reference to the given string and assigns it to the Warning field.
func (o *SecurityFilterMeta) SetWarning(v string) {
	o.Warning = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilterMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Warning != nil {
		toSerialize["warning"] = o.Warning
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityFilterMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Warning *string `json:"warning,omitempty"`
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
	o.Warning = all.Warning
	return nil
}
