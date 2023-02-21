// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerRuleRelationships Relationships of a scanning rule.
type SensitiveDataScannerRuleRelationships struct {
	// A scanning group data.
	Group *SensitiveDataScannerGroupData `json:"group,omitempty"`
	// A standard pattern.
	StandardPattern *SensitiveDataScannerStandardPatternData `json:"standard_pattern,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerRuleRelationships instantiates a new SensitiveDataScannerRuleRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerRuleRelationships() *SensitiveDataScannerRuleRelationships {
	this := SensitiveDataScannerRuleRelationships{}
	return &this
}

// NewSensitiveDataScannerRuleRelationshipsWithDefaults instantiates a new SensitiveDataScannerRuleRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerRuleRelationshipsWithDefaults() *SensitiveDataScannerRuleRelationships {
	this := SensitiveDataScannerRuleRelationships{}
	return &this
}

// GetGroup returns the Group field value if set, zero value otherwise.
func (o *SensitiveDataScannerRuleRelationships) GetGroup() SensitiveDataScannerGroupData {
	if o == nil || o.Group == nil {
		var ret SensitiveDataScannerGroupData
		return ret
	}
	return *o.Group
}

// GetGroupOk returns a tuple with the Group field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleRelationships) GetGroupOk() (*SensitiveDataScannerGroupData, bool) {
	if o == nil || o.Group == nil {
		return nil, false
	}
	return o.Group, true
}

// HasGroup returns a boolean if a field has been set.
func (o *SensitiveDataScannerRuleRelationships) HasGroup() bool {
	return o != nil && o.Group != nil
}

// SetGroup gets a reference to the given SensitiveDataScannerGroupData and assigns it to the Group field.
func (o *SensitiveDataScannerRuleRelationships) SetGroup(v SensitiveDataScannerGroupData) {
	o.Group = &v
}

// GetStandardPattern returns the StandardPattern field value if set, zero value otherwise.
func (o *SensitiveDataScannerRuleRelationships) GetStandardPattern() SensitiveDataScannerStandardPatternData {
	if o == nil || o.StandardPattern == nil {
		var ret SensitiveDataScannerStandardPatternData
		return ret
	}
	return *o.StandardPattern
}

// GetStandardPatternOk returns a tuple with the StandardPattern field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerRuleRelationships) GetStandardPatternOk() (*SensitiveDataScannerStandardPatternData, bool) {
	if o == nil || o.StandardPattern == nil {
		return nil, false
	}
	return o.StandardPattern, true
}

// HasStandardPattern returns a boolean if a field has been set.
func (o *SensitiveDataScannerRuleRelationships) HasStandardPattern() bool {
	return o != nil && o.StandardPattern != nil
}

// SetStandardPattern gets a reference to the given SensitiveDataScannerStandardPatternData and assigns it to the StandardPattern field.
func (o *SensitiveDataScannerRuleRelationships) SetStandardPattern(v SensitiveDataScannerStandardPatternData) {
	o.StandardPattern = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerRuleRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Group != nil {
		toSerialize["group"] = o.Group
	}
	if o.StandardPattern != nil {
		toSerialize["standard_pattern"] = o.StandardPattern
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerRuleRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Group           *SensitiveDataScannerGroupData           `json:"group,omitempty"`
		StandardPattern *SensitiveDataScannerStandardPatternData `json:"standard_pattern,omitempty"`
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
	if all.Group != nil && all.Group.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Group = all.Group
	if all.StandardPattern != nil && all.StandardPattern.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.StandardPattern = all.StandardPattern
	return nil
}
