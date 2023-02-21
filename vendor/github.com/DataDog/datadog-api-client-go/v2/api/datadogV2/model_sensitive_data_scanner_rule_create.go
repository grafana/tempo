// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SensitiveDataScannerRuleCreate Data related to the creation of a rule.
type SensitiveDataScannerRuleCreate struct {
	// Attributes of the Sensitive Data Scanner rule.
	Attributes SensitiveDataScannerRuleAttributes `json:"attributes"`
	// Relationships of a scanning rule.
	Relationships SensitiveDataScannerRuleRelationships `json:"relationships"`
	// Sensitive Data Scanner rule type.
	Type SensitiveDataScannerRuleType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerRuleCreate instantiates a new SensitiveDataScannerRuleCreate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerRuleCreate(attributes SensitiveDataScannerRuleAttributes, relationships SensitiveDataScannerRuleRelationships, typeVar SensitiveDataScannerRuleType) *SensitiveDataScannerRuleCreate {
	this := SensitiveDataScannerRuleCreate{}
	this.Attributes = attributes
	this.Relationships = relationships
	this.Type = typeVar
	return &this
}

// NewSensitiveDataScannerRuleCreateWithDefaults instantiates a new SensitiveDataScannerRuleCreate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerRuleCreateWithDefaults() *SensitiveDataScannerRuleCreate {
	this := SensitiveDataScannerRuleCreate{}
	var typeVar SensitiveDataScannerRuleType = SENSITIVEDATASCANNERRULETYPE_SENSITIVE_DATA_SCANNER_RULE
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value.
func (o *SensitiveDataScannerRuleCreate) GetAttributes() SensitiveDataScannerRuleAttributes {
	if o == nil {
		var ret SensitiveDataScannerRuleAttributes
		return ret
	}
	return o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleCreate) GetAttributesOk() (*SensitiveDataScannerRuleAttributes, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Attributes, true
}

// SetAttributes sets field value.
func (o *SensitiveDataScannerRuleCreate) SetAttributes(v SensitiveDataScannerRuleAttributes) {
	o.Attributes = v
}

// GetRelationships returns the Relationships field value.
func (o *SensitiveDataScannerRuleCreate) GetRelationships() SensitiveDataScannerRuleRelationships {
	if o == nil {
		var ret SensitiveDataScannerRuleRelationships
		return ret
	}
	return o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleCreate) GetRelationshipsOk() (*SensitiveDataScannerRuleRelationships, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Relationships, true
}

// SetRelationships sets field value.
func (o *SensitiveDataScannerRuleCreate) SetRelationships(v SensitiveDataScannerRuleRelationships) {
	o.Relationships = v
}

// GetType returns the Type field value.
func (o *SensitiveDataScannerRuleCreate) GetType() SensitiveDataScannerRuleType {
	if o == nil {
		var ret SensitiveDataScannerRuleType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleCreate) GetTypeOk() (*SensitiveDataScannerRuleType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SensitiveDataScannerRuleCreate) SetType(v SensitiveDataScannerRuleType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerRuleCreate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["attributes"] = o.Attributes
	toSerialize["relationships"] = o.Relationships
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerRuleCreate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Attributes    *SensitiveDataScannerRuleAttributes    `json:"attributes"`
		Relationships *SensitiveDataScannerRuleRelationships `json:"relationships"`
		Type          *SensitiveDataScannerRuleType          `json:"type"`
	}{}
	all := struct {
		Attributes    SensitiveDataScannerRuleAttributes    `json:"attributes"`
		Relationships SensitiveDataScannerRuleRelationships `json:"relationships"`
		Type          SensitiveDataScannerRuleType          `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Attributes == nil {
		return fmt.Errorf("required field attributes missing")
	}
	if required.Relationships == nil {
		return fmt.Errorf("required field relationships missing")
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
	if all.Relationships.UnparsedObject != nil && o.UnparsedObject == nil {
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
