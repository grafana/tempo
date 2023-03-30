// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// IncidentIntegrationMetadataPatchData Incident integration metadata data for a patch request.
type IncidentIntegrationMetadataPatchData struct {
	// Incident integration metadata's attributes for a create request.
	Attributes IncidentIntegrationMetadataAttributes `json:"attributes"`
	// Integration metadata resource type.
	Type IncidentIntegrationMetadataType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentIntegrationMetadataPatchData instantiates a new IncidentIntegrationMetadataPatchData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentIntegrationMetadataPatchData(attributes IncidentIntegrationMetadataAttributes, typeVar IncidentIntegrationMetadataType) *IncidentIntegrationMetadataPatchData {
	this := IncidentIntegrationMetadataPatchData{}
	this.Attributes = attributes
	this.Type = typeVar
	return &this
}

// NewIncidentIntegrationMetadataPatchDataWithDefaults instantiates a new IncidentIntegrationMetadataPatchData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentIntegrationMetadataPatchDataWithDefaults() *IncidentIntegrationMetadataPatchData {
	this := IncidentIntegrationMetadataPatchData{}
	var typeVar IncidentIntegrationMetadataType = INCIDENTINTEGRATIONMETADATATYPE_INCIDENT_INTEGRATIONS
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *IncidentIntegrationMetadataPatchData) GetAttributes() IncidentIntegrationMetadataAttributes {
	if o == nil {
		var ret IncidentIntegrationMetadataAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataPatchData) GetAttributesOk() (*IncidentIntegrationMetadataAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *IncidentIntegrationMetadataPatchData) SetAttributes(v IncidentIntegrationMetadataAttributes) {
	o.Attributes = v
}

// GetType returns the Type field value.
func (o *IncidentIntegrationMetadataPatchData) GetType() IncidentIntegrationMetadataType {
	if o == nil {
		var ret IncidentIntegrationMetadataType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *IncidentIntegrationMetadataPatchData) GetTypeOk() (*IncidentIntegrationMetadataType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *IncidentIntegrationMetadataPatchData) SetType(v IncidentIntegrationMetadataType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentIntegrationMetadataPatchData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentIntegrationMetadataPatchData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *IncidentIntegrationMetadataAttributes `json:"attributes"`
		Type       *IncidentIntegrationMetadataType       `json:"type"`
	}{}
	all := struct {
		Attributes IncidentIntegrationMetadataAttributes `json:"attributes"`
		Type       IncidentIntegrationMetadataType       `json:"type"`
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
	o.Type = all.Type
	return nil
}
