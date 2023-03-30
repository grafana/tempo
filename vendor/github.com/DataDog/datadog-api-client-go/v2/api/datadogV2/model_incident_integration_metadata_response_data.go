// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentIntegrationMetadataResponseData Incident integration metadata from a response.
type IncidentIntegrationMetadataResponseData struct {
	// Incident integration metadata's attributes for a create request.
	Attributes *IncidentIntegrationMetadataAttributes `json:"attributes,omitempty"`
	// The incident integration metadata's ID.
	Id string `json:"id"`
	// Integration metadata resource type.
	Type IncidentIntegrationMetadataType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentIntegrationMetadataResponseData instantiates a new IncidentIntegrationMetadataResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentIntegrationMetadataResponseData(id string, typeVar IncidentIntegrationMetadataType) *IncidentIntegrationMetadataResponseData {
	this := IncidentIntegrationMetadataResponseData{}
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewIncidentIntegrationMetadataResponseDataWithDefaults instantiates a new IncidentIntegrationMetadataResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentIntegrationMetadataResponseDataWithDefaults() *IncidentIntegrationMetadataResponseData {
	this := IncidentIntegrationMetadataResponseData{}
	var typeVar IncidentIntegrationMetadataType = INCIDENTINTEGRATIONMETADATATYPE_INCIDENT_INTEGRATIONS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *IncidentIntegrationMetadataResponseData) GetAttributes() IncidentIntegrationMetadataAttributes {
	if o == nil || o.Attributes == nil {
		var ret IncidentIntegrationMetadataAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataResponseData) GetAttributesOk() (*IncidentIntegrationMetadataAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *IncidentIntegrationMetadataResponseData) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given IncidentIntegrationMetadataAttributes and assigns it to the Attributes field.
func (o *IncidentIntegrationMetadataResponseData) SetAttributes(v IncidentIntegrationMetadataAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value.
func (o *IncidentIntegrationMetadataResponseData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataResponseData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *IncidentIntegrationMetadataResponseData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *IncidentIntegrationMetadataResponseData) GetType() IncidentIntegrationMetadataType {
	if o == nil {
		var ret IncidentIntegrationMetadataType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataResponseData) GetTypeOk() (*IncidentIntegrationMetadataType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentIntegrationMetadataResponseData) SetType(v IncidentIntegrationMetadataType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentIntegrationMetadataResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentIntegrationMetadataResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Id   *string                          `json:"id"`
		Type *IncidentIntegrationMetadataType `json:"type"`
	}{}
	all := struct {
		Attributes *IncidentIntegrationMetadataAttributes `json:"attributes,omitempty"`
		Id         string                                 `json:"id"`
		Type       IncidentIntegrationMetadataType        `json:"type"`
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
	o.Type = all.Type
	return nil
}
