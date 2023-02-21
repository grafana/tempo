// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ScalarMeta Metadata for the resulting numerical values.
type ScalarMeta struct {
	// Detailed information about the unit.
	// First element describes the "primary unit" (for example, `bytes` in `bytes per second`).
	// The second element describes the "per unit" (for example, `second` in `bytes per second`).
	// If the second element is not present, the API returns null.
	Unit []Unit `json:"unit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewScalarMeta instantiates a new ScalarMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewScalarMeta() *ScalarMeta {
	this := ScalarMeta{}
	return &this
}

// NewScalarMetaWithDefaults instantiates a new ScalarMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewScalarMetaWithDefaults() *ScalarMeta {
	this := ScalarMeta{}
	return &this
}

// GetUnit returns the Unit field value if set, zero value otherwise.
func (o *ScalarMeta) GetUnit() []Unit {
	if o == nil || o.Unit == nil {
		var ret []Unit
		return ret
	}
	return o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ScalarMeta) GetUnitOk() (*[]Unit, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return &o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *ScalarMeta) HasUnit() bool {
	return o != nil && o.Unit != nil
}

// SetUnit gets a reference to the given []Unit and assigns it to the Unit field.
func (o *ScalarMeta) SetUnit(v []Unit) {
	o.Unit = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ScalarMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Unit != nil {
		toSerialize["unit"] = o.Unit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ScalarMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Unit []Unit `json:"unit,omitempty"`
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
	return nil
}
