// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// OpsgenieServiceUpdateData Opsgenie service for an update request.
type OpsgenieServiceUpdateData struct {
	// The Opsgenie service attributes for an update request.
	Attributes OpsgenieServiceUpdateAttributes `json:"attributes"`
	// The ID of the Opsgenie service.
	Id string `json:"id"`
	// Opsgenie service resource type.
	Type OpsgenieServiceType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOpsgenieServiceUpdateData instantiates a new OpsgenieServiceUpdateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOpsgenieServiceUpdateData(attributes OpsgenieServiceUpdateAttributes, id string, typeVar OpsgenieServiceType) *OpsgenieServiceUpdateData {
	this := OpsgenieServiceUpdateData{}
	this.Attributes = attributes
	this.Id = id
	this.Type = typeVar
	return &this
}

// NewOpsgenieServiceUpdateDataWithDefaults instantiates a new OpsgenieServiceUpdateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOpsgenieServiceUpdateDataWithDefaults() *OpsgenieServiceUpdateData {
	this := OpsgenieServiceUpdateData{}
	var typeVar OpsgenieServiceType = OPSGENIESERVICETYPE_OPSGENIE_SERVICE
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *OpsgenieServiceUpdateData) GetAttributes() OpsgenieServiceUpdateAttributes {
	if o == nil {
		var ret OpsgenieServiceUpdateAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateData) GetAttributesOk() (*OpsgenieServiceUpdateAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *OpsgenieServiceUpdateData) SetAttributes(v OpsgenieServiceUpdateAttributes) {
	o.Attributes = v
}

// GetId returns the Id field value.
func (o *OpsgenieServiceUpdateData) GetId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateData) GetIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *OpsgenieServiceUpdateData) SetId(v string) {
	o.Id = v
}

// GetType returns the Type field value.
func (o *OpsgenieServiceUpdateData) GetType() OpsgenieServiceType {
	if o == nil {
		var ret OpsgenieServiceType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateData) GetTypeOk() (*OpsgenieServiceType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *OpsgenieServiceUpdateData) SetType(v OpsgenieServiceType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o OpsgenieServiceUpdateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["id"] = o.Id
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OpsgenieServiceUpdateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes *OpsgenieServiceUpdateAttributes `json:"attributes"`
		Id         *string                          `json:"id"`
		Type       *OpsgenieServiceType             `json:"type"`
	}{}
	all := struct {
		Attributes OpsgenieServiceUpdateAttributes `json:"attributes"`
		Id         string                          `json:"id"`
		Type       OpsgenieServiceType             `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
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
	if all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
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
