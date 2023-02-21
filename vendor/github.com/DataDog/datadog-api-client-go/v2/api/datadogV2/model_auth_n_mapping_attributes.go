// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// AuthNMappingAttributes Attributes of AuthN Mapping.
type AuthNMappingAttributes struct {
	// Key portion of a key/value pair of the attribute sent from the Identity Provider.
	AttributeKey *string `json:"attribute_key,omitempty"`
	// Value portion of a key/value pair of the attribute sent from the Identity Provider.
	AttributeValue *string `json:"attribute_value,omitempty"`
	// Creation time of the AuthN Mapping.
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Time of last AuthN Mapping modification.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
	// The ID of the SAML assertion attribute.
	SamlAssertionAttributeId *string `json:"saml_assertion_attribute_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuthNMappingAttributes instantiates a new AuthNMappingAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuthNMappingAttributes() *AuthNMappingAttributes {
	this := AuthNMappingAttributes{}
	return &this
}

// NewAuthNMappingAttributesWithDefaults instantiates a new AuthNMappingAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuthNMappingAttributesWithDefaults() *AuthNMappingAttributes {
	this := AuthNMappingAttributes{}
	return &this
}

// GetAttributeKey returns the AttributeKey field value if set, zero value otherwise.
func (o *AuthNMappingAttributes) GetAttributeKey() string {
	if o == nil || o.AttributeKey == nil {
		var ret string
		return ret
	}
	return *o.AttributeKey
}

// GetAttributeKeyOk returns a tuple with the AttributeKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingAttributes) GetAttributeKeyOk() (*string, bool) {
	if o == nil || o.AttributeKey == nil {
		return nil, false
	}
	return o.AttributeKey, true
}

// HasAttributeKey returns a boolean if a field has been set.
func (o *AuthNMappingAttributes) HasAttributeKey() bool {
	return o != nil && o.AttributeKey != nil
}

// SetAttributeKey gets a reference to the given string and assigns it to the AttributeKey field.
func (o *AuthNMappingAttributes) SetAttributeKey(v string) {
	o.AttributeKey = &v
}

// GetAttributeValue returns the AttributeValue field value if set, zero value otherwise.
func (o *AuthNMappingAttributes) GetAttributeValue() string {
	if o == nil || o.AttributeValue == nil {
		var ret string
		return ret
	}
	return *o.AttributeValue
}

// GetAttributeValueOk returns a tuple with the AttributeValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingAttributes) GetAttributeValueOk() (*string, bool) {
	if o == nil || o.AttributeValue == nil {
		return nil, false
	}
	return o.AttributeValue, true
}

// HasAttributeValue returns a boolean if a field has been set.
func (o *AuthNMappingAttributes) HasAttributeValue() bool {
	return o != nil && o.AttributeValue != nil
}

// SetAttributeValue gets a reference to the given string and assigns it to the AttributeValue field.
func (o *AuthNMappingAttributes) SetAttributeValue(v string) {
	o.AttributeValue = &v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *AuthNMappingAttributes) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingAttributes) GetCreatedAtOk() (*time.Time, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *AuthNMappingAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given time.Time and assigns it to the CreatedAt field.
func (o *AuthNMappingAttributes) SetCreatedAt(v time.Time) {
	o.CreatedAt = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *AuthNMappingAttributes) GetModifiedAt() time.Time {
	if o == nil || o.ModifiedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingAttributes) GetModifiedAtOk() (*time.Time, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *AuthNMappingAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given time.Time and assigns it to the ModifiedAt field.
func (o *AuthNMappingAttributes) SetModifiedAt(v time.Time) {
	o.ModifiedAt = &v
}

// GetSamlAssertionAttributeId returns the SamlAssertionAttributeId field value if set, zero value otherwise.
func (o *AuthNMappingAttributes) GetSamlAssertionAttributeId() string {
	if o == nil || o.SamlAssertionAttributeId == nil {
		var ret string
		return ret
	}
	return *o.SamlAssertionAttributeId
}

// GetSamlAssertionAttributeIdOk returns a tuple with the SamlAssertionAttributeId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingAttributes) GetSamlAssertionAttributeIdOk() (*string, bool) {
	if o == nil || o.SamlAssertionAttributeId == nil {
		return nil, false
	}
	return o.SamlAssertionAttributeId, true
}

// HasSamlAssertionAttributeId returns a boolean if a field has been set.
func (o *AuthNMappingAttributes) HasSamlAssertionAttributeId() bool {
	return o != nil && o.SamlAssertionAttributeId != nil
}

// SetSamlAssertionAttributeId gets a reference to the given string and assigns it to the SamlAssertionAttributeId field.
func (o *AuthNMappingAttributes) SetSamlAssertionAttributeId(v string) {
	o.SamlAssertionAttributeId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuthNMappingAttributes) MarshalJSON() ([]byte, error) {
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
	if o.CreatedAt != nil {
		if o.CreatedAt.Nanosecond() == 0 {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created_at"] = o.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.ModifiedAt != nil {
		if o.ModifiedAt.Nanosecond() == 0 {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified_at"] = o.ModifiedAt.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.SamlAssertionAttributeId != nil {
		toSerialize["saml_assertion_attribute_id"] = o.SamlAssertionAttributeId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuthNMappingAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AttributeKey             *string    `json:"attribute_key,omitempty"`
		AttributeValue           *string    `json:"attribute_value,omitempty"`
		CreatedAt                *time.Time `json:"created_at,omitempty"`
		ModifiedAt               *time.Time `json:"modified_at,omitempty"`
		SamlAssertionAttributeId *string    `json:"saml_assertion_attribute_id,omitempty"`
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
	o.CreatedAt = all.CreatedAt
	o.ModifiedAt = all.ModifiedAt
	o.SamlAssertionAttributeId = all.SamlAssertionAttributeId
	return nil
}
