// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentServiceUpdateData Incident Service payload for update requests.
type IncidentServiceUpdateData struct {
	// The incident service's attributes for an update request.
	Attributes *IncidentServiceUpdateAttributes `json:"attributes,omitempty"`
	// The incident service's ID.
	Id *string `json:"id,omitempty"`
	// The incident service's relationships.
	Relationships *IncidentServiceRelationships `json:"relationships,omitempty"`
	// Incident service resource type.
	Type IncidentServiceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentServiceUpdateData instantiates a new IncidentServiceUpdateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentServiceUpdateData(typeVar IncidentServiceType) *IncidentServiceUpdateData {
	this := IncidentServiceUpdateData{}
	this.Type = typeVar
	return &this
}

// NewIncidentServiceUpdateDataWithDefaults instantiates a new IncidentServiceUpdateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentServiceUpdateDataWithDefaults() *IncidentServiceUpdateData {
	this := IncidentServiceUpdateData{}
	var typeVar IncidentServiceType = INCIDENTSERVICETYPE_SERVICES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IncidentServiceUpdateData) GetAttributes() IncidentServiceUpdateAttributes {
	if o == nil || o.Attributes == nil {
		var ret IncidentServiceUpdateAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceUpdateData) GetAttributesOk() (*IncidentServiceUpdateAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IncidentServiceUpdateData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IncidentServiceUpdateAttributes and assigns it to the Attributes field.
func (o *IncidentServiceUpdateData) SetAttributes(v IncidentServiceUpdateAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *IncidentServiceUpdateData) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceUpdateData) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *IncidentServiceUpdateData) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *IncidentServiceUpdateData) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *IncidentServiceUpdateData) GetRelationships() IncidentServiceRelationships {
	if o == nil || o.Relationships == nil {
		var ret IncidentServiceRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceUpdateData) GetRelationshipsOk() (*IncidentServiceRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *IncidentServiceUpdateData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given IncidentServiceRelationships and assigns it to the Relationships field.
func (o *IncidentServiceUpdateData) SetRelationships(v IncidentServiceRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *IncidentServiceUpdateData) GetType() IncidentServiceType {
	if o == nil {
		var ret IncidentServiceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentServiceUpdateData) GetTypeOk() (*IncidentServiceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentServiceUpdateData) SetType(v IncidentServiceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentServiceUpdateData) MarshalJSON() ([]byte, error) {
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
func (o *IncidentServiceUpdateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *IncidentServiceType `json:"type"`
	}{}
	all := struct {
		Attributes    *IncidentServiceUpdateAttributes `json:"attributes,omitempty"`
		Id            *string                          `json:"id,omitempty"`
		Relationships *IncidentServiceRelationships    `json:"relationships,omitempty"`
		Type          IncidentServiceType              `json:"type"`
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
