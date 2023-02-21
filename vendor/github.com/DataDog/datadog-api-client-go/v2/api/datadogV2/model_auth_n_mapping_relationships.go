// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// AuthNMappingRelationships All relationships associated with AuthN Mapping.
type AuthNMappingRelationships struct {
	// Relationship to role.
	Role *RelationshipToRole `json:"role,omitempty"`
	// AuthN Mapping relationship to SAML Assertion Attribute.
	SamlAssertionAttribute *RelationshipToSAMLAssertionAttribute `json:"saml_assertion_attribute,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAuthNMappingRelationships instantiates a new AuthNMappingRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAuthNMappingRelationships() *AuthNMappingRelationships {
	this := AuthNMappingRelationships{}
	return &this
}

// NewAuthNMappingRelationshipsWithDefaults instantiates a new AuthNMappingRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAuthNMappingRelationshipsWithDefaults() *AuthNMappingRelationships {
	this := AuthNMappingRelationships{}
	return &this
}

// GetRole returns the Role field value if set, zero value otherwise.
func (o *AuthNMappingRelationships) GetRole() RelationshipToRole {
	if o == nil || o.Role == nil {
		var ret RelationshipToRole
		return ret
	}
	return *o.Role
}

// GetRoleOk returns a tuple with the Role field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingRelationships) GetRoleOk() (*RelationshipToRole, bool) {
	if o == nil || o.Role == nil {
		return nil, false
	}
	return o.Role, true
}

// HasRole returns a boolean if a field has been set.
func (o *AuthNMappingRelationships) HasRole() bool {
	return o != nil && o.Role != nil
}

// SetRole gets a reference to the given RelationshipToRole and assigns it to the Role field.
func (o *AuthNMappingRelationships) SetRole(v RelationshipToRole) {
	o.Role = &v
}

// GetSamlAssertionAttribute returns the SamlAssertionAttribute field value if set, zero value otherwise.
func (o *AuthNMappingRelationships) GetSamlAssertionAttribute() RelationshipToSAMLAssertionAttribute {
	if o == nil || o.SamlAssertionAttribute == nil {
		var ret RelationshipToSAMLAssertionAttribute
		return ret
	}
	return *o.SamlAssertionAttribute
}

// GetSamlAssertionAttributeOk returns a tuple with the SamlAssertionAttribute field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AuthNMappingRelationships) GetSamlAssertionAttributeOk() (*RelationshipToSAMLAssertionAttribute, bool) {
	if o == nil || o.SamlAssertionAttribute == nil {
		return nil, false
	}
	return o.SamlAssertionAttribute, true
}

// HasSamlAssertionAttribute returns a boolean if a field has been set.
func (o *AuthNMappingRelationships) HasSamlAssertionAttribute() bool {
	return o != nil && o.SamlAssertionAttribute != nil
}

// SetSamlAssertionAttribute gets a reference to the given RelationshipToSAMLAssertionAttribute and assigns it to the SamlAssertionAttribute field.
func (o *AuthNMappingRelationships) SetSamlAssertionAttribute(v RelationshipToSAMLAssertionAttribute) {
	o.SamlAssertionAttribute = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o AuthNMappingRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Role != nil {
		toSerialize["role"] = o.Role
	}
	if o.SamlAssertionAttribute != nil {
		toSerialize["saml_assertion_attribute"] = o.SamlAssertionAttribute
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *AuthNMappingRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Role                   *RelationshipToRole                   `json:"role,omitempty"`
		SamlAssertionAttribute *RelationshipToSAMLAssertionAttribute `json:"saml_assertion_attribute,omitempty"`
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
	if all.Role != nil && all.Role.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Role = all.Role
	if all.SamlAssertionAttribute != nil && all.SamlAssertionAttribute.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SamlAssertionAttribute = all.SamlAssertionAttribute
	return nil
}
