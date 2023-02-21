// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentServiceResponseData Incident Service data from responses.
type IncidentServiceResponseData struct {
	// The incident service's attributes from a response.
	Attributes *IncidentServiceResponseAttributes `json:"attributes,omitempty"`
	// The incident service's ID.
	Id string `json:"id"`
	// The incident service's relationships.
	Relationships *IncidentServiceRelationships `json:"relationships,omitempty"`
	// Incident service resource type.
	Type IncidentServiceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentServiceResponseData instantiates a new IncidentServiceResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentServiceResponseData(id string, typeVar IncidentServiceType) *IncidentServiceResponseData {
	this := IncidentServiceResponseData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewIncidentServiceResponseDataWithDefaults instantiates a new IncidentServiceResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentServiceResponseDataWithDefaults() *IncidentServiceResponseData {
	this := IncidentServiceResponseData{}
	var typeVar IncidentServiceType = INCIDENTSERVICETYPE_SERVICES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IncidentServiceResponseData) GetAttributes() IncidentServiceResponseAttributes {
	if o == nil || o.Attributes == nil {
		var ret IncidentServiceResponseAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponseData) GetAttributesOk() (*IncidentServiceResponseAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IncidentServiceResponseData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IncidentServiceResponseAttributes and assigns it to the Attributes field.
func (o *IncidentServiceResponseData) SetAttributes(v IncidentServiceResponseAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value.
func (o *IncidentServiceResponseData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponseData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *IncidentServiceResponseData) SetId(v string) {
	o.Id = v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *IncidentServiceResponseData) GetRelationships() IncidentServiceRelationships {
	if o == nil || o.Relationships == nil {
		var ret IncidentServiceRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponseData) GetRelationshipsOk() (*IncidentServiceRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *IncidentServiceResponseData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given IncidentServiceRelationships and assigns it to the Relationships field.
func (o *IncidentServiceResponseData) SetRelationships(v IncidentServiceRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *IncidentServiceResponseData) GetType() IncidentServiceType {
	if o == nil {
		var ret IncidentServiceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentServiceResponseData) GetTypeOk() (*IncidentServiceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentServiceResponseData) SetType(v IncidentServiceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentServiceResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["id"] = o.Id
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
func (o *IncidentServiceResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string              `json:"id"`
		Type *IncidentServiceType `json:"type"`
	}{}
	all := struct {
		Attributes    *IncidentServiceResponseAttributes `json:"attributes,omitempty"`
		Id            string                             `json:"id"`
		Relationships *IncidentServiceRelationships      `json:"relationships,omitempty"`
		Type          IncidentServiceType                `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Id == nil {
		return fmt.Errorf("required field id missing")
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
