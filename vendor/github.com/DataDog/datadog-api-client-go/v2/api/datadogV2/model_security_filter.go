// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityFilter The security filter's properties.
type SecurityFilter struct {
	// The object describing a security filter.
	Attributes *SecurityFilterAttributes `json:"attributes,omitempty"`
	// The ID of the security filter.
	Id *string `json:"id,omitempty"`
	// The type of the resource. The value should always be `security_filters`.
	Type *SecurityFilterType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityFilter instantiates a new SecurityFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityFilter() *SecurityFilter {
	this := SecurityFilter{}
	var typeVar SecurityFilterType = SECURITYFILTERTYPE_SECURITY_FILTERS
	this.Type = &typeVar
	return &this
}

// NewSecurityFilterWithDefaults instantiates a new SecurityFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityFilterWithDefaults() *SecurityFilter {
	this := SecurityFilter{}
	var typeVar SecurityFilterType = SECURITYFILTERTYPE_SECURITY_FILTERS
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SecurityFilter) GetAttributes() SecurityFilterAttributes {
	if o == nil || o.Attributes == nil {
		var ret SecurityFilterAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilter) GetAttributesOk() (*SecurityFilterAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SecurityFilter) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given SecurityFilterAttributes and assigns it to the Attributes field.
func (o *SecurityFilter) SetAttributes(v SecurityFilterAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SecurityFilter) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilter) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SecurityFilter) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SecurityFilter) SetId(v string) {
	o.Id = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SecurityFilter) GetType() SecurityFilterType {
	if o == nil || o.Type == nil {
		var ret SecurityFilterType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityFilter) GetTypeOk() (*SecurityFilterType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SecurityFilter) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SecurityFilterType and assigns it to the Type field.
func (o *SecurityFilter) SetType(v SecurityFilterType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityFilter) MarshalJSON() ([]byte, error) {
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
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes *SecurityFilterAttributes `json:"attributes,omitempty"`
		Id         *string                   `json:"id,omitempty"`
		Type       *SecurityFilterType       `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
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
