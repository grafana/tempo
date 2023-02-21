// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuthNMappingUpdateAttributes Key/Value pair of attributes used for update request.
type AuthNMappingUpdateAttributes struct {
	// Key portion of a key/value pair of the attribute sent from the Identity Provider.
	AttributeKey *string `json:"attribute_key,omitempty"`
	// Value portion of a key/value pair of the attribute sent from the Identity Provider.
	AttributeValue *string `json:"attribute_value,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuthNMappingUpdateAttributes instantiates a new AuthNMappingUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuthNMappingUpdateAttributes() *AuthNMappingUpdateAttributes {
	this := AuthNMappingUpdateAttributes{}
	return &this
}

// NewAuthNMappingUpdateAttributesWithDefaults instantiates a new AuthNMappingUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuthNMappingUpdateAttributesWithDefaults() *AuthNMappingUpdateAttributes {
	this := AuthNMappingUpdateAttributes{}
	return &this
}

// GetAttributeKey returns the AttributeKey field value if set, zero value otherwise.
func (o *AuthNMappingUpdateAttributes) GetAttributeKey() string {
	if o == nil || o.AttributeKey == nil {
		var ret string
		return ret
	}
	return *o.AttributeKey
}

// GetAttributeKeyOk returns a tuple with the AttributeKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingUpdateAttributes) GetAttributeKeyOk() (*string, bool) {
	if o == nil || o.AttributeKey == nil {
		return nil, false
	}
	return o.AttributeKey, true
}

// HasAttributeKey returns a boolean if a field has been set.
func (o *AuthNMappingUpdateAttributes) HasAttributeKey() bool {
	return o != nil && o.AttributeKey != nil
}

// SetAttributeKey gets a reference to the given string and assigns it to the AttributeKey field.
func (o *AuthNMappingUpdateAttributes) SetAttributeKey(v string) {
	o.AttributeKey = &v
}

// GetAttributeValue returns the AttributeValue field value if set, zero value otherwise.
func (o *AuthNMappingUpdateAttributes) GetAttributeValue() string {
	if o == nil || o.AttributeValue == nil {
		var ret string
		return ret
	}
	return *o.AttributeValue
}

// GetAttributeValueOk returns a tuple with the AttributeValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingUpdateAttributes) GetAttributeValueOk() (*string, bool) {
	if o == nil || o.AttributeValue == nil {
		return nil, false
	}
	return o.AttributeValue, true
}

// HasAttributeValue returns a boolean if a field has been set.
func (o *AuthNMappingUpdateAttributes) HasAttributeValue() bool {
	return o != nil && o.AttributeValue != nil
}

// SetAttributeValue gets a reference to the given string and assigns it to the AttributeValue field.
func (o *AuthNMappingUpdateAttributes) SetAttributeValue(v string) {
	o.AttributeValue = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuthNMappingUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AttributeKey != nil {
		toSerialize["attribute_key"] = o.AttributeKey
	}
	if o.AttributeValue != nil {
		toSerialize["attribute_value"] = o.AttributeValue
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuthNMappingUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AttributeKey   *string `json:"attribute_key,omitempty"`
		AttributeValue *string `json:"attribute_value,omitempty"`
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
	o.AttributeKey = all.AttributeKey
	o.AttributeValue = all.AttributeValue
	return nil
}
