// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// RolesResponse Response containing information about multiple roles.
type RolesResponse struct {
	// Array of returned roles.
	Data []Role `json:"data,omitempty"`
	// Object describing meta attributes of response.
	Meta *ResponseMetaAttributes `json:"meta,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRolesResponse instantiates a new RolesResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRolesResponse() *RolesResponse {
	this := RolesResponse{}
	return &this
}

// NewRolesResponseWithDefaults instantiates a new RolesResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRolesResponseWithDefaults() *RolesResponse {
	this := RolesResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *RolesResponse) GetData() []Role {
	if o == nil || o.Data == nil {
		var ret []Role
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RolesResponse) GetDataOk() (*[]Role, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *RolesResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []Role and assigns it to the Data field.
func (o *RolesResponse) SetData(v []Role) {
	o.Data = v
}

// GetMeta returns the Meta field value if set, zero value otherwise.
func (o *RolesResponse) GetMeta() ResponseMetaAttributes {
	if o == nil || o.Meta == nil {
		var ret ResponseMetaAttributes
		return ret
	}
	return *o.Meta
}

// GetMetaOk returns a tuple with the Meta field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RolesResponse) GetMetaOk() (*ResponseMetaAttributes, bool) {
	if o == nil || o.Meta == nil {
		return nil, false
	}
	return o.Meta, true
}

// HasMeta returns a boolean if a field has been set.
func (o *RolesResponse) HasMeta() bool {
	return o != nil && o.Meta != nil
}

// SetMeta gets a reference to the given ResponseMetaAttributes and assigns it to the Meta field.
func (o *RolesResponse) SetMeta(v ResponseMetaAttributes) {
	o.Meta = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o RolesResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Meta != nil {
		toSerialize["meta"] = o.Meta
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RolesResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data []Role                  `json:"data,omitempty"`
		Meta *ResponseMetaAttributes `json:"meta,omitempty"`
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
	o.Data = all.Data
	if all.Meta != nil && all.Meta.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Meta = all.Meta
	return nil
}
