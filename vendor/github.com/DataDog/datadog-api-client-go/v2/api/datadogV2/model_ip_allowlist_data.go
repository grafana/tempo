// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IPAllowlistData IP allowlist data.
type IPAllowlistData struct {
	// Attributes of the IP allowlist.
	Attributes *IPAllowlistAttributes `json:"attributes,omitempty"`
	// The unique identifier of the org.
	Id *string `json:"id,omitempty"`
	// IP allowlist type.
	Type IPAllowlistType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIPAllowlistData instantiates a new IPAllowlistData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPAllowlistData(typeVar IPAllowlistType) *IPAllowlistData {
	this := IPAllowlistData{}
	this.Type = typeVar
	return &this
}

// NewIPAllowlistDataWithDefaults instantiates a new IPAllowlistData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPAllowlistDataWithDefaults() *IPAllowlistData {
	this := IPAllowlistData{}
	var typeVar IPAllowlistType = IPALLOWLISTTYPE_IP_ALLOWLIST
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IPAllowlistData) GetAttributes() IPAllowlistAttributes {
	if o == nil || o.Attributes == nil {
		var ret IPAllowlistAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistData) GetAttributesOk() (*IPAllowlistAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IPAllowlistData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IPAllowlistAttributes and assigns it to the Attributes field.
func (o *IPAllowlistData) SetAttributes(v IPAllowlistAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *IPAllowlistData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *IPAllowlistData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *IPAllowlistData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value.
func (o *IPAllowlistData) GetType() IPAllowlistType {
	if o == nil {
		var ret IPAllowlistType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IPAllowlistData) GetTypeOk() (*IPAllowlistType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IPAllowlistData) SetType(v IPAllowlistType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPAllowlistData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IPAllowlistData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *IPAllowlistType `json:"type"`
	}{}
	all := struct {
		Attributes *IPAllowlistAttributes `json:"attributes,omitempty"`
		Id         *string                `json:"id,omitempty"`
		Type       IPAllowlistType        `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Id = all.Id
	o.Type = all.Type
	return nil
}
