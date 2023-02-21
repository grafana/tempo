// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerGroupResponse Response data related to the creation of a group.
type SensitiveDataScannerGroupResponse struct {
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

// NewSensitiveDataScannerGroupResponse instantiates a new SensitiveDataScannerGroupResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerGroupResponse() *SensitiveDataScannerGroupResponse {
	this := SensitiveDataScannerGroupResponse{}
	var typeVar SensitiveDataScannerGroupType = SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP
	this.Type = &typeVar
	return &this
}

// NewSensitiveDataScannerGroupResponseWithDefaults instantiates a new SensitiveDataScannerGroupResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerGroupResponseWithDefaults() *SensitiveDataScannerGroupResponse {
	this := SensitiveDataScannerGroupResponse{}
	var typeVar SensitiveDataScannerGroupType = SENSITIVEDATASCANNERGROUPTYPE_SENSITIVE_DATA_SCANNER_GROUP
	this.Type = &typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupResponse) GetAttributes() SensitiveDataScannerGroupAttributes {
	if o == nil || o.Attributes == nil {
		var ret SensitiveDataScannerGroupAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupResponse) GetAttributesOk() (*SensitiveDataScannerGroupAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupResponse) HasAttributes() bool {
	return o != nil && o.Attributes != nil
}

// SetAttributes gets a reference to the given SensitiveDataScannerGroupAttributes and assigns it to the Attributes field.
func (o *SensitiveDataScannerGroupResponse) SetAttributes(v SensitiveDataScannerGroupAttributes) {
	o.Attributes = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupResponse) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupResponse) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupResponse) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SensitiveDataScannerGroupResponse) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupResponse) GetRelationships() SensitiveDataScannerGroupRelationships {
	if o == nil || o.Relationships == nil {
		var ret SensitiveDataScannerGroupRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupResponse) GetRelationshipsOk() (*SensitiveDataScannerGroupRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupResponse) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given SensitiveDataScannerGroupRelationships and assigns it to the Relationships field.
func (o *SensitiveDataScannerGroupResponse) SetRelationships(v SensitiveDataScannerGroupRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupResponse) GetType() SensitiveDataScannerGroupType {
	if o == nil || o.Type == nil {
		var ret SensitiveDataScannerGroupType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupResponse) GetTypeOk() (*SensitiveDataScannerGroupType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupResponse) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SensitiveDataScannerGroupType and assigns it to the Type field.
func (o *SensitiveDataScannerGroupResponse) SetType(v SensitiveDataScannerGroupType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerGroupResponse) MarshalJSON() ([]byte, error) {
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
func (o *SensitiveDataScannerGroupResponse) UnmarshalJSON(bytes []byte) (err error) {
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
