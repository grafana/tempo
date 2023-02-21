// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerReorderConfig Data related to the reordering of scanning groups.
type SensitiveDataScannerReorderConfig struct {
	// ID of the configuration.
	Id *string `json:"id,omitempty"`
	// Relationships of the configuration.
	Relationships *SensitiveDataScannerConfigurationRelationships `json:"relationships,omitempty"`
	// Sensitive Data Scanner configuration type.
	Type *SensitiveDataScannerConfigurationType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerReorderConfig instantiates a new SensitiveDataScannerReorderConfig object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerReorderConfig() *SensitiveDataScannerReorderConfig {
	this := SensitiveDataScannerReorderConfig{}
	var typeVar SensitiveDataScannerConfigurationType = SENSITIVEDATASCANNERCONFIGURATIONTYPE_SENSITIVE_DATA_SCANNER_CONFIGURATIONS
	this.Type = &typeVar
	return &this
}

// NewSensitiveDataScannerReorderConfigWithDefaults instantiates a new SensitiveDataScannerReorderConfig object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerReorderConfigWithDefaults() *SensitiveDataScannerReorderConfig {
	this := SensitiveDataScannerReorderConfig{}
	var typeVar SensitiveDataScannerConfigurationType = SENSITIVEDATASCANNERCONFIGURATIONTYPE_SENSITIVE_DATA_SCANNER_CONFIGURATIONS
	this.Type = &typeVar
	return &this
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SensitiveDataScannerReorderConfig) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerReorderConfig) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SensitiveDataScannerReorderConfig) HasId() bool {
	return o != nil && o.Id != nil
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *SensitiveDataScannerReorderConfig) SetId(v string) {
	o.Id = &v
}

// GetRelationships returns the Relationships field value if set, zero value otherwise.
func (o *SensitiveDataScannerReorderConfig) GetRelationships() SensitiveDataScannerConfigurationRelationships {
	if o == nil || o.Relationships == nil {
		var ret SensitiveDataScannerConfigurationRelationships
		return ret
	}
	return *o.Relationships
}

// GetRelationshipsOk returns a tuple with the Relationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerReorderConfig) GetRelationshipsOk() (*SensitiveDataScannerConfigurationRelationships, bool) {
	if o == nil || o.Relationships == nil {
		return nil, false
	}
	return o.Relationships, true
}

// HasRelationships returns a boolean if a field has been set.
func (o *SensitiveDataScannerReorderConfig) HasRelationships() bool {
	return o != nil && o.Relationships != nil
}

// SetRelationships gets a reference to the given SensitiveDataScannerConfigurationRelationships and assigns it to the Relationships field.
func (o *SensitiveDataScannerReorderConfig) SetRelationships(v SensitiveDataScannerConfigurationRelationships) {
	o.Relationships = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SensitiveDataScannerReorderConfig) GetType() SensitiveDataScannerConfigurationType {
	if o == nil || o.Type == nil {
		var ret SensitiveDataScannerConfigurationType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerReorderConfig) GetTypeOk() (*SensitiveDataScannerConfigurationType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SensitiveDataScannerReorderConfig) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SensitiveDataScannerConfigurationType and assigns it to the Type field.
func (o *SensitiveDataScannerReorderConfig) SetType(v SensitiveDataScannerConfigurationType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerReorderConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
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
func (o *SensitiveDataScannerReorderConfig) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Id            *string                                         `json:"id,omitempty"`
		Relationships *SensitiveDataScannerConfigurationRelationships `json:"relationships,omitempty"`
		Type          *SensitiveDataScannerConfigurationType          `json:"type,omitempty"`
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
