// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerGroupUpdate Data related to the update of a group.
type SensitiveDataScannerGroupUpdate struct {
	// Attributes of the Sensitive Data Scanner group.
	Attributes *SensitiveDataScannerGroupAttributes `json:"attributes,omitempty"`
	// ID of the group.
	Id *string `json:"id,omitempty"`
	// Relationships of the group.
	Relationships *SensitiveDataScannerGroupRelationships `json:"relationships,omitempty"`
	// Sensitive Data Scanner group type.
	Type *SensitiveDataScannerGroupType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerGroupUpdate instantiates a new SensitiveDataScannerGroupUpdate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerGroupUpdate() *SensitiveDataScannerGroupUpdate {
	this := SensitiveDataScannerGroupUpdate{}
	var typeVar SensitiveDataScannerGroupType = SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP
	this.Type = &typeVar
	return &this
}

// NewSensitiveDataScannerGroupUpdateWithDefaults instantiates a new SensitiveDataScannerGroupUpdate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerGroupUpdateWithDefaults() *SensitiveDataScannerGroupUpdate {
	this := SensitiveDataScannerGroupUpdate{}
	var typeVar SensitiveDataScannerGroupType = SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupUpdate) GetAttributes() SensitiveDataScannerGroupAttributes {
	if o == nil || o.Attributes == nil {
		var ret SensitiveDataScannerGroupAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupUpdate) GetAttributesOk() (*SensitiveDataScannerGroupAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupUpdate) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given SensitiveDataScannerGroupAttributes and assigns it to the Attributes field.
func (o *SensitiveDataScannerGroupUpdate) SetAttributes(v SensitiveDataScannerGroupAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupUpdate) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupUpdate) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupUpdate) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SensitiveDataScannerGroupUpdate) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupUpdate) GetRelationships() SensitiveDataScannerGroupRelationships {
	if o == nil || o.Relationships == nil {
		var ret SensitiveDataScannerGroupRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupUpdate) GetRelationshipsOk() (*SensitiveDataScannerGroupRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupUpdate) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given SensitiveDataScannerGroupRelationships and assigns it to the Relationships field.
func (o *SensitiveDataScannerGroupUpdate) SetRelationships(v SensitiveDataScannerGroupRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupUpdate) GetType() SensitiveDataScannerGroupType {
	if o == nil || o.Type == nil {
		var ret SensitiveDataScannerGroupType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupUpdate) GetTypeOk() (*SensitiveDataScannerGroupType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupUpdate) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SensitiveDataScannerGroupType and assigns it to the Type field.
func (o *SensitiveDataScannerGroupUpdate) SetType(v SensitiveDataScannerGroupType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerGroupUpdate) MarshalJSON() ([]byte, error) {
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
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerGroupUpdate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Attributes    *SensitiveDataScannerGroupAttributes    `json:"attributes,omitempty"`
		Id            *string                                 `json:"id,omitempty"`
		Relationships *SensitiveDataScannerGroupRelationships `json:"relationships,omitempty"`
		Type          *SensitiveDataScannerGroupType          `json:"type,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
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
