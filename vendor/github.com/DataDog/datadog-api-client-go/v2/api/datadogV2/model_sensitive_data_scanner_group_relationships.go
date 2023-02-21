// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerGroupRelationships Relationships of the group.
type SensitiveDataScannerGroupRelationships struct {
	// A Sensitive Data Scanner configuration data.
	Configuration *SensitiveDataScannerConfigurationData `json:"configuration,omitempty"`
	// Rules included in the group.
	Rules *SensitiveDataScannerRuleData `json:"rules,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerGroupRelationships instantiates a new SensitiveDataScannerGroupRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerGroupRelationships() *SensitiveDataScannerGroupRelationships {
	this := SensitiveDataScannerGroupRelationships{}
	return &this
}

// NewSensitiveDataScannerGroupRelationshipsWithDefaults instantiates a new SensitiveDataScannerGroupRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerGroupRelationshipsWithDefaults() *SensitiveDataScannerGroupRelationships {
	this := SensitiveDataScannerGroupRelationships{}
	return &this
}

// GetConfiguration returns the Configuration field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupRelationships) GetConfiguration() SensitiveDataScannerConfigurationData {
	if o == nil || o.Configuration == nil {
		var ret SensitiveDataScannerConfigurationData
		return ret
	}
	return *o.Configuration
}

// GetConfigurationOk returns a tuple with the Configuration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupRelationships) GetConfigurationOk() (*SensitiveDataScannerConfigurationData, bool) {
	if o == nil || o.Configuration == nil {
		return nil, false
	}
	return o.Configuration, true
}

// HasConfiguration returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupRelationships) HasConfiguration() bool {
	return o != nil && o.Configuration != nil
}

// SetConfiguration gets a reference to the given SensitiveDataScannerConfigurationData and assigns it to the Configuration field.
func (o *SensitiveDataScannerGroupRelationships) SetConfiguration(v SensitiveDataScannerConfigurationData) {
	o.Configuration = &v
}

// GetRules returns the Rules field value if set, zero value otherwise.
func (o *SensitiveDataScannerGroupRelationships) GetRules() SensitiveDataScannerRuleData {
	if o == nil || o.Rules == nil {
		var ret SensitiveDataScannerRuleData
		return ret
	}
	return *o.Rules
}

// GetRulesOk returns a tuple with the Rules field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerGroupRelationships) GetRulesOk() (*SensitiveDataScannerRuleData, bool) {
	if o == nil || o.Rules == nil {
		return nil, false
	}
	return o.Rules, true
}

// HasRules returns a boolean if a field has been set.
func (o *SensitiveDataScannerGroupRelationships) HasRules() bool {
	return o != nil && o.Rules != nil
}

// SetRules gets a reference to the given SensitiveDataScannerRuleData and assigns it to the Rules field.
func (o *SensitiveDataScannerGroupRelationships) SetRules(v SensitiveDataScannerRuleData) {
	o.Rules = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerGroupRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Configuration != nil {
		toSerialize["configuration"] = o.Configuration
	}
	if o.Rules != nil {
		toSerialize["rules"] = o.Rules
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerGroupRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Configuration *SensitiveDataScannerConfigurationData `json:"configuration,omitempty"`
		Rules         *SensitiveDataScannerRuleData          `json:"rules,omitempty"`
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
	if all.Configuration != nil && all.Configuration.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Configuration = all.Configuration
	if all.Rules != nil && all.Rules.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Rules = all.Rules
	return nil
}
