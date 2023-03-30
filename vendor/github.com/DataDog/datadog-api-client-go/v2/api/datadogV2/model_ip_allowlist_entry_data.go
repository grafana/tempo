// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IPAllowlistEntryData Data of the IP allowlist entry object.
type IPAllowlistEntryData struct {
	// Attributes of the IP allowlist entry.
	Attributes *IPAllowlistEntryAttributes `json:"attributes,omitempty"`
	// The unique identifier of the IP allowlist entry.
	Id *string `json:"id,omitempty"`
	// IP allowlist Entry type.
	Type IPAllowlistEntryType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIPAllowlistEntryData instantiates a new IPAllowlistEntryData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPAllowlistEntryData(typeVar IPAllowlistEntryType) *IPAllowlistEntryData {
	this := IPAllowlistEntryData{}
	this.Type = typeVar
	return &this
}

// NewIPAllowlistEntryDataWithDefaults instantiates a new IPAllowlistEntryData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPAllowlistEntryDataWithDefaults() *IPAllowlistEntryData {
	this := IPAllowlistEntryData{}
	var typeVar IPAllowlistEntryType = IPALLOWLISTENTRYTYPE_IP_ALLOWLIST_ENTRY
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IPAllowlistEntryData) GetAttributes() IPAllowlistEntryAttributes {
	if o == nil || o.Attributes == nil {
		var ret IPAllowlistEntryAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryData) GetAttributesOk() (*IPAllowlistEntryAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IPAllowlistEntryData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IPAllowlistEntryAttributes and assigns it to the Attributes field.
func (o *IPAllowlistEntryData) SetAttributes(v IPAllowlistEntryAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *IPAllowlistEntryData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *IPAllowlistEntryData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *IPAllowlistEntryData) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value.
func (o *IPAllowlistEntryData) GetType() IPAllowlistEntryType {
	if o == nil {
		var ret IPAllowlistEntryType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IPAllowlistEntryData) GetTypeOk() (*IPAllowlistEntryType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IPAllowlistEntryData) SetType(v IPAllowlistEntryType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPAllowlistEntryData) MarshalJSON() ([]byte, error) {
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
func (o *IPAllowlistEntryData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *IPAllowlistEntryType `json:"type"`
	}{}
	all := struct {
		Attributes *IPAllowlistEntryAttributes `json:"attributes,omitempty"`
		Id         *string                     `json:"id,omitempty"`
		Type       IPAllowlistEntryType        `json:"type"`
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
