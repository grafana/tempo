// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// APIKeyRelationships Resources related to the API key.
type APIKeyRelationships struct {
	// Relationship to user.
	CreatedBy *RelationshipToUser `json:"created_by,omitempty"`
	// Relationship to user.
	ModifiedBy *RelationshipToUser `json:"modified_by,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAPIKeyRelationships instantiates a new APIKeyRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAPIKeyRelationships() *APIKeyRelationships {
	this := APIKeyRelationships{}
	return &this
}

// NewAPIKeyRelationshipsWithDefaults instantiates a new APIKeyRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAPIKeyRelationshipsWithDefaults() *APIKeyRelationships {
	this := APIKeyRelationships{}
	return &this
}

// GetCreatedBy returns the CreatedBy field value if set, zero value otherwise.
func (o *APIKeyRelationships) GetCreatedBy() RelationshipToUser {
	if o == nil || o.CreatedBy == nil {
		var ret RelationshipToUser
		return ret
	}
	return *o.CreatedBy
}

// GetCreatedByOk returns a tuple with the CreatedBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *APIKeyRelationships) GetCreatedByOk() (*RelationshipToUser, bool) {
	if o == nil || o.CreatedBy == nil {
		return nil, false
	}
	return o.CreatedBy, true
}

// HasCreatedBy returns a boolean if a field has been set.
func (o *APIKeyRelationships) HasCreatedBy() bool {
	return o != nil && o.CreatedBy != nil
}

// SetCreatedBy gets a reference to the given RelationshipToUser and assigns it to the CreatedBy field.
func (o *APIKeyRelationships) SetCreatedBy(v RelationshipToUser) {
	o.CreatedBy = &v
}

// GetModifiedBy returns the ModifiedBy field value if set, zero value otherwise.
func (o *APIKeyRelationships) GetModifiedBy() RelationshipToUser {
	if o == nil || o.ModifiedBy == nil {
		var ret RelationshipToUser
		return ret
	}
	return *o.ModifiedBy
}

// GetModifiedByOk returns a tuple with the ModifiedBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *APIKeyRelationships) GetModifiedByOk() (*RelationshipToUser, bool) {
	if o == nil || o.ModifiedBy == nil {
		return nil, false
	}
	return o.ModifiedBy, true
}

// HasModifiedBy returns a boolean if a field has been set.
func (o *APIKeyRelationships) HasModifiedBy() bool {
	return o != nil && o.ModifiedBy != nil
}

// SetModifiedBy gets a reference to the given RelationshipToUser and assigns it to the ModifiedBy field.
func (o *APIKeyRelationships) SetModifiedBy(v RelationshipToUser) {
	o.ModifiedBy = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o APIKeyRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CreatedBy != nil {
		toSerialize["created_by"] = o.CreatedBy
	}
	if o.ModifiedBy != nil {
		toSerialize["modified_by"] = o.ModifiedBy
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *APIKeyRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CreatedBy  *RelationshipToUser `json:"created_by,omitempty"`
		ModifiedBy *RelationshipToUser `json:"modified_by,omitempty"`
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
	if all.CreatedBy != nil && all.CreatedBy.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CreatedBy = all.CreatedBy
	if all.ModifiedBy != nil && all.ModifiedBy.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ModifiedBy = all.ModifiedBy
	return nil
}
