// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentServiceCreateData Incident Service payload for create requests.
type IncidentServiceCreateData struct {
	// The incident service's attributes for a create request.
	Attributes *IncidentServiceCreateAttributes `json:"attributes,omitempty"`
	// The incident service's relationships.
	Relationships *IncidentServiceRelationships `json:"relationships,omitempty"`
	// Incident service resource type.
	Type IncidentServiceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentServiceCreateData instantiates a new IncidentServiceCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentServiceCreateData(typeVar IncidentServiceType) *IncidentServiceCreateData {
	this := IncidentServiceCreateData{}
	this.Type = typeVar
	return &this
}

// NewIncidentServiceCreateDataWithDefaults instantiates a new IncidentServiceCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentServiceCreateDataWithDefaults() *IncidentServiceCreateData {
	this := IncidentServiceCreateData{}
	var typeVar IncidentServiceType = INCIDENTSERVICETYPE_SERVICES
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IncidentServiceCreateData) GetAttributes() IncidentServiceCreateAttributes {
	if o == nil || o.Attributes == nil {
		var ret IncidentServiceCreateAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceCreateData) GetAttributesOk() (*IncidentServiceCreateAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IncidentServiceCreateData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IncidentServiceCreateAttributes and assigns it to the Attributes field.
func (o *IncidentServiceCreateData) SetAttributes(v IncidentServiceCreateAttributes) {
	o.Attributes = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *IncidentServiceCreateData) GetRelationships() IncidentServiceRelationships {
	if o == nil || o.Relationships == nil {
		var ret IncidentServiceRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentServiceCreateData) GetRelationshipsOk() (*IncidentServiceRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *IncidentServiceCreateData) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given IncidentServiceRelationships and assigns it to the Relationships field.
func (o *IncidentServiceCreateData) SetRelationships(v IncidentServiceRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value.
func (o *IncidentServiceCreateData) GetType() IncidentServiceType {
	if o == nil {
		var ret IncidentServiceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentServiceCreateData) GetTypeOk() (*IncidentServiceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentServiceCreateData) SetType(v IncidentServiceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentServiceCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
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
func (o *IncidentServiceCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *IncidentServiceType `json:"type"`
	}{}
	all := struct {
		Attributes    *IncidentServiceCreateAttributes `json:"attributes,omitempty"`
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
