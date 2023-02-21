// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerConfigurationRelationships Relationships of the configuration.
type SensitiveDataScannerConfigurationRelationships struct {
	// List of groups, ordered.
	Groups *SensitiveDataScannerGroupList `json:"groups,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerConfigurationRelationships instantiates a new SensitiveDataScannerConfigurationRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerConfigurationRelationships() *SensitiveDataScannerConfigurationRelationships {
	this := SensitiveDataScannerConfigurationRelationships{}
	return &this
}

// NewSensitiveDataScannerConfigurationRelationshipsWithDefaults instantiates a new SensitiveDataScannerConfigurationRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerConfigurationRelationshipsWithDefaults() *SensitiveDataScannerConfigurationRelationships {
	this := SensitiveDataScannerConfigurationRelationships{}
	return &this
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *SensitiveDataScannerConfigurationRelationships) GetGroups() SensitiveDataScannerGroupList {
	if o == nil || o.Groups == nil {
		var ret SensitiveDataScannerGroupList
		return ret
	}
	return *o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerConfigurationRelationships) GetGroupsOk() (*SensitiveDataScannerGroupList, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *SensitiveDataScannerConfigurationRelationships) HasGroups() bool {
	return o != nil && o.Groups != nil
}

// SetGroups gets a reference to the given SensitiveDataScannerGroupList and assigns it to the Groups field.
func (o *SensitiveDataScannerConfigurationRelationships) SetGroups(v SensitiveDataScannerGroupList) {
	o.Groups = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerConfigurationRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerConfigurationRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Groups *SensitiveDataScannerGroupList `json:"groups,omitempty"`
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
	if all.Groups != nil && all.Groups.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Groups = all.Groups
	return nil
}
