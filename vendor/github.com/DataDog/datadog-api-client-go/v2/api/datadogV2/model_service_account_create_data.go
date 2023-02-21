// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceAccountCreateData Object to create a service account User.
type ServiceAccountCreateData struct {
	// Attributes of the created user.
	Attributes ServiceAccountCreateAttributes `json:"attributes"`
	// Relationships of the user object.
	Relationships *UserRelationships `json:"relationships,omitempty"`
	// Users resource type.
	Type UsersType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceAccountCreateData instantiates a new ServiceAccountCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceAccountCreateData(attributes ServiceAccountCreateAttributes, typeVar UsersType) *ServiceAccountCreateData {
	this := ServiceAccountCreateData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewServiceAccountCreateDataWithDefaults instantiates a new ServiceAccountCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceAccountCreateDataWithDefaults() *ServiceAccountCreateData {
	this := ServiceAccountCreateData{}
	var typeVar UsersType = USERSTYPE_USERS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *ServiceAccountCreateData) GetAttributes() ServiceAccountCreateAttributes {
	if o == nil {
		var ret ServiceAccountCreateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateData) GetAttributesOk() (*ServiceAccountCreateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *ServiceAccountCreateData) SetAttributes(v ServiceAccountCreateAttributes) {
	o.Attributes = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *ServiceAccountCreateData) GetRelationships() UserRelationships {
	if o == nil || o.Relationships == nil {
		var ret UserRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateData) GetRelationshipsOk() (*UserRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *ServiceAccountCreateData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given UserRelationships and assigns it to the Relationships field.
func (o *ServiceAccountCreateData) SetRelationships(v UserRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *ServiceAccountCreateData) GetType() UsersType {
	if o == nil {
		var ret UsersType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ServiceAccountCreateData) GetTypeOk() (*UsersType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ServiceAccountCreateData) SetType(v UsersType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceAccountCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	if o.Relationships != nil {
		toSerialize["relationships"] = o.Relationships
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceAccountCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *ServiceAccountCreateAttributes `json:"attributes"`
		Type       *UsersType                      `json:"type"`
	}{}
	all := struct {
		Attributes    ServiceAccountCreateAttributes `json:"attributes"`
		Relationships *UserRelationships             `json:"relationships,omitempty"`
		Type          UsersType                      `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
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
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	if all.Relationships != nil && all.Relationships.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Relationships = all.Relationships
	o.Type = all.Type
	return nil
}
